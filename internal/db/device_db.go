package db

import (
	"context"
	"crypto/rand"
	"fmt"
	"encoding/hex"
	"time"
)

type Device struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OrgID     string    `json:"org_id"`
	Token     string    `json:"token"`
	OS        string    `json:"os"`
	Status    string    `json:"status"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
}

func (d *DB) RegisterDevice(ctx context.Context, name, deviceKey, os string) (*Device, error) {
	var orgID string
	err := d.pool.QueryRow(ctx,
		"UPDATE chat.device_reg_codes SET status='used', used_at=now() WHERE code=$1 AND status='active' RETURNING org_id::text",
		deviceKey).Scan(&orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid or used registration code")
	}
	token := generateToken()
	dev := &Device{Name: name, OrgID: orgID, Token: token, OS: os, Status: "online"}
	err = d.pool.QueryRow(ctx,
		"INSERT INTO chat.devices (id, name, org_id, token, os) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET name=$2, token=$4, os=$5, last_seen=NOW(), status='online' RETURNING id, created_at",
		name, name, orgID, token, os).Scan(&dev.ID, &dev.CreatedAt)
	if err != nil {
		return nil, err
	}
	d.pool.Exec(ctx, "UPDATE chat.device_reg_codes SET device_id=$1 WHERE code=$2", dev.ID, deviceKey)
	return dev, nil
}

func (d *DB) ValidateDeviceToken(ctx context.Context, token string) (*Device, error) {
	var dev Device
	err := d.pool.QueryRow(ctx,
		"UPDATE chat.devices SET last_seen=NOW(), status='online' WHERE token=$1 RETURNING id, name, org_id, token, os, status, last_seen, created_at",
		token,
	).Scan(&dev.ID, &dev.Name, &dev.OrgID, &dev.Token, &dev.OS, &dev.Status, &dev.LastSeen, &dev.CreatedAt)
	return &dev, err
}

func (d *DB) ListDevicesByOrg(ctx context.Context, orgID string) ([]Device, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT id, name, org_id, token, os, status, last_seen, created_at FROM chat.devices WHERE org_id=$1 ORDER BY last_seen DESC", orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var devs []Device
	for rows.Next() {
		var dev Device
		if err := rows.Scan(&dev.ID, &dev.Name, &dev.OrgID, &dev.Token, &dev.OS, &dev.Status, &dev.LastSeen, &dev.CreatedAt); err != nil {
			return nil, err
		}
		devs = append(devs, dev)
	}
	return devs, nil
}

func (d *DB) DeleteDevice(ctx context.Context, id string) error {
	_, err := d.pool.Exec(ctx, "DELETE FROM chat.devices WHERE id=$1", id)
	return err
}

func (d *DB) PendingAgentsByOrg(ctx context.Context, orgID string) ([]Agent, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT id, bot_id, display_name, model, system_prompt, capabilities, status, COALESCE(api_key,''), COALESCE(api_url,''), COALESCE(merchant_id,'') FROM chat.agents WHERE merchant_id=$1 AND status='active' ORDER BY id",
		orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.BotID, &a.DisplayName, &a.Model, &a.SystemPrompt, &a.Capabilities, &a.Status, &a.APIKey, &a.APIURL, &a.MerchantID); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

func generateToken() string {
	b := make([]byte, 20)
	rand.Read(b)
	return "d_" + hex.EncodeToString(b)
}

func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	rb := make([]byte, n)
	rand.Read(rb)
	for i := range b {
		b[i] = chars[rb[i]%36]
	}
	return string(b)
}

func (db *DB) GenerateRegCode(ctx context.Context, orgID string, createdBy string) (string, error) {
	code := "dev-" + randomString(12)
	var exists bool
	err := db.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM chat.device_reg_codes WHERE code=$1)", code).Scan(&exists)
	if err != nil {
		return "", err
	}
	if exists {
		return db.GenerateRegCode(ctx, orgID, createdBy)
	}
	_, err = db.pool.Exec(ctx, "INSERT INTO chat.device_reg_codes (code, org_id, created_by) VALUES ($1,$2,$3)", code, orgID, createdBy)
	return code, err
}

func (db *DB) ListRegCodes(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	rows, err := db.pool.Query(ctx, "SELECT code, status, device_id, created_at, used_at FROM chat.device_reg_codes WHERE org_id=$1 AND status!=$2 ORDER BY created_at DESC", orgID, "revoked")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var codes []map[string]interface{}
	for rows.Next() {
		var code, status string
		var deviceID *string
		var createdAt, usedAt *time.Time
		if err := rows.Scan(&code, &status, &deviceID, &createdAt, &usedAt); err != nil {
			continue
		}
		codes = append(codes, map[string]interface{}{
			"code": code, "status": status, "created_at": createdAt, "used_at": usedAt,
		})
		if deviceID != nil {
			codes[len(codes)-1]["device_id"] = *deviceID
		}
	}
	return codes, nil
}

func (db *DB) RevokeRegCode(ctx context.Context, orgID, code string) error {
	_, err := db.pool.Exec(ctx, "UPDATE chat.device_reg_codes SET status=$1 WHERE org_id=$2 AND code=$3 AND status=$4", "revoked", orgID, code, "active")
	return err
}
