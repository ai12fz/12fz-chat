package db

import (
	"context"
	"crypto/rand"
	"os/exec"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

type Device struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OrgID     string    `json:"org_id"`
	Token     string    `json:"token"`
	OS        string    `json:"os"`
	Status    string    `json:"status"`
	LocalIP   string    `json:"local_ip"`
	LastSeen  time.Time `json:"last_seen"`
	AllowSkills   bool `json:"allow_install_skills"`
	AllowSoftware bool `json:"allow_install_software"`
	CreatedAt time.Time  `json:"created_at"`
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
		"UPDATE chat.devices SET last_seen=NOW() WHERE token=$1 RETURNING id, name, org_id, token, os, status, COALESCE(local_ip,''), last_seen, created_at",
		token,
	).Scan(&dev.ID, &dev.Name, &dev.OrgID, &dev.Token, &dev.OS, &dev.Status, &dev.LocalIP, &dev.LastSeen, &dev.CreatedAt)
	return &dev, err
}

func (d *DB) ListDevicesByOrg(ctx context.Context, orgID string) ([]Device, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT id, name, org_id, token, os, status, last_seen, created_at, COALESCE(allow_install_skills,true), COALESCE(allow_install_software,false), COALESCE(local_ip,'') FROM chat.devices WHERE org_id=$1 ORDER BY last_seen DESC", orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var devs []Device
	for rows.Next() {
		var dev Device
		if err := rows.Scan(&dev.ID, &dev.Name, &dev.OrgID, &dev.Token, &dev.OS, &dev.Status, &dev.LastSeen, &dev.CreatedAt, &dev.AllowSkills, &dev.AllowSoftware, &dev.LocalIP); err != nil {
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


// autoFriendDevice adds the device as a friend for all users in the org
func (db *DB) autoFriendDevice(ctx context.Context, deviceID, deviceName, orgID string) {
	rows, err := db.pool.Query(ctx,
		"SELECT user_id FROM org_user WHERE org_id=$1", orgID)
	if err != nil { return }
	defer rows.Close()
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil { continue }
		db.pool.Exec(ctx,
			"INSERT INTO chat.friends (user_id, friend_id, user_type) VALUES ($1, $2, $3) ON CONFLICT (user_id, friend_id) DO NOTHING",
			strconv.FormatInt(uid, 10), deviceID, "device")
	}
}

func (db *DB) StoreAPIKey(ctx context.Context, key, token string) error {
	// Insert into new_api.tokens via direct shell command
	cmd := exec.Command("psql", "-U", "new_api", "-d", "new_api", "-h", "localhost", "-c",
		fmt.Sprintf("INSERT INTO tokens (key, name, user_id, status, remain_quota, unlimited_quota) VALUES ('%s', '%s', 1, 1, 100000, true) ON CONFLICT (key) DO NOTHING", key, "device-"+token[:8]))
	return cmd.Run()
}
// SetDeviceOnline updates device status to online in DB
func (db *DB) SetDeviceOnline(ctx context.Context, deviceID string) error {
	_, err := db.pool.Exec(ctx, "UPDATE chat.devices SET status='online', last_seen=NOW() WHERE id=$1", deviceID)
	return err
}

// SetDeviceOffline updates device status to offline in DB
func (db *DB) SetDeviceOffline(ctx context.Context, deviceID string) error {
	_, err := db.pool.Exec(ctx, "UPDATE chat.devices SET status='offline' WHERE id=$1", deviceID)
	return err
}

// UpdateDeviceLastSeen updates device online timestamp (heartbeat only, does NOT change status)
func (db *DB) UpdateDeviceLastSeen(ctx context.Context, deviceID string) error {
	_, err := db.pool.Exec(ctx, "UPDATE chat.devices SET last_seen=NOW() WHERE id=$1", deviceID)
	return err
}

// ResetDeviceStatus resets all device statuses to offline (called at server startup)
func (db *DB) ResetDeviceStatus(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, "UPDATE chat.devices SET status='offline'")
	return err
}

func (d *DB) LogDeviceActivity(ctx context.Context, deviceID, action, detail string) error {
	_, err := d.pool.Exec(ctx, `INSERT INTO chat.device_activity (device_id, action, detail) VALUES ($1, $2, $3)`, deviceID, action, detail)
	return err
}

func (d *DB) GetDeviceActivity(ctx context.Context, deviceID string, limit int) ([]map[string]interface{}, error) {
	rows, err := d.pool.Query(ctx, `SELECT action, detail, created_at FROM chat.device_activity WHERE device_id=$1 ORDER BY created_at DESC LIMIT $2`, deviceID, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var action, detail string
		var ts time.Time
		rows.Scan(&action, &detail, &ts)
		out = append(out, map[string]interface{}{"action": action, "detail": detail, "time": ts.Format("15:04:05")})
	}
	return out, nil
}

func (d *DB) CleanupOldMessages(ctx context.Context) (int64, error) {
	tag, err := d.pool.Exec(ctx, `DELETE FROM chat.friend_messages WHERE id NOT IN (
		SELECT id FROM (
			SELECT id, row_number() OVER (PARTITION BY LEAST(from_id, to_id), GREATEST(from_id, to_id) ORDER BY created_at DESC) as rn
			FROM chat.friend_messages
		) sub WHERE rn <= 50
	)`)
	if err != nil { return 0, err }
	return tag.RowsAffected(), nil
}

// SetDeviceLocalIP updates the local_ip for a device
func (db *DB) SetDeviceLocalIP(ctx context.Context, deviceID, ip string) error {
	_, err := db.pool.Exec(ctx, "UPDATE chat.devices SET local_ip=$2 WHERE id=$1", deviceID, ip)
	return err
}
