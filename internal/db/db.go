package db

import (
	"context"
	"fmt"
	"time"

	"github.com/ai12fz/12fz-chat/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
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

func NewFromPool(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

func (d *DB) Close() {
	d.pool.Close()
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
			"bot_id TEXT NOT NULL," +
			"role TEXT DEFAULT 'member'," +
			"joined_at TIMESTAMPTZ DEFAULT NOW()," +
			"last_read_msg_id INT DEFAULT 0," +
			"PRIMARY KEY (group_id, bot_id)" +
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

func (d *DB) SaveMessage(ctx context.Context, m *model.Message) error {
	err := d.pool.QueryRow(ctx,
		"INSERT INTO chat.messages (group_id, sender_id, content, msg_type) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		m.GroupID, m.SenderID, m.Content, m.MsgType,
	).Scan(&m.ID, &m.CreatedAt)
	return err
}

func (d *DB) GetMessages(ctx context.Context, groupID int64, limit, offset int) ([]model.Message, error) {
	if limit <= 0 || limit > 200 {
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
	LastMsgAt time.Time `json:"last_msg_at"`
}

func (d *DB) CreateGroup(ctx context.Context, name, createdBy string) (*model.Group, error) {
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
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt, &g.LastMsgAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// ListGroupsForUser returns groups the user is a member of, sorted by last_msg_at DESC
func (d *DB) ListGroupsForUser(ctx context.Context, botID string) ([]GroupWithMeta, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT g.id, g.name, g.created_by, g.created_at, g.last_msg_at
		 FROM chat.groups g
		 JOIN chat.group_members m ON m.group_id = g.id
		 WHERE m.bot_id = $1
		 ORDER BY g.last_msg_at DESC`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []GroupWithMeta
	for rows.Next() {
		var g GroupWithMeta
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt, &g.LastMsgAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (d *DB) AddMember(ctx context.Context, groupID int64, botID, role string) error {
	_, err := d.pool.Exec(ctx,
		"INSERT INTO chat.group_members (group_id, bot_id, role) VALUES ($1, $2, $3) ON CONFLICT (group_id, bot_id) DO UPDATE SET role = $3",
		groupID, botID, role)
	return err
}

func (d *DB) GetMembers(ctx context.Context, groupID int64) ([]model.GroupMember, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT group_id, bot_id, role, joined_at FROM chat.group_members WHERE group_id = $1",
		groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []model.GroupMember
	for rows.Next() {
		var m model.GroupMember
		if err := rows.Scan(&m.GroupID, &m.BotID, &m.Role, &m.JoinedAt); err != nil {
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
		"UPDATE chat.group_members SET last_read_msg_id = $1 WHERE group_id = $2 AND bot_id = $3",
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
		   WHERE gm.group_id = $1 AND gm.bot_id = $2
		 )`,
		groupID, botID).Scan(&count)
	return count, err
}

// GetUnreadCountForUser returns unread count for all user's groups
func (d *DB) GetUnreadCountForUser(ctx context.Context, botID string) (map[int64]int, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT m.group_id, COUNT(*) AS unread
		 FROM chat.messages m
		 JOIN chat.group_members gm ON gm.group_id = m.group_id AND gm.bot_id = $1
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

// ── DM (Direct Message) Group ──

// FindOrCreateDMGroup finds or creates a 2-person DM group.
// DM group name: "__dm__userA__userB__" where userA < userB (sorted).
func (d *DB) FindOrCreateDMGroup(ctx context.Context, userID, friendID string) (*model.Group, error) {
	userA, userB := userID, friendID
	if userA > userB {
		userA, userB = userB, userA
	}
	dmName := fmt.Sprintf("__dm__%s__%s__", userA, userB)

	// Try to find existing DM group
	var g model.Group
	err := d.pool.QueryRow(ctx,
		"SELECT id, name, created_by, created_at FROM chat.groups WHERE name = $1",
		dmName).Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt)
	if err == nil {
		return &g, nil
	}

	// Create new DM group
	g = model.Group{Name: dmName, CreatedBy: userID}
	err = d.pool.QueryRow(ctx,
		"INSERT INTO chat.groups (name, created_by) VALUES ($1, $2) RETURNING id, created_at",
		dmName, userID,
	).Scan(&g.ID, &g.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Add both users as members
	if err := d.AddMember(ctx, g.ID, userID, "admin"); err != nil {
		return nil, err
	}
	if err := d.AddMember(ctx, g.ID, friendID, "member"); err != nil {
		return nil, err
	}

	return &g, nil
}

// ── Friend ──

func (d *DB) AddFriend(ctx context.Context, userID, friendID string) error {
	_, err := d.pool.Exec(ctx,
		"INSERT INTO chat.friends (user_id, friend_id, status) VALUES ($1, $2, 'pending') ON CONFLICT DO NOTHING",
		userID, friendID)
	return err
}

func (d *DB) GetFriends(ctx context.Context, userID string) ([]model.Friend, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT user_id, friend_id, status, created_at FROM chat.friends WHERE user_id = $1",
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var friends []model.Friend
	for rows.Next() {
		var f model.Friend
		if err := rows.Scan(&f.UserID, &f.FriendID, &f.Status, &f.CreatedAt); err != nil {
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

// CreateAndReturnMessageWithType inserts a message with custom msg_type (e.g. "image")
func (d *DB) CreateAndReturnMessageWithType(ctx context.Context, groupID int64, senderID, content, msgType string) (*MessageResult, error) {
	m := &MessageResult{
		GroupID:  groupID,
		SenderID: senderID,
		Content:  content,
		MsgType:  msgType,
	}
	err := d.pool.QueryRow(ctx,
		`INSERT INTO chat.messages (group_id, sender_id, content, msg_type) VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		groupID, senderID, content, msgType,
	).Scan(&m.ID, &m.CreatedAt)
	return m, err
}
