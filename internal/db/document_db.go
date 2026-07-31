package db

import (
	"context"
	"fmt"
	"time"
)

// Document is a merchant-scoped file produced by an agent/bot and
// downloadable by the merchant's users from the chat UI.
type Document struct {
	ID           int64     `json:"id"`
	MerchantID   string    `json:"merchant_id"`
	BotID        string    `json:"bot_id"`
	UserID       string    `json:"user_id"`
	Title        string    `json:"title"`
	Filename     string    `json:"filename"`
	Size         int64     `json:"size"`
	MIME         string    `json:"mime"`
	StoragePath  string    `json:"-"`
	DownloadCount int64    `json:"download_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// DefaultDocQuota is the number of documents kept per merchant when no
// explicit quota row exists in chat.doc_quotas. Tiered pricing (future)
// will write rows into chat.doc_quotas per merchant/plan.
const DefaultDocQuota = 20

// InsertDocument stores a new document row and returns its id and created_at.
func (d *DB) InsertDocument(ctx context.Context, doc *Document) (int64, time.Time, error) {
	var id int64
	var createdAt time.Time
	err := d.pool.QueryRow(ctx,
		`INSERT INTO chat.documents (merchant_id, bot_id, user_id, title, filename, size, mime, storage_path)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at`,
		doc.MerchantID, doc.BotID, doc.UserID, doc.Title, doc.Filename, doc.Size, doc.MIME, doc.StoragePath,
	).Scan(&id, &createdAt)
	return id, createdAt, err
}

// ListDocuments returns the newest documents for a merchant, newest first.
func (d *DB) ListDocuments(ctx context.Context, merchantID string, limit int) ([]Document, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := d.pool.Query(ctx,
		`SELECT id, merchant_id, bot_id, user_id, title, filename, size, mime, download_count, created_at
		 FROM chat.documents WHERE merchant_id = $1 ORDER BY id DESC LIMIT $2`,
		merchantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	docs := make([]Document, 0, 16)
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.MerchantID, &doc.BotID, &doc.UserID, &doc.Title,
			&doc.Filename, &doc.Size, &doc.MIME, &doc.DownloadCount, &doc.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// GetDocument fetches one document row by id.
func (d *DB) GetDocument(ctx context.Context, id int64) (*Document, error) {
	var doc Document
	err := d.pool.QueryRow(ctx,
		`SELECT id, merchant_id, bot_id, user_id, title, filename, size, mime, storage_path, download_count, created_at
		 FROM chat.documents WHERE id = $1`, id).
		Scan(&doc.ID, &doc.MerchantID, &doc.BotID, &doc.UserID, &doc.Title,
			&doc.Filename, &doc.Size, &doc.MIME, &doc.StoragePath, &doc.DownloadCount, &doc.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// IncrementDownloadCount bumps the download counter for a document.
func (d *DB) IncrementDownloadCount(ctx context.Context, id int64) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE chat.documents SET download_count = download_count + 1 WHERE id = $1`, id)
	return err
}

// DeleteDocument removes a document row by id.
func (d *DB) DeleteDocument(ctx context.Context, id int64) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM chat.documents WHERE id = $1`, id)
	return err
}

// GetDocQuota returns the document retention limit for a merchant.
// Falls back to DefaultDocQuota when no row exists (tiered pricing hook:
// marketplace can upsert chat.doc_quotas to grant bigger limits).
func (d *DB) GetDocQuota(ctx context.Context, merchantID string) (int, error) {
	var limit int
	err := d.pool.QueryRow(ctx,
		`SELECT doc_limit FROM chat.doc_quotas WHERE merchant_id = $1`, merchantID).Scan(&limit)
	if err != nil {
		return DefaultDocQuota, nil // no row → default plan
	}
	if limit <= 0 {
		return DefaultDocQuota, nil
	}
	return limit, nil
}

// TrimDocumentsToQuota deletes the oldest documents of a merchant beyond the
// retention limit. Returns the trimmed rows (id + storage_path) so the caller
// can remove the physical files after the rows are gone.
func (d *DB) TrimDocumentsToQuota(ctx context.Context, merchantID string, keep int) ([]Document, error) {
	if keep <= 0 {
		keep = DefaultDocQuota
	}
	rows, err := d.pool.Query(ctx,
		`SELECT id, storage_path FROM chat.documents WHERE merchant_id = $1
		 AND id NOT IN (SELECT id FROM chat.documents WHERE merchant_id = $1 ORDER BY id DESC LIMIT $2)`,
		merchantID, keep)
	if err != nil {
		return nil, err
	}
	trimmed := make([]Document, 0, 8)
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.StoragePath); err != nil {
			rows.Close()
			return nil, err
		}
		trimmed = append(trimmed, doc)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, doc := range trimmed {
		if _, err := d.pool.Exec(ctx, `DELETE FROM chat.documents WHERE id = $1`, doc.ID); err != nil {
			return trimmed, err
		}
	}
	return trimmed, nil
}

// OrgIDForBotID resolves the merchant (org) for an authenticated identity.
// Order: chat.devices (device tokens / bridge bots) → org_user (JWT users).
func (d *DB) OrgIDForBotID(ctx context.Context, botID string) (string, error) {
	if botID == "" {
		return "", fmt.Errorf("empty identity")
	}
	var orgID string
	// Device (bridge bot / agent host) → org_id from chat.devices.
	err := d.pool.QueryRow(ctx,
		`SELECT org_id FROM chat.devices WHERE id = $1`, botID).Scan(&orgID)
	if err == nil && orgID != "" {
		return orgID, nil
	}
	// User (JWT bot_id == user_id) → org_id from org_user.
	err = d.pool.QueryRow(ctx,
		`SELECT org_id::text FROM org_user WHERE user_id = $1`, botID).Scan(&orgID)
	if err == nil && orgID != "" {
		return orgID, nil
	}
	return "", fmt.Errorf("cannot resolve merchant for identity %q", botID)
}

// UpsertDocQuota sets a merchant's document retention limit
// (tiered-pricing hook: raise the limit when a merchant upgrades).
func (d *DB) UpsertDocQuota(ctx context.Context, merchantID string, limit int) error {
	if limit <= 0 {
		limit = DefaultDocQuota
	}
	_, err := d.pool.Exec(ctx,
		`INSERT INTO chat.doc_quotas (merchant_id, doc_limit, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (merchant_id) DO UPDATE SET doc_limit = $2, updated_at = NOW()`,
		merchantID, limit)
	return err
}
