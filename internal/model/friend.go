package model

import "time"

type Friend struct {
	UserID    string    `json:"user_id"`
	FriendID  string    `json:"friend_id"`
	UserType  string    `json:"user_type"`
	Category  string    `json:"category,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type FriendMessage struct {
	ID        int64     `json:"id"`
	FromID    string    `json:"from_id"`
	ToID      string    `json:"to_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
