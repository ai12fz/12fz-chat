package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ai12fz/12fz-chat/internal/db"
	"github.com/ai12fz/12fz-chat/internal/ws"
	"github.com/gorilla/mux"
)

type contextKey string

const (
	contextBotID contextKey = "bot_id"
)

type HTTPHandler struct {
	db          *db.DB
	hub         *ws.Hub
	authHandler *AuthHandler
	startTime   time.Time
	uploadDir   string
}

func NewHTTPHandler(database *db.DB, hub *ws.Hub, auth *AuthHandler, uploadDir string) *HTTPHandler {
	return &HTTPHandler{
		db:          database,
		hub:         hub,
		authHandler: auth,
		startTime:   time.Now(),
		uploadDir:   uploadDir,
	}
}

// AuthMiddleware validates JWT token from Authorization header
func (h *HTTPHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ExtractTokenFromHeader(r)
		if token == "" {
			http.Error(w, `{"error":"missing authorization"}`, 401)
			return
		}
		botID, err := h.authHandler.ValidateToken(token)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, 401)
			return
		}
		ctx := context.WithValue(r.Context(), contextBotID, botID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getBotID extracts bot_id from request context
func getBotID(r *http.Request) string {
	if v, ok := r.Context().Value(contextBotID).(string); ok {
		return v
	}
	return ""
}

// StaticHandler serves the frontend HTML and assets from frontend/dist/
func (h *HTTPHandler) StaticHandler() http.Handler {
	fs := http.FileServer(http.Dir("frontend/dist"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For SPA: serve index.html for all non-API, non-WS routes
		path := r.URL.Path
		if path == "/" || path == "" {
			http.ServeFile(w, r, "frontend/dist/index.html")
			return
		}
		// Serve static files if they exist
		fs.ServeHTTP(w, r)
	})
}

// ── Group ──

func (h *HTTPHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	botID := getBotID(r)
	group, err := h.db.CreateGroup(r.Context(), req.Name, botID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	// Auto-add creator as admin
	if err := h.db.AddMember(r.Context(), group.ID, botID, "admin"); err != nil {
		log.Printf("[http] add creator to group error: %v", err)
	}
	jsonResp(w, group, 201)
}

func (h *HTTPHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	groups, err := h.db.ListGroupsForUser(r.Context(), botID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, groups, 200)
}

func (h *HTTPHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		jsonError(w, "invalid group id", 400)
		return
	}
	members, err := h.db.GetMembers(r.Context(), id)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, members, 200)
}

func (h *HTTPHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		jsonError(w, "invalid group id", 400)
		return
	}
	var req struct {
		BotID string `json:"bot_id"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}
	if err := h.db.AddMember(r.Context(), id, req.BotID, req.Role); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 201)
}

func (h *HTTPHandler) GetMyGroups(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	groups, err := h.db.GetUserGroups(r.Context(), botID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, groups, 200)
}

// ── Message ──

func (h *HTTPHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.ParseInt(r.URL.Query().Get("group_id"), 10, 64)
	if err != nil {
		jsonError(w, "missing group_id", 400)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	msgs, err := h.db.GetMessages(r.Context(), groupID, limit, offset)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, msgs, 200)
}

// POST /api/messages - sends a message and broadcasts via WebSocket
func (h *HTTPHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupID int64  `json:"group_id"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		jsonError(w, "empty content", 400)
		return
	}

	botID := getBotID(r)

	// Save to DB
	msg, err := h.db.CreateAndReturnMessage(r.Context(), req.GroupID, botID, req.Content)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	// Update group's last_msg_at
	_ = h.db.UpdateGroupLastMsg(r.Context(), req.GroupID)

	// Broadcast via WebSocket to all group members
	go h.broadcastMessage(msg)

	jsonResp(w, msg, 201)
}

// broadcastMessage sends a message to all online group members via WS
func (h *HTTPHandler) broadcastMessage(m *db.MessageResult) {
	chatMsg := ws.ChatMessage{
		ID:       m.ID,
		GroupID:  m.GroupID,
		SenderID: m.SenderID,
		Content:  m.Content,
		MsgType:  m.MsgType,
		SendAt:   m.CreatedAt,
	}

	data, err := json.Marshal(ws.WSMessage{
		Type: "message",
		Data: mustJSON(chatMsg),
	})
	if err != nil {
		return
	}

	// Get group members
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	members, err := h.db.GetMembers(ctx, m.GroupID)
	if err != nil {
		return
	}

	var botIDs []string
	for _, member := range members {
		botIDs = append(botIDs, member.BotID)
	}
	h.hub.SendToGroup(m.GroupID, data, botIDs)
}

// ── Unread / Read ──

