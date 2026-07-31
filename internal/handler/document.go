package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ai12fz/12fz-chat/internal/db"
	"github.com/gorilla/mux"
)

// maxDocUploadBytes caps a single document upload (64 MB).
const maxDocUploadBytes = 64 << 20

// UploadDocument accepts a multipart POST (field "file", optional "title" and
// "user_id"), stores the file under the merchant's docs dir, records a row in
// chat.documents and trims the merchant's retention to its quota.
func (h *HTTPHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	merchantID, err := h.db.OrgIDForBotID(r.Context(), botID)
	if err != nil {
		jsonError(w, err.Error(), 403)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxDocUploadBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "invalid multipart form: "+err.Error(), 400)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "missing file field", 400)
		return
	}
	defer file.Close()

	filename := filepath.Base(header.Filename)
	if filename == "" || filename == "." {
		filename = fmt.Sprintf("doc-%d", time.Now().UnixNano())
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = filename
	}
	userID := strings.TrimSpace(r.FormValue("user_id"))

	if err := os.MkdirAll(h.docsDir, 0o755); err != nil {
		jsonError(w, "storage unavailable: "+err.Error(), 500)
		return
	}
	storageName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomHex(8), filepath.Ext(filename))
	dst, err := os.Create(filepath.Join(h.docsDir, storageName))
	if err != nil {
		jsonError(w, "cannot create file: "+err.Error(), 500)
		return
	}
	size, copyErr := io.Copy(dst, file)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(filepath.Join(h.docsDir, storageName))
		jsonError(w, "cannot store file", 500)
		return
	}

	doc := &db.Document{
		MerchantID:  merchantID,
		BotID:       botID,
		UserID:      userID,
		Title:       title,
		Filename:    filename,
		Size:        size,
		MIME:        header.Header.Get("Content-Type"),
		StoragePath: storageName,
	}
	id, createdAt, err := h.db.InsertDocument(r.Context(), doc)
	if err != nil {
		os.Remove(filepath.Join(h.docsDir, storageName))
		jsonError(w, "db insert failed: "+err.Error(), 500)
		return
	}
	doc.ID = id
	doc.CreatedAt = createdAt

	// Retention: keep the newest N per merchant, drop the rest (row + file).
	if quota, qErr := h.db.GetDocQuota(r.Context(), merchantID); qErr == nil {
		trimmed, tErr := h.db.TrimDocumentsToQuota(r.Context(), merchantID, quota)
		if tErr != nil {
			log.Printf("[docs] trim error for %s: %v", merchantID, tErr)
		}
		for _, old := range trimmed {
			os.Remove(filepath.Join(h.docsDir, old.StoragePath))
		}
	}

	log.Printf("[docs] upload id=%d bot=%s merchant=%s file=%s size=%d", id, botID, merchantID, filename, size)
	jsonResp(w, doc, 201)
}

// ListDocuments returns the merchant's newest documents (newest first).
func (h *HTTPHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	merchantID, err := h.db.OrgIDForBotID(r.Context(), botID)
	if err != nil {
		jsonError(w, err.Error(), 403)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	docs, err := h.db.ListDocuments(r.Context(), merchantID, limit)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if docs == nil {
		docs = make([]db.Document, 0)
	}
	jsonResp(w, docs, 200)
}

// DownloadDocument streams a merchant's document with a download-friendly
// Content-Disposition (RFC 5987 for non-ASCII filenames).
func (h *HTTPHandler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		jsonError(w, "invalid document id", 400)
		return
	}
	doc, err := h.db.GetDocument(r.Context(), id)
	if err != nil {
		jsonError(w, "document not found", 404)
		return
	}
	merchantID, err := h.db.OrgIDForBotID(r.Context(), botID)
	if err != nil || doc.MerchantID != merchantID {
		jsonError(w, "forbidden", 403)
		return
	}
	path := filepath.Join(h.docsDir, doc.StoragePath)
	if _, err := os.Stat(path); err != nil {
		jsonError(w, "file missing on disk", 404)
		return
	}

	h.db.IncrementDownloadCount(r.Context(), doc.ID)

	contentType := doc.MIME
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(doc.Filename))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	asciiName := sanitizeASCII(doc.Filename)
	disposition := fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s",
		asciiName, url.PathEscape(doc.Filename))
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(doc.Size, 10))
	http.ServeFile(w, r, path)
}

// PreviewDocument streams a merchant's document inline so it can be viewed in
// the browser (PDF/images/text) instead of downloaded. Supports Range requests
// via http.ServeContent so the browser's native PDF viewer works.
func (h *HTTPHandler) PreviewDocument(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		jsonError(w, "invalid document id", 400)
		return
	}
	doc, err := h.db.GetDocument(r.Context(), id)
	if err != nil {
		jsonError(w, "document not found", 404)
		return
	}
	merchantID, err := h.db.OrgIDForBotID(r.Context(), botID)
	if err != nil || doc.MerchantID != merchantID {
		jsonError(w, "forbidden", 403)
		return
	}
	path := filepath.Join(h.docsDir, doc.StoragePath)
	f, err := os.Open(path)
	if err != nil {
		jsonError(w, "file missing on disk", 404)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		jsonError(w, "cannot stat file", 500)
		return
	}

	contentType := doc.MIME
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = mime.TypeByExtension(filepath.Ext(doc.Filename))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	asciiName := sanitizeASCII(doc.Filename)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("inline; filename=\"%s\"; filename*=UTF-8''%s", asciiName, url.PathEscape(doc.Filename)))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, doc.Filename, st.ModTime(), f)
}

// randomHex returns n random bytes as lowercase hex.
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// sanitizeASCII keeps only printable ASCII for the legacy filename= token.
func sanitizeASCII(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 32 && r <= 126 {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "download"
	}
	return b.String()
}

// ── Admin: document quota (tiered-pricing hook) ──

// AdminDocQuota GET/PUT allow the platform admin (user id "1") to view and
// change a merchant's document retention limit.
func (h *HTTPHandler) AdminDocQuota(w http.ResponseWriter, r *http.Request) {
	if getBotID(r) != "1" {
		jsonError(w, "forbidden", 403)
		return
	}
	switch r.Method {
	case http.MethodGet:
		merchantID := strings.TrimSpace(r.URL.Query().Get("merchant_id"))
		if merchantID == "" {
			jsonError(w, "missing merchant_id", 400)
			return
		}
		limit, _ := h.db.GetDocQuota(r.Context(), merchantID)
		jsonResp(w, map[string]interface{}{"merchant_id": merchantID, "doc_limit": limit}, 200)
	case http.MethodPut:
		var req struct {
			MerchantID string `json:"merchant_id"`
			DocLimit   int    `json:"doc_limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MerchantID == "" {
			jsonError(w, "bad request", 400)
			return
		}
		if req.DocLimit < 1 || req.DocLimit > 1000 {
			jsonError(w, "doc_limit must be 1..1000", 400)
			return
		}
		if err := h.db.UpsertDocQuota(r.Context(), req.MerchantID, req.DocLimit); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]interface{}{"merchant_id": req.MerchantID, "doc_limit": req.DocLimit}, 200)
	default:
		jsonError(w, "method not allowed", 405)
	}
}
