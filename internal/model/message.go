package model

import "time"

type Message struct {
	ID        int64     `json:"id"`
	GroupID   int64     `json:"group_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	MsgType   string    `json:"msg_type"`
	CreatedAt time.Time `json:"created_at"`
}
