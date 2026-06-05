package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins in dev
	},
}

// WSMessage is the protocol envelope
type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// ChatMessage payload
type ChatMessage struct {
	ID       int64     `json:"id"`
	GroupID  int64     `json:"group_id"`
	SenderID string    `json:"sender_id"`
	Content  string    `json:"content"`
	MsgType  string    `json:"msg_type"`
	SendAt   time.Time `json:"send_at"`
}

// Event payload
type EventPayload struct {
	Event string `json:"event"`
	BotID string `json:"bot_id"`
}

type Client struct {
	BotID string
	conn  *websocket.Conn
	hub   *Hub
	send  chan []byte
}

type Hub struct {
	mu       sync.RWMutex
	clients  map[string]*Client // botID -> client (single connection per bot)
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	// Close old connection if bot reconnects
	if old, ok := h.clients[client.BotID]; ok {
		close(old.send)
		old.conn.Close()
	}
	h.clients[client.BotID] = client
	h.mu.Unlock()

	h.broadcastEvent("user_online", client.BotID)
	log.Printf("[ws] %s connected", client.BotID)
}

func (h *Hub) Unregister(botID string) {
	h.mu.Lock()
	delete(h.clients, botID)
	h.mu.Unlock()

	h.broadcastEvent("user_offline", botID)
	log.Printf("[ws] %s disconnected", botID)
}

func (h *Hub) broadcastEvent(event, botID string) {
	data, _ := json.Marshal(WSMessage{
		Type: "event",
		Data: mustJSON(EventPayload{Event: event, BotID: botID}),
	})
	h.Broadcast(data)
}

func (h *Hub) Broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		select {
		case c.send <- data:
		default:
			// drop slow client
		}
	}
}

func (h *Hub) SendToBot(botID string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if c, ok := h.clients[botID]; ok {
		select {
		case c.send <- data:
		default:
		}
	}
}

func (h *Hub) SendToGroup(groupID int64, data []byte, dbGroupMembers []string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, botID := range dbGroupMembers {
		if c, ok := h.clients[botID]; ok {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

func (h *Hub) IsOnline(botID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[botID]
	return ok
}

// ── Client read/write pumps ──

func (c *Client) ReadPump(botIDs []string, handler MessageHandler) {
	defer func() {
		c.hub.Unregister(c.BotID)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(65536)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		var msg WSMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		if msg.Type == "pong" {
			continue
		}
		if msg.Type == "message" {
			handler.HandleMessage(c.BotID, msg.Data)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type MessageHandler interface {
	HandleMessage(senderID string, data json.RawMessage)
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// ServeWS handles WebSocket upgrade
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, botID string, handler MessageHandler) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	client := &Client{
		BotID: botID,
		conn:  conn,
		hub:   h,
		send:  make(chan []byte, 256),
	}
	h.Register(client)
	go client.WritePump()
	// Send a welcome message
	hello, _ := json.Marshal(WSMessage{
		Type: "hello",
		Data: mustJSON(map[string]string{"bot_id": botID, "msg": fmt.Sprintf("Welcome %s to 12FZ Chat", botID)}),
	})
	client.send <- hello

	go client.ReadPump([]string{botID}, handler)
}
