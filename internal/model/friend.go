package model

import "time"

type Friend struct {
	UserID    string    `json:"user_id"`
	FriendID  string    `json:"friend_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type FriendMessage struct {
	ID        int64
	FromID    string
	ToID      string
	Content   string
	CreatedAt time.Time
}
