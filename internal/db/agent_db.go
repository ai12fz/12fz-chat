package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"fmt"

	"github.com/ai12fz/12fz-chat/internal/model"
)

type Agent struct {
	ID                   int       `json:"id"`
	SwarmName            string    `json:"swarm_name"`
	BotID                string    `json:"bot_id"`
	DisplayName          string    `json:"display_name"`
	DeviceID             string    `json:"device_id"`
	Model                string    `json:"model"`
	ModelProvider        string    `json:"model_provider"`
	SystemPrompt         string    `json:"system_prompt"`
	APIKey               string    `json:"api_key"`
	APIURL               string    `json:"api_url"`
	MerchantID           string    `json:"merchant_id"`
	Category             string    `json:"category"`
	Capabilities         []string  `json:"capabilities"`
	Status               string    `json:"status"`
	AgentType            string    `json:"agent_type"`
	Token                string    `json:"token"`
	AllowInstallSkills   bool      `json:"allow_install_skills"`
	AllowInstallSoftware bool      `json:"allow_install_software"`
	HeartbeatAt          time.Time `json:"heartbeat_at"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

const agentCols = "id, bot_id, display_name, device_id, model, COALESCE(model_provider,''), system_prompt, category, capabilities, status, api_key, api_url, merchant_id, COALESCE(agent_type,'api'), COALESCE(token,''), COALESCE(swarm_name,''), allow_install_skills, allow_install_software, COALESCE(heartbeat_at, TIMESTAMPTZ 'epoch'), created_at, updated_at"

func (d *DB) ListAgents(ctx context.Context, merchantID ...string) ([]Agent, error) {
	query := "SELECT " + agentCols + " FROM chat.agents"
	var args []interface{}
	var conds []string
	if len(merchantID) > 0 && merchantID[0] != "" {
		conds = append(conds, "merchant_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, merchantID[0])
	}
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY id"
	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.BotID, &a.DisplayName, &a.DeviceID, &a.Model, &a.ModelProvider, &a.SystemPrompt, &a.Category, &a.Capabilities, &a.Status, &a.APIKey, &a.APIURL, &a.MerchantID, &a.AgentType, &a.Token, &a.SwarmName, &a.AllowInstallSkills, &a.AllowInstallSoftware, &a.HeartbeatAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// ListAgentsByDevice returns agents belonging to a device (device_id match, including legacy swarm_name match)
func (d *DB) ListAgentsByDevice(ctx context.Context, deviceID string) ([]Agent, error) {
	query := "SELECT " + agentCols + " FROM chat.agents WHERE device_id = $1 OR (COALESCE(device_id,'')='' AND swarm_name = $1) ORDER BY id"
	rows, err := d.pool.Query(ctx, query, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.BotID, &a.DisplayName, &a.DeviceID, &a.Model, &a.ModelProvider, &a.SystemPrompt, &a.Category, &a.Capabilities, &a.Status, &a.APIKey, &a.APIURL, &a.MerchantID, &a.AgentType, &a.Token, &a.SwarmName, &a.AllowInstallSkills, &a.AllowInstallSoftware, &a.HeartbeatAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

func (d *DB) GetAgent(ctx context.Context, botID string) (*Agent, error) {
	var a Agent
	err := d.pool.QueryRow(ctx,
		"SELECT "+agentCols+" FROM chat.agents WHERE bot_id = $1",
		botID,
	).Scan(&a.ID, &a.BotID, &a.DisplayName, &a.DeviceID, &a.Model, &a.ModelProvider, &a.SystemPrompt, &a.Category, &a.Capabilities, &a.Status, &a.APIKey, &a.APIURL, &a.MerchantID, &a.AgentType, &a.Token, &a.SwarmName, &a.AllowInstallSkills, &a.AllowInstallSoftware, &a.HeartbeatAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (d *DB) CreateAgent(ctx context.Context, a *Agent) error {
	// 自动生成 token
	if a.Token == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("generate token: %w", err)
		}
		a.Token = "sk-" + hex.EncodeToString(b)
	}
	if a.AgentType == "" {
		a.AgentType = "api"
	}
	if a.Model == "" {
		a.Model = "deepseek-v4-flash"
	}
	return d.pool.QueryRow(ctx,
		"INSERT INTO chat.agents (bot_id, display_name, device_id, model, model_provider, system_prompt, category, capabilities, status, api_key, api_url, merchant_id, agent_type, token, allow_install_skills, allow_install_software) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING id, created_at, updated_at",
		a.BotID, a.DisplayName, a.DeviceID, a.Model, a.ModelProvider, a.SystemPrompt, a.Category, a.Capabilities, a.Status, a.APIKey, a.APIURL, a.MerchantID, a.AgentType, a.Token, a.AllowInstallSkills, a.AllowInstallSoftware,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (d *DB) UpdateAgent(ctx context.Context, botID string, a *Agent) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE chat.agents SET display_name=COALESCE(NULLIF($1,''), display_name), model=COALESCE(NULLIF($2,''), model), system_prompt=$3, capabilities=$4, status=COALESCE(NULLIF($5,''), status), api_key=$6, api_url=$7, merchant_id=$8, agent_type=COALESCE(NULLIF($9,''), agent_type), device_id=$10, model_provider=COALESCE(NULLIF($11,''), model_provider), allow_install_skills=$12, allow_install_software=$13, updated_at=NOW() WHERE bot_id=$14`,
		a.DisplayName, a.Model, a.SystemPrompt, a.Capabilities, a.Status, a.APIKey, a.APIURL, a.MerchantID, a.AgentType, a.DeviceID, a.ModelProvider, a.AllowInstallSkills, a.AllowInstallSoftware, botID,
	)
	return err
}

// TouchAgentHeartbeat updates agent heartbeat_at and status
func (d *DB) TouchAgentHeartbeat(ctx context.Context, botID string) error {
	_, err := d.pool.Exec(ctx,
		"UPDATE chat.agents SET heartbeat_at=NOW(), status='active' WHERE bot_id=$1", botID)
	return err
}

// AgentNameExists checks if display_name already exists
func (d *DB) AgentNameExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := d.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM chat.agents WHERE display_name=$1)", name).Scan(&exists)
	return exists, err
}

func (d *DB) DeleteAgent(ctx context.Context, botID string) error {
	_, err := d.pool.Exec(ctx, "DELETE FROM chat.agents WHERE bot_id = $1", botID)
	return err
}

func (d *DB) GetGroupsForBot(ctx context.Context, botID string) ([]model.Group, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT g.id, g.name, g.created_by, g.created_at FROM chat.groups g JOIN chat.group_members gm ON g.id = gm.group_id WHERE gm.user_id = $1", botID)
	if err != nil { return nil, err }
	defer rows.Close()
	var groups []model.Group
	for rows.Next() {
		var g model.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt); err != nil { return nil, err }
		groups = append(groups, g)
	}
	return groups, nil
}

func (d *DB) SetBotGroups(ctx context.Context, botID string, groupIDs []int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, "DELETE FROM chat.group_members WHERE user_id = $1", botID)
	if err != nil { return err }
	for _, gid := range groupIDs {
		_, err = tx.Exec(ctx,
			"INSERT INTO chat.group_members (group_id, user_id, role) VALUES ($1, $2, 'member') ON CONFLICT DO NOTHING",
			gid, botID)
		if err != nil { return err }
	}
	return tx.Commit(ctx)
}
