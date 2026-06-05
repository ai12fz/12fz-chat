package db

import (
	"context"
	"fmt"
	"time"

	"github.com/ai12fz/12fz-chat/internal/config"
	"github.com/ai12fz/12fz-chat/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func Connect(cfg *config.Config) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.PGConnStr)
	if err != nil {
		return nil, fmt.Errorf("pg connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pg ping: %w", err)
	}
	return &DB{pool: pool}, nil
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
			"created_at TIMESTAMPTZ DEFAULT NOW()" +
		")",

		"CREATE TABLE IF NOT EXISTS chat.group_members (" +
			"group_id INT REFERENCES chat.groups(id) ON DELETE CASCADE," +
			"bot_id TEXT NOT NULL," +
			"role TEXT DEFAULT 'member'," +
			"joined_at TIMESTAMPTZ DEFAULT NOW()," +
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
	}
	for _, s := range stmts {
		if _, err := d.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
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

func (d *DB) CreateGroup(ctx context.Context, name, createdBy string) (*model.Group, error) {
	g := &model.Group{Name: name, CreatedBy: createdBy}
	err := d.pool.QueryRow(ctx,
		"INSERT INTO chat.groups (name, created_by) VALUES ($1, $2) RETURNING id, created_at",
		name, createdBy,
	).Scan(&g.ID, &g.CreatedAt)
	return g, err
}

func (d *DB) ListGroups(ctx context.Context) ([]model.Group, error) {
	rows, err := d.pool.Query(ctx, "SELECT id, name, created_by, created_at FROM chat.groups ORDER BY id")
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

func (d *DB) GetUserGroups(ctx context.Context, botID string) ([]model.Group, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT g.id, g.name, g.created_by, g.created_at FROM chat.groups g JOIN chat.group_members m ON m.group_id = g.id WHERE m.bot_id = $1 ORDER BY g.id",
		botID)
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

func (d *DB) CreateAndReturnMessage(ctx context.Context, groupID int64, senderID, content string) (*model.Message, error) {
	m := &model.Message{
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
