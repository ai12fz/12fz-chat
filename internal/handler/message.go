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
		GroupID  int64  `json:"group_id"`
		FriendID string `json:"friend_id"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("[handler] bad message from %s: %v", senderID, err)
		return
	}

	// Friend message: relay directly via SendToBot
	if msg.FriendID != "" {
		id, err := h.db.SaveFriendMessage(context.Background(), senderID, msg.FriendID, msg.Content); if err != nil { log.Printf("[handler] save friend msg error: %v", err) } else { log.Printf("[handler] saved friend msg id=%d", id) }
		log.Printf("[handler] friend msg from=%s to=%s", senderID, msg.FriendID)
		fm := map[string]interface{}{
			"type":      "message",
			"from":      senderID,
			"to":        msg.FriendID,
			"content":   msg.Content,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(fm)
		broadcastData, _ := json.Marshal(ws.WSMessage{Type: "message", Data: data})
		log.Printf("[handler] sending to friend %s", msg.FriendID)
		h.hub.SendToBot(msg.FriendID, broadcastData)
		log.Printf("[handler] echoing back to sender %s", senderID)
		h.hub.SendToBot(senderID, broadcastData)
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

	chatMsgJSON, _ := json.Marshal(chatMsg)
	broadcastData, _ := json.Marshal(ws.WSMessage{
		Type: "message",
		Data: json.RawMessage(chatMsgJSON),
	})

	var botIDs []string
	for _, member := range members {
		botIDs = append(botIDs, member.BotID)
	}
	h.hub.SendToGroup(m.GroupID, broadcastData, botIDs)
}
