package ws

import (
	"github.com/gorilla/websocket"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

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
	raw   net.Conn
	hub   *Hub
	send  chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client
	db      DB
}

type DB interface {
	UpdateDeviceLastSeen(ctx context.Context, deviceID string) error
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]*Client)}
}

func NewHubWithDB(db DB) *Hub {
	return &Hub{clients: make(map[string]*Client), db: db}
}

func (h *Hub) Register(client *Client) {
	var oldClose func()
	h.mu.Lock()
	if old, ok := h.clients[client.BotID]; ok {
		oldConn := old.raw
		oldSend := old.send
		oldClose = func() {
			close(oldSend)
			oldConn.Close()
		}
	}
	h.clients[client.BotID] = client
	h.mu.Unlock()

	if oldClose != nil {
		oldClose()
	}

	h.broadcastEvent("user_online", client.BotID)
	log.Printf("[ws] %s connected", client.BotID)
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	if existing, ok := h.clients[client.BotID]; ok && existing == client {
		delete(h.clients, client.BotID)
	}
	h.mu.Unlock()
	if _, ok := h.clients[client.BotID]; !ok {
		h.broadcastEvent("user_offline", client.BotID)
	}
	log.Printf("[ws] %s disconnected", client.BotID)
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
		}
	}
}

func (h *Hub) SendToBot(botID string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if c, ok := h.clients[botID]; ok {
		log.Printf("[sendtobot] found %s in hub, pushing", botID)
		select {
		case c.send <- data:
		default:
			log.Printf("[sendtobot] %s channel full, dropped", botID)
		}
	} else {
		log.Printf("[sendtobot] %s NOT in hub", botID)
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

// ── Raw WebSocket pumps (no gorilla) ──

func (c *Client) WritePumpRaw() {
	defer c.raw.Close()
	for msg := range c.send {
		l := len(msg)
		frame := []byte{0x81}
		if l < 126 {
			frame = append(frame, byte(l))
		} else if l < 65536 {
			frame = append(frame, 126, byte(l>>8), byte(l))
		} else {
			frame = append(frame, 127)
			for i := 7; i >= 0; i-- {
				frame = append(frame, byte(l>>(i*8)))
			}
		}
		frame = append(frame, msg...)
		if _, err := c.raw.Write(frame); err != nil { log.Printf("[ws] WritePump write error for %s: %v", c.BotID, err)
			return
		}
	}
}

func (c *Client) ReadPumpRaw(handler MessageHandler) {
	defer func() {
		c.hub.Unregister(c)
		c.raw.Close()
	}()

	log.Printf("[ws] ReadPump start for %s", c.BotID)
	buf := make([]byte, 65536)
	for {
		n, err := c.raw.Read(buf[:2])
		if err != nil || n < 2 {
			log.Printf("[ws] ReadPump error for %s: %v", c.BotID, err)
			return
		}
		opcode := buf[0] & 0x0F
		plen := int(buf[1] & 0x7F)
		masked := buf[1]&0x80 != 0

		if plen == 126 {
			c.raw.Read(buf[:2])
			plen = int(buf[0])<<8 | int(buf[1])
		} else if plen == 127 {
			c.raw.Read(buf[:8])
			plen = 0
			for i := 0; i < 8; i++ {
				plen = (plen << 8) | int(buf[i])
			}
		}

		var payload []byte
		if masked {
			c.raw.Read(buf[:4])
			mask := buf[:4]
			payload = make([]byte, plen)
			c.raw.Read(payload)
			for i := 0; i < len(payload); i++ {
				payload[i] ^= mask[i%4]
			}
		} else {
			payload = make([]byte, plen)
			c.raw.Read(payload)
		}

		if opcode == 8 {
			return
		}
		if opcode == 9 {
			c.raw.Write([]byte{0x8A, 0x00})
			continue
		}
		if opcode == 1 {
			var msg WSMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				continue
			}
			if msg.Type == "message" {
				handler.HandleMessage(c.BotID, msg.Data)
			}
			if msg.Type == "heartbeat" {
				// Update last_seen for the device/bot
				if c.hub.db != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					c.hub.db.UpdateDeviceLastSeen(ctx, c.BotID)
				}
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

// ServeWS - raw hijack WebSocket (no gorilla dependency)
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, botID string, handler MessageHandler) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", 500)
		return
	}
	rawConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		rawConn.Close()
		return
	}
	hash := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	accept := base64.StdEncoding.EncodeToString(hash[:])
	fmt.Fprintf(rawConn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", accept)

	client := &Client{
		BotID: botID,
		raw:   rawConn,
		hub:   h,
		send:  make(chan []byte, 256),
	}
	h.Register(client)

	go client.WritePumpRaw()

	hello, _ := json.Marshal(WSMessage{
		Type: "hello",
		Data: mustJSON(map[string]string{"bot_id": botID, "msg": fmt.Sprintf("Welcome %s to 12FZ Chat", botID)}),
	})
	client.send <- hello

	go client.ReadPumpRaw(handler)
	// Keep handler alive so http.Server doesn't close the hijacked connection
	select {}
	log.Printf("[ws] servews done for %s", botID)
}

// PushToDevice sends a message to a specific connected device
func (h *Hub) PushToDevice(deviceID string, msg []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for id, client := range h.clients {
		if id == deviceID {
			select {
			case client.send <- msg:
				return true
			default:
				return false
			}
		}
	}
	return false
}

func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}


var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Hub) ServeWSGorilla(w http.ResponseWriter, r *http.Request, botID string, msgHandler MessageHandler) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade failed for %s: %v", botID, err)
		return
	}
	hello, _ := json.Marshal(WSMessage{
		Type: "hello",
		Data: mustJSON(map[string]string{"bot_id": botID, "msg": fmt.Sprintf("Welcome %s to 12FZ Chat", botID)}),
	})
	conn.WriteMessage(websocket.TextMessage, hello)
	client := &Client{BotID: botID, hub: h, send: make(chan []byte, 256)}
	h.Register(client)
	go client.WritePumpGorilla(conn)
	defer h.Unregister(client)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[ws] read error for %s: %v", botID, err)
			break
		}
		msgHandler.HandleMessage(botID, msg)
	}
	conn.Close()
}

func (c *Client) WritePumpGorilla(conn *websocket.Conn) {
	for msg := range c.send {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
}
