package handler

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/ai12fz/12fz-chat/internal/db"
	"github.com/ai12fz/12fz-chat/internal/model"
	"github.com/ai12fz/12fz-chat/internal/ws"
)

type MessageHandler struct {
	db  *db.DB
	hub *ws.Hub
}

func NewMessageHandler(database *db.DB, hub *ws.Hub) *MessageHandler {
	return &MessageHandler{db: database, hub: hub}
}

func (h *MessageHandler) HandleMessage(senderID string, data json.RawMessage) {
	var msg struct {
		GroupID int64  `json:"group_id"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("[handler] bad message from %s: %v", senderID, err)
		return
	}

	m := &model.Message{
		GroupID:  msg.GroupID,
		SenderID: senderID,
		Content:  msg.Content,
		MsgType:  "text",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.db.SaveMessage(ctx, m); err != nil {
		log.Printf("[handler] save msg error: %v", err)
		return
	}
	
	// Update last_msg_at
	if err := h.db.UpdateGroupLastMsg(ctx, msg.GroupID); err != nil {
		log.Printf("[handler] update last_msg_at error: %v", err)
	}

	// Get group members for delivery
	members, err := h.db.GetMembers(ctx, msg.GroupID)
	if err != nil {
		log.Printf("[handler] get members error: %v", err)
		return
	}

	// Broadcast to all online group members
	chatMsg := ws.ChatMessage{
		ID:       m.ID,
		GroupID:  m.GroupID,
		SenderID: m.SenderID,
		Content:  m.Content,
		MsgType:  m.MsgType,
		SendAt:   m.CreatedAt,
	}

	broadcastData, _ := json.Marshal(ws.WSMessage{
		Type: "message",
		Data: mustJSON(chatMsg),
	})

	var botIDs []string
	for _, member := range members {
		botIDs = append(botIDs, member.BotID)
	}
	h.hub.SendToGroup(m.GroupID, broadcastData, botIDs)
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
