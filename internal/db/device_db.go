package db

import (
	"context"
	"crypto/rand"
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
	err := d.platformPool.QueryRow(ctx,
		"SELECT org_id::text FROM org_merchant WHERE device_key = $1 AND device_key IS NOT NULL", deviceKey).Scan(&orgID)
	if err != nil {
		return nil, err
	}
	token := generateToken()
	dev := &Device{Name: name, OrgID: orgID, Token: token, OS: os, Status: "online"}
	err = d.pool.QueryRow(ctx,
		"INSERT INTO chat.devices (id, name, org_id, token, os) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET name=$2, token=$4, os=$5, last_seen=NOW(), status='online' RETURNING id, created_at",
		name, name, orgID, token, os).Scan(&dev.ID, &dev.CreatedAt)
	if err != nil {
		return nil, err
	}
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

func (d *DB) GetDeviceKeyByOrg(ctx context.Context, orgID string) (string, error) {
	var key string
	err := d.platformPool.QueryRow(ctx,
		"SELECT COALESCE(device_key, '') FROM org_merchant WHERE org_id=$1", orgID).Scan(&key)
	return key, err
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
