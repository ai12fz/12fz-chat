package db

import (
	"context"
	"time"

	"github.com/ai12fz/12fz-chat/internal/model"
)

type Agent struct {
	ID           int       `json:"id"`
	BotID        string    `json:"bot_id"`
	DisplayName  string    `json:"display_name"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt"`
	APIKey       string    `json:"api_key"`
	APIURL       string    `json:"api_url"`
	MerchantID   string    `json:"merchant_id"`
	Capabilities []string  `json:"capabilities"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (d *DB) ListAgents(ctx context.Context, merchantID ...string) ([]Agent, error) {
	query := "SELECT id, bot_id, display_name, model, system_prompt, capabilities, status, api_key, api_url, merchant_id, created_at, updated_at FROM chat.agents"
	var args []interface{}
	if len(merchantID) > 0 && merchantID[0] != "" {
		query += " WHERE merchant_id = $1"
		args = append(args, merchantID[0])
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
		if err := rows.Scan(&a.ID, &a.BotID, &a.DisplayName, &a.Model, &a.SystemPrompt, &a.Capabilities, &a.Status, &a.APIKey, &a.APIURL, &a.MerchantID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

func (d *DB) GetAgent(ctx context.Context, botID string) (*Agent, error) {
	var a Agent
	err := d.pool.QueryRow(ctx,
		"SELECT id, bot_id, display_name, model, system_prompt, capabilities, status, api_key, created_at, updated_at FROM chat.agents capabilities, status, api_key, api_url, merchant_id FROM chat.agents WHERE user_id = $1",
		botID,
	).Scan(&a.ID, &a.BotID, &a.DisplayName, &a.Model, &a.SystemPrompt, &a.Capabilities, &a.Status, &a.APIKey, &a.APIURL, &a.MerchantID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (d *DB) CreateAgent(ctx context.Context, a *Agent) error {
	return d.pool.QueryRow(ctx,
		"INSERT INTO chat.agents (bot_id, display_name, model, system_prompt, capabilities, status, api_key) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at, updated_at",
		a.BotID, a.DisplayName, a.Model, a.SystemPrompt, a.Capabilities, a.Status, a.APIKey, a.APIURL, a.MerchantID,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (d *DB) UpdateAgent(ctx context.Context, botID string, a *Agent) error {
	_, err := d.pool.Exec(ctx,
		"UPDATE chat.agents SET display_name=$1, model=$2, system_prompt=$3, capabilities=$4, status=$5, api_key=$7, api_url=$8, merchant_id=$9, updated_at=NOW() WHERE bot_id=$6",
		a.DisplayName, a.Model, a.SystemPrompt, a.Capabilities, a.Status, botID,
	)
	return err
}

func (d *DB) DeleteAgent(ctx context.Context, botID string) error {
	_, err := d.pool.Exec(ctx, "DELETE FROM chat.agents WHERE user_id = $1", botID)
	return err
}

func (d *DB) GetGroupsForBot(ctx context.Context, botID string) ([]model.Group, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT g.id, g.name, g.created_by, g.created_at FROM chat.groups g JOIN chat.group_members gm ON g.id = gm.group_id WHERE gm.user_id = $1", botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []model.Group
	for rows.Next() {
		var g model.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (d *DB) SetBotGroups(ctx context.Context, botID string, groupIDs []int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, "DELETE FROM chat.group_members WHERE user_id = $1", botID)
	if err != nil {
		return err
	}
	for _, gid := range groupIDs {
		_, err = tx.Exec(ctx,
			"INSERT INTO chat.group_members (group_id, user_id, role) VALUES ($1, $2, 'member') ON CONFLICT DO NOTHING",
			gid, botID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ── User ──

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	MerchantID   string `json:"merchant_id"`
	Role         string `json:"role"`
}

func (d *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := d.pool.QueryRow(ctx,
		"SELECT id, username, password_hash, merchant_id, role FROM chat.users WHERE username = $1",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.MerchantID, &u.Role)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

