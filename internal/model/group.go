package model

import "time"

type Group struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type GroupMember struct {
	GroupID  int64     `json:"group_id"`
	BotID    string    `json:"bot_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}
