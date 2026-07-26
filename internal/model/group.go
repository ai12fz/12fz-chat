package model

import "time"

type Group struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type GroupMember struct {
	GroupID       int64     `json:"group_id"`
	UserID        int64     `json:"user_id"`
	Role          string    `json:"role"`
	JoinedAt      time.Time `json:"joined_at"`
	LastReadMsgID int64     `json:"last_read_msg_id,omitempty"`
}