func (h *HTTPHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	counts, err := h.db.GetUnreadCountForUser(r.Context(), botID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if counts == nil {
		counts = make(map[int64]int)
	}
	jsonResp(w, counts, 200)
}

func (h *HTTPHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupID      int64 `json:"group_id"`
		LastReadMsgID int64 `json:"last_read_msg_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	botID := getBotID(r)
	if err := h.db.UpdateLastRead(r.Context(), req.GroupID, botID, req.LastReadMsgID); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

// ── Friend ──

func (h *HTTPHandler) AddFriend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		FriendID string `json:"friend_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	if err := h.db.AddFriend(r.Context(), req.UserID, req.FriendID); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 201)
}

func (h *HTTPHandler) GetFriends(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["user_id"]
	friends, err := h.db.GetFriends(r.Context(), userID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, friends, 200)
}

// ── DM (Direct Message) Group ──

func (h *HTTPHandler) CreateDMGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FriendID string `json:"friend_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	botID := getBotID(r)
	group, err := h.db.FindOrCreateDMGroup(r.Context(), botID, req.FriendID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, group, 200)
}

// ── Upload ──

// resizeImage scales image down so width ≤ maxWidth, maintaining aspect ratio.
// Uses nearest-neighbor for speed. Re-encodes as JPEG quality 85.
func resizeImage(img image.Image, maxWidth int) ([]byte, error) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= maxWidth {
		// No resize needed, encode as-is
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	newW := maxWidth
	newH := h * maxWidth / w
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := float64(x) * float64(w) / float64(newW)
			srcY := float64(y) * float64(h) / float64(newH)
			xi := int(srcX)
			yi := int(srcY)
			if xi >= w {
				xi = w - 1
			}
			if yi >= h {
				yi = h - 1
			}
			dst.Set(x, y, img.At(xi, yi))
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// POST /api/upload - upload an image, resize if needed, save, create image message
func (h *HTTPHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 10MB (larger than 2MB to allow resizing)
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, "file too large (max 10MB)", 400)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		jsonError(w, "missing image field", 400)
		return
	}
	defer file.Close()

	// Validate MIME type
	buf := make([]byte, 512)
	file.Read(buf)
	file.Seek(0, io.SeekStart)
	mimeType := http.DetectContentType(buf)
	if !strings.HasPrefix(mimeType, "image/") {
		jsonError(w, "only image files allowed", 400)
		return
	}

	// Decode image
	img, _, err := image.Decode(file)
	if err != nil {
		errStr := fmt.Sprintf("decode_err=%v|mime=%s", err, mimeType)
		log.Printf("[upload] %s", errStr)
		jsonResp(w, map[string]string{"error": "invalid image file", "debug": errStr}, 400)
		return
	}

	// Resize if width > 1024 (auto-shrink to 1024)
	imageData, err := resizeImage(img, 1024)
	if err != nil {
		log.Printf("[upload] resize: %v", err)
		jsonError(w, "image processing failed", 500)
		return
	}
	file.Close()

	// Check resized size against 2MB limit
	if len(imageData) > 2<<20 {
		jsonError(w, "image too large even after resize (max 2MB)", 400)
		return
	}

	// Generate unique filename (always .jpg after resize)
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d%s", time.Now().UnixNano(), header.Filename)))
	filename := hex.EncodeToString(hash[:16]) + ".jpg"

	// Ensure upload dir exists
	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		log.Printf("[upload] mkdir: %v", err)
		jsonError(w, "server error", 500)
		return
	}

	// Save resized image
	if err := os.WriteFile(filepath.Join(h.uploadDir, filename), imageData, 0644); err != nil {
		log.Printf("[upload] write: %v", err)
		jsonError(w, "save failed", 500)
		return
	}

	// Get group_id from form
	groupIDStr := r.FormValue("group_id")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		jsonError(w, "missing group_id", 400)
		return
	}

	botID := getBotID(r)
	imageURL := "/uploads/" + filename

	// Create message with msg_type='image'
	msg, err := h.db.CreateAndReturnMessageWithType(r.Context(), groupID, botID, imageURL, "image")
	if err != nil {
		log.Printf("[upload] save msg: %v", err)
		jsonError(w, "save message failed", 500)
		return
	}

	// Update group's last_msg_at
	_ = h.db.UpdateGroupLastMsg(r.Context(), groupID)

	// Broadcast via WebSocket
	go h.broadcastMessage(msg)

	jsonResp(w, msg, 201)
}

// ── Avatar Upload ──

func (h *HTTPHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20) // 5MB limit
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		jsonError(w, "file too large (max 5MB)", 400)
		return
	}

	file, _, err := r.FormFile("avatar")
	if err != nil {
		jsonError(w, "missing avatar field", 400)
		return
	}
	defer file.Close()

	// Validate MIME type
	buf := make([]byte, 512)
	file.Read(buf)
	file.Seek(0, io.SeekStart)
	mimeType := http.DetectContentType(buf)
	if !strings.HasPrefix(mimeType, "image/") {
		jsonError(w, "only image files allowed", 400)
		return
	}

	// Decode image
	img, _, err := image.Decode(file)
	if err != nil {
		jsonError(w, "invalid image file", 400)
		return
	}

	// Resize avatar to max 200px
	avatarData, err := resizeImage(img, 200)
	if err != nil {
		jsonError(w, "image processing failed", 500)
		return
	}

	botID := getBotID(r)
	if botID == "" {
		jsonError(w, "unauthorized", 401)
		return
	}

	// Ensure upload dir exists
	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		log.Printf("[avatar] mkdir: %v", err)
		jsonError(w, "server error", 500)
		return
	}

	// Save avatar: avatar_{botID}.jpg (overwrites on re-upload)
	filename := "avatar_" + botID + ".jpg"
	if err := os.WriteFile(filepath.Join(h.uploadDir, filename), avatarData, 0644); err != nil {
		log.Printf("[avatar] write: %v", err)
		jsonError(w, "save failed", 500)
		return
	}

	avatarURL := "/uploads/" + filename
	jsonResp(w, map[string]string{"avatar_url": avatarURL}, 200)
}

// ── Health ──

func (h *HTTPHandler) Health(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{
		"status":  "ok",
		"service": "12fz-chat",
		"uptime":  time.Since(h.startTime).String(),
	}, 200)
}

// ── Helpers ──

func jsonResp(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	jsonResp(w, map[string]string{"error": msg}, status)
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
