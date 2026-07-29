package db

import (
	"context"
	"fmt"
	"time"

	"github.com/ai12fz/12fz-chat/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
	platformPool *pgxpool.Pool
}

func Connect(cfg interface{ PGConnString() string }) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.PGConnString())
	if err != nil {
		return nil, fmt.Errorf("pg connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pg ping: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Pool() *pgxpool.Pool { return d.pool }

func ConnectBoth(chatDSN, platformDSN string) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, chatDSN)
	if err != nil { return nil, fmt.Errorf("chat db: %w", err) }
	if err := pool.Ping(ctx); err != nil { return nil, fmt.Errorf("chat ping: %w", err) }

	platformPool, err := pgxpool.New(ctx, platformDSN)
	if err != nil { pool.Close(); return nil, fmt.Errorf("zt db: %w", err) }
	if err := platformPool.Ping(ctx); err != nil { pool.Close(); platformPool.Close(); return nil, fmt.Errorf("zt ping: %w", err) }

	return &DB{pool: pool, platformPool: platformPool}, nil
}



func NewFromPool(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

func (d *DB) Close() {
	d.pool.Close()
}


type OrgUser struct {
	UserID   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

func (d *DB) GetOrgUserByID(ctx context.Context, userID int64) (*OrgUser, error) {

	var u OrgUser
	err := d.platformPool.QueryRow(ctx,
		"SELECT user_id, nickname, phone, COALESCE(email, ''), status FROM org_user WHERE user_id = $1",
		userID,
	).Scan(&u.UserID, &u.Nickname, &u.Phone, &u.Email, &u.Status)
	return &u, err
}

func (d *DB) GetOrgID(ctx context.Context, userID int64) (string, error) {
	var orgID string
	err := d.platformPool.QueryRow(ctx,
		"SELECT org_id::text FROM org_user WHERE user_id = $1", userID).Scan(&orgID)
	return orgID, err
}

func (d *DB) GetOrgUserForLogin(ctx context.Context, account, password string) (*OrgUser, error) {
	var u OrgUser
	err := d.platformPool.QueryRow(ctx,
		"SELECT user_id, COALESCE(nickname, '') FROM org_user WHERE (nickname = $1 OR phone = $1) AND password = $2",
		account, password,
	).Scan(&u.UserID, &u.Nickname)
	return &u, err
}

func (d *DB) AutoMigrate(ctx context.Context) error {
	stmts := []string{
		"CREATE SCHEMA IF NOT EXISTS chat",

		"CREATE TABLE IF NOT EXISTS chat.groups (" +
			"id SERIAL PRIMARY KEY," +
			"name TEXT NOT NULL," +
			"created_by TEXT NOT NULL," +
			"created_at TIMESTAMPTZ DEFAULT NOW()," +
			"last_msg_at TIMESTAMPTZ DEFAULT NOW()" +
			")",

		"CREATE TABLE IF NOT EXISTS chat.group_members (" +
			"group_id INT REFERENCES chat.groups(id) ON DELETE CASCADE," +
			"user_id BIGINT NOT NULL," +
			"role TEXT DEFAULT 'member'," +
			"joined_at TIMESTAMPTZ DEFAULT NOW()," +
			"last_read_msg_id INT DEFAULT 0," +
			"PRIMARY KEY (group_id, user_id)" +
			")",

		"CREATE TABLE IF NOT EXISTS chat.messages (" +
			"id SERIAL PRIMARY KEY," +
			"group_id INT NOT NULL REFERENCES chat.groups(id) ON DELETE CASCADE," +
			"sender_id TEXT NOT NULL," +
			"content TEXT NOT NULL DEFAULT ''," +
			"msg_type TEXT DEFAULT 'text'," +
			"created_at TIMESTAMPTZ DEFAULT NOW()" +
			")",

		"CREATE INDEX IF NOT EXISTS idx_messages_group_id ON chat.messages(group_id)",
		"CREATE INDEX IF NOT EXISTS idx_messages_created_at ON chat.messages(created_at)",

		"CREATE TABLE IF NOT EXISTS chat.friends (" +
			"user_id TEXT NOT NULL," +
			"friend_id TEXT NOT NULL," +
			"status TEXT DEFAULT 'pending'," +
			"created_at TIMESTAMPTZ DEFAULT NOW()," +
			"PRIMARY KEY (user_id, friend_id)" +
			")",

		// Add last_msg_at column if it doesn't exist (for existing databases)
		"DO $$ BEGIN " +
			"ALTER TABLE chat.groups ADD COLUMN IF NOT EXISTS last_msg_at TIMESTAMPTZ DEFAULT NOW(); " +
		"EXCEPTION WHEN duplicate_column THEN NULL; END $$",

		// Add last_read_msg_id column if it doesn't exist
		"DO $$ BEGIN " +
			"ALTER TABLE chat.group_members ADD COLUMN IF NOT EXISTS last_read_msg_id INT DEFAULT 0; " +
		"EXCEPTION WHEN duplicate_column THEN NULL; END $$",
	}
	for _, s := range stmts {
		if _, err := d.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("migrate: %w\nSQL: %s", err, s)
		}
	}
	return nil
}

// MessageResult is the return type for CreateAndReturnMessage
type MessageResult struct {
	ID        int64     `json:"id"`
	GroupID   int64     `json:"group_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	MsgType   string    `json:"msg_type"`
	CreatedAt time.Time `json:"created_at"`
}

// ── Message ──

func (d *DB) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := d.pool.Exec(ctx, sql, args...)
	return err
}

func (d *DB) SaveMessage(ctx context.Context, m *model.Message) error {
	err := d.pool.QueryRow(ctx,
		"INSERT INTO chat.messages (group_id, sender_id, content, msg_type) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		m.GroupID, m.SenderID, m.Content, m.MsgType,
	).Scan(&m.ID, &m.CreatedAt)
	return err
}

func (d *DB) GetMessages(ctx context.Context, groupID int64, limit, offset int) ([]model.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := d.pool.Query(ctx,
		"SELECT id, group_id, sender_id, content, msg_type, created_at FROM chat.messages WHERE group_id = $1 ORDER BY id DESC LIMIT $2 OFFSET $3",
		groupID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []model.Message
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.GroupID, &m.SenderID, &m.Content, &m.MsgType, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	// Reverse to chronological
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// ── Group ──

type GroupWithMeta struct {
	model.Group
	LastMsgAt     time.Time `json:"last_msg_at"`
	LastReadMsgID int64     `json:"last_read_msg_id"`
	Unread        int       `json:"unread"`
}

func (d *DB) CreateGroup(ctx context.Context, name string, createdBy int64) (*model.Group, error) {
	g := &model.Group{Name: name, CreatedBy: createdBy}
	err := d.pool.QueryRow(ctx,
		"INSERT INTO chat.groups (name, created_by) VALUES ($1, $2) RETURNING id, created_at",
		name, createdBy,
	).Scan(&g.ID, &g.CreatedAt)
	return g, err
}

func (d *DB) ListGroups(ctx context.Context) ([]GroupWithMeta, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT id, name, created_by, created_at, last_msg_at FROM chat.groups ORDER BY last_msg_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []GroupWithMeta
	for rows.Next() {
		var g GroupWithMeta
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt, &g.LastMsgAt, &g.LastReadMsgID, &g.Unread); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// ListGroupsForUser returns groups the user is a member of, sorted by last_msg_at DESC
func (d *DB) ListGroupsForUser(ctx context.Context, botID string) ([]GroupWithMeta, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT g.id, g.name, g.created_by, g.created_at, g.last_msg_at,
		       m.last_read_msg_id,
		       (SELECT COUNT(*) FROM chat.messages WHERE group_id = g.id AND id > COALESCE(m.last_read_msg_id, 0)) as unread
		 FROM chat.groups g
		 JOIN chat.group_members m ON m.group_id = g.id
		 WHERE m.user_id = $1
		 ORDER BY g.last_msg_at DESC`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []GroupWithMeta
	for rows.Next() {
		var g GroupWithMeta
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt, &g.LastMsgAt, &g.LastReadMsgID, &g.Unread); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (d *DB) AddMember(ctx context.Context, groupID int64, botID, role string) error {
	_, err := d.pool.Exec(ctx,
		"INSERT INTO chat.group_members (group_id, user_id, role) VALUES ($1, $2, $3) ON CONFLICT (group_id, user_id) DO UPDATE SET role = $3",
		groupID, botID, role)
	return err
}

func (d *DB) GetMembers(ctx context.Context, groupID int64) ([]model.GroupMember, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT group_id, user_id, role, joined_at FROM chat.group_members WHERE group_id = $1",
		groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []model.GroupMember
	for rows.Next() {
		var m model.GroupMember
		if err := rows.Scan(&m.GroupID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func (d *DB) GetUserGroups(ctx context.Context, botID string) ([]GroupWithMeta, error) {
	return d.ListGroupsForUser(ctx, botID)
}

// UpdateGroupLastMsg updates the last_msg_at timestamp for a group
func (d *DB) UpdateGroupLastMsg(ctx context.Context, groupID int64) error {
	_, err := d.pool.Exec(ctx,
		"UPDATE chat.groups SET last_msg_at = NOW() WHERE id = $1", groupID)
	return err
}

// UpdateLastRead updates the last_read_msg_id for a member in a group
func (d *DB) UpdateLastRead(ctx context.Context, groupID int64, botID string, msgID int64) error {
	_, err := d.pool.Exec(ctx,
		"UPDATE chat.group_members SET last_read_msg_id = $1 WHERE group_id = $2 AND user_id = $3",
		msgID, groupID, botID)
	return err
}

// GetUnreadCount returns the number of unread messages for a member in a group
func (d *DB) GetUnreadCount(ctx context.Context, groupID int64, botID string) (int, error) {
	var count int
	err := d.pool.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*), 0) FROM chat.messages m
		 WHERE m.group_id = $1 AND m.id > (
		   SELECT COALESCE(gm.last_read_msg_id, 0) FROM chat.group_members gm
		   WHERE gm.group_id = $1 AND gm.user_id = $2
		 )`,
		groupID, botID).Scan(&count)
	return count, err
}

// GetUnreadCountForUser returns unread count for all user's groups
func (d *DB) GetUnreadCountForUser(ctx context.Context, botID string) (map[int64]int, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT m.group_id, COUNT(*) AS unread
		 FROM chat.messages m
		 JOIN chat.group_members gm ON gm.group_id = m.group_id AND gm.user_id = $1
		 WHERE m.id > COALESCE(gm.last_read_msg_id, 0)
		 GROUP BY m.group_id`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[int64]int)
	for rows.Next() {
		var groupID int64
		var count int
		if err := rows.Scan(&groupID, &count); err != nil {
			return nil, err
		}
		result[groupID] = count
	}
	return result, nil
}

// ── Friend ──

func (d *DB) GetOrgAdminID(ctx context.Context, orgID string) (string, error) {
	var userID string
	err := d.platformPool.QueryRow(ctx,
		"SELECT user_id::text FROM org_user WHERE org_id = $1 AND is_owner = true LIMIT 1", orgID).Scan(&userID)
	return userID, err
}

func (d *DB) AutoFriendDevice(ctx context.Context, userID, deviceName string) error {
	_, err := d.pool.Exec(ctx,
		"INSERT INTO chat.friends (user_id, friend_id, status, user_type) VALUES ($1, $2, 'accepted', 'device') ON CONFLICT (user_id, friend_id) DO UPDATE SET status='accepted'",
		userID, deviceName)
	return err
}

func (d *DB) AddFriend(ctx context.Context, userID, friendID, userType string) error {
	_, err := d.pool.Exec(ctx,
		"INSERT INTO chat.friends (user_id, friend_id, status, user_type) VALUES ($1, $2, 'accepted', $3) ON CONFLICT (user_id, friend_id) DO UPDATE SET user_type = $3",
		userID, friendID, userType)
	return err
}

func (d *DB) GetFriends(ctx context.Context, userID string) ([]model.Friend, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT f.user_id, f.friend_id, f.status, COALESCE(f.user_type, CASE WHEN a.id IS NOT NULL THEN 'agent' ELSE 'human' END) as user_type, COALESCE(f.category, E'日常') as category, f.created_at, COALESCE(a.display_name, d.name, f.friend_id) as name FROM chat.friends f LEFT JOIN chat.agents a ON f.friend_id = a.bot_id LEFT JOIN chat.devices d ON f.friend_id = d.id WHERE f.user_id = $1",
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var friends []model.Friend
	for rows.Next() {
		var f model.Friend
		if err := rows.Scan(&f.UserID, &f.FriendID, &f.Status, &f.UserType, &f.Category, &f.CreatedAt, &f.Name, &f.Name); err != nil {
			return nil, err
		}
		friends = append(friends, f)
	}
	return friends, nil
}

func (d *DB) CreateAndReturnMessage(ctx context.Context, groupID int64, senderID, content string) (*MessageResult, error) {
	m := &MessageResult{
		GroupID:  groupID,
		SenderID: senderID,
		Content:  content,
		MsgType:  "text",
	}
	err := d.pool.QueryRow(ctx,
		`INSERT INTO chat.messages (group_id, sender_id, content) VALUES ($1, $2, $3) RETURNING id, created_at`,
		groupID, senderID, content,
	).Scan(&m.ID, &m.CreatedAt)
	return m, err
}


func (d *DB) SaveFriendMessage(ctx context.Context, fromID, toID, content string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx,
		"INSERT INTO chat.friend_messages (from_id, to_id, content) VALUES ($1, $2, $3) RETURNING id",
		fromID, toID, content).Scan(&id)
	return id, err
}

func (d *DB) GetFriendMessages(ctx context.Context, userID, otherID string, limit, offset int) ([]model.FriendMessage, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT id, from_id, to_id, content, created_at FROM chat.friend_messages
		 WHERE (from_id=$1 AND to_id=$2) OR (from_id=$2 AND to_id=$1)
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		userID, otherID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []model.FriendMessage
	for rows.Next() {
		var m model.FriendMessage
		if err := rows.Scan(&m.ID, &m.FromID, &m.ToID, &m.Content, &m.CreatedAt); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}


func (d *DB) UpdateFriendCategory(ctx context.Context, userID, friendID, category string) error {
	_, err := d.pool.Exec(ctx,
		"UPDATE chat.friends SET category =  WHERE user_id =  AND friend_id = ",
		category, userID, friendID)
	return err
}
func (d *DB) UpdateFriendStatus(ctx context.Context, userID, friendID, status string) error {
	_, err := d.pool.Exec(ctx,
		"UPDATE chat.friends SET status=$1 WHERE user_id=$2 AND friend_id=$3",
		status, userID, friendID)
	return err
}

func (d *DB) DeleteFriend(ctx context.Context, userID, friendID string) error {
	_, err := d.pool.Exec(ctx,
		"DELETE FROM chat.friends WHERE user_id=$1 AND friend_id=$2",
		userID, friendID)
	return err
}


// GetBotStatus returns the current status of a bot/agent
func (d *DB) UpdateBotStatus(botID, status string) {
	if d != nil && d.pool != nil {
		d.pool.Exec(context.Background(),
			"INSERT INTO chat.bot_statuses (bot_id, status, updated_at) VALUES (, , NOW()) ON CONFLICT (bot_id) DO UPDATE SET status=, updated_at=NOW()",
			botID, status)
	}
}

func (d *DB) GetBotStatus(ctx context.Context, botID string) (map[string]interface{}, error) {
	sql := `SELECT bot_id, status, COALESCE(current_task_id, ''), COALESCE(current_task_title, ''), COALESCE(message, ''), heartbeat_at, updated_at FROM chat.bot_statuses WHERE bot_id=$1`
	row := d.pool.QueryRow(ctx, sql, botID)
	var id, status, taskID, taskTitle, msg string
	var heartbeat, updated time.Time
	err := row.Scan(&id, &status, &taskID, &taskTitle, &msg, &heartbeat, &updated)
	if err == pgx.ErrNoRows {
		return map[string]interface{}{
			"bot_id": botID,
			"status": "offline",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"bot_id":             id,
		"status":             status,
		"heartbeat_at":       heartbeat.Format(time.RFC3339),
		"updated_at":         updated.Format(time.RFC3339),
	}
	if taskID != "" {
		result["current_task_id"] = taskID
		result["current_task_title"] = taskTitle
	}
	if msg != "" {
		result["message"] = msg
	}
	return result, nil
}
func (d *DB) SaveSystemMessage(ctx context.Context, userID, content string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx,
		"INSERT INTO chat.friend_messages (from_id, to_id, content, created_at) VALUES ('system', $1, $2, NOW()) RETURNING id",
		userID, content).Scan(&id)
	return id, err
}
func (d *DB) ValidateAPIKey(ctx context.Context, key string) (string, error) {
	var orgID string
	err := d.pool.QueryRow(ctx,
		"SELECT org_id FROM chat.api_keys WHERE key_hash=encode(sha256($1::bytea),'hex') AND status='active'", key).Scan(&orgID)
	if err != nil { return "", err }
	d.pool.Exec(ctx, "UPDATE chat.api_keys SET last_used_at=NOW() WHERE key_hash=encode(sha256($1::bytea),'hex')", key)
	return orgID, nil
}

