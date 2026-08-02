package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ai12fz/12fz-chat/internal/db"
	"github.com/ai12fz/12fz-chat/internal/model"
	"github.com/ai12fz/12fz-chat/internal/ws"
	"github.com/gorilla/mux"
)

type contextKey string

const (
	contextBotID contextKey = "bot_id"
)

type HTTPHandler struct {
	db          *db.DB
	hub         *ws.Hub
	authHandler *AuthHandler
	docsDir     string
	startTime   time.Time
}

func NewHTTPHandler(database *db.DB, hub *ws.Hub, auth *AuthHandler, docsDir string) *HTTPHandler {
	return &HTTPHandler{
		db:          database,
		hub:         hub,
		authHandler: auth,
		docsDir:     docsDir,
		startTime:   time.Now(),
	}
}

// AuthMiddleware validates JWT token from Authorization header
func (h *HTTPHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ExtractTokenFromHeader(r)
		if token == "" {
			http.Error(w, `{"error":"missing authorization"}`, 401)
			return
		}
		botID, err := h.authHandler.ValidateToken(token)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, 401)
			return
		}
		ctx := context.WithValue(r.Context(), contextBotID, botID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminOnly restricts a subrouter to the platform admin (user id "1").
func (h *HTTPHandler) AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if getBotID(r) != "1" {
			http.Error(w, `{"error":"forbidden"}`, 403)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// getBotID extracts bot_id from request context
func getBotID(r *http.Request) string {
	if v, ok := r.Context().Value(contextBotID).(string); ok {
		// Strip :chat suffix (used by iframe connections)
		if idx := strings.Index(v, ":"); idx > 0 {
			return v[:idx]
		}
		return v
	}
	return ""
}

// StaticHandler serves the frontend HTML and assets from frontend/dist/
func (h *HTTPHandler) StaticHandler() http.Handler {
	fs := http.FileServer(http.Dir("frontend/dist"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For SPA: serve index.html for all non-API, non-WS routes
		path := r.URL.Path
		if !strings.HasPrefix(path, "/assets/") && !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/ws") && !strings.HasSuffix(path, ".sh") {
			http.ServeFile(w, r, "frontend/dist/index.html")
			return
		}
		// Serve static files if they exist
		fs.ServeHTTP(w, r)
	})
}

// ── Group ──

func (h *HTTPHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	botID := getBotID(r)
	group, err := h.db.CreateGroup(r.Context(), req.Name, func() int64 { id, _ := strconv.ParseInt(botID, 10, 64); return id }())
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	// Auto-add creator as admin
	if err := h.db.AddMember(r.Context(), group.ID, botID, "admin"); err != nil {
		log.Printf("[http] add creator to group error: %v", err)
	}
	jsonResp(w, group, 201)
}

func (h *HTTPHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	groups, err := h.db.ListGroupsForUser(r.Context(), botID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, groups, 200)
}

func (h *HTTPHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		jsonError(w, "invalid group id", 400)
		return
	}
	members, err := h.db.GetMembers(r.Context(), id)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, members, 200)
}

func (h *HTTPHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		jsonError(w, "invalid group id", 400)
		return
	}
	var req struct {
		BotID string `json:"bot_id"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}
	if err := h.db.AddMember(r.Context(), id, req.BotID, req.Role); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 201)
}

func (h *HTTPHandler) GetMyGroups(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	groups, err := h.db.GetUserGroups(r.Context(), botID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, groups, 200)
}

// ── Message ──

func (h *HTTPHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.ParseInt(r.URL.Query().Get("group_id"), 10, 64)
	if err != nil {
		jsonError(w, "missing group_id", 400)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	msgs, err := h.db.GetMessages(r.Context(), groupID, limit, offset)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, msgs, 200)
}

// POST /api/messages - sends a message and broadcasts via WebSocket
func (h *HTTPHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupID int64  `json:"group_id"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		jsonError(w, "empty content", 400)
		return
	}

	botID := getBotID(r)

	// Save to DB
	msg, err := h.db.CreateAndReturnMessage(r.Context(), req.GroupID, botID, req.Content)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	// Update group's last_msg_at
	_ = h.db.UpdateGroupLastMsg(r.Context(), req.GroupID)

	// Broadcast via WebSocket to all group members
	go h.broadcastMessage(msg)

	jsonResp(w, msg, 201)
}

// broadcastMessage sends a message to all online group members via WS
func (h *HTTPHandler) broadcastMessage(m *db.MessageResult) {
	chatMsg := ws.ChatMessage{
		ID:       m.ID,
		GroupID:  m.GroupID,
		SenderID: m.SenderID,
		Content:  m.Content,
		MsgType:  m.MsgType,
		SendAt:   m.CreatedAt,
	}

	data, err := json.Marshal(ws.WSMessage{
		Type: "message",
		Data: mustJSON(chatMsg),
	})
	if err != nil {
		return
	}

	// Get group members
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	members, err := h.db.GetMembers(ctx, m.GroupID)
	if err != nil {
		return
	}

	var botIDs []string
	for _, member := range members {
		botIDs = append(botIDs, strconv.FormatInt(member.UserID, 10))
		if member.BotID != "" {
			botIDs = append(botIDs, member.BotID)
		}
	}
	h.hub.SendToGroup(m.GroupID, data, botIDs)
}

// ── Unread / Read ──

func (h *HTTPHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	counts, err := h.db.GetUnreadCountForUser(r.Context(), botID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if counts == nil {
		counts = make(map[int64]int)
	}
	jsonResp(w, counts, 200)
}

func (h *HTTPHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupID      int64 `json:"group_id"`
		LastReadMsgID int64 `json:"last_read_msg_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	botID := getBotID(r)
	if err := h.db.UpdateLastRead(r.Context(), req.GroupID, botID, req.LastReadMsgID); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

// ── Friend ──

func (h *HTTPHandler) AddFriend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		FriendID string `json:"friend_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	if err := h.db.AddFriend(r.Context(), req.UserID, req.FriendID, "human"); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 201)
}

func (h *HTTPHandler) UpdateFriendCategory(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	friendID := mux.Vars(r)["id"]
	var req struct{ Category string `json:"category"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResp(w, map[string]string{"error": "bad request"}, 400)
		return
	}
	if req.Category == "" {
		jsonResp(w, map[string]string{"error": "category required"}, 400)
		return
	}
	if err := h.db.UpdateFriendCategory(r.Context(), botID, friendID, "", req.Category); err != nil {
		jsonResp(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) GetFriends(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["user_id"]
	friends, err := h.db.GetFriends(r.Context(), userID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, friends, 200)
}

// ListOrgStaff returns staff of the caller's merchant org (excluding the caller).
// GET /api/org/staff
func (h *HTTPHandler) ListOrgStaff(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	uid, err := strconv.ParseInt(botID, 10, 64)
	if err != nil {
		jsonError(w, "invalid bot id", 400)
		return
	}
	orgID, err := h.db.GetOrgID(r.Context(), uid)
	if err != nil || orgID == "" {
		jsonError(w, "org not found", 404)
		return
	}
	staff, err := h.db.ListOrgStaff(r.Context(), orgID, uid)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, staff, 200)
}

// GrantFriend authorizes a device/agent friend to one or more staff users.
// POST /api/friends/{id}/grant  body: {"user_ids": [..]}
// Authorization is stored as extra rows in chat.friends (one per staff),
// so a single host/agent can be granted to multiple employees.
func (h *HTTPHandler) GrantFriend(w http.ResponseWriter, r *http.Request) {
	friendID := mux.Vars(r)["id"]
	var req struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	if len(req.UserIDs) == 0 {
		jsonError(w, "user_ids required", 400)
		return
	}
	// Find the friend's user_type from the caller's own row so grants copy the same type.
	userType := "human"
	friends, err := h.db.GetFriends(r.Context(), getBotID(r))
	if err == nil {
		for _, f := range friends {
			if f.FriendID == friendID {
				userType = f.UserType
				break
			}
		}
	}
	for _, uid := range req.UserIDs {
		if err := h.db.AddFriend(r.Context(), uid, friendID, userType); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}



func (h *HTTPHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "invalid user id", 400)
		return
	}
	orgUser, err := h.db.GetOrgUserByID(r.Context(), userID)
	if err != nil {
		jsonError(w, "user not found", 404)
		return
	}
	jsonResp(w, map[string]interface{}{
		"user_id":  orgUser.UserID,
		"nickname": orgUser.Nickname,
		"phone":    orgUser.Phone,
	}, 200)
}

// ── Health ──

func (h *HTTPHandler) Health(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]any{
		"status":  "ok",
		"service": "12fz-chat",
		"uptime":  time.Since(h.startTime).Seconds(),
	}, 200)
}

// ── Helpers ──

func jsonResp(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	jsonResp(w, map[string]string{"error": msg}, status)
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}


// DeviceHeartbeat updates last_seen for a device
// ListSkills returns all available skills
func (h *HTTPHandler) ListSkills(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id"); if orgID == "" { orgID = "00000000-0000-0000-0000-000000000000" }
	skills, err := h.db.ListSkills(r.Context(), orgID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, skills, 200)
}

// ListCapabilities returns tools+skills for current org
func (h *HTTPHandler) ListCapabilities(w http.ResponseWriter, r *http.Request) {
	var orgID string
	if oid := r.URL.Query().Get("org_id"); oid != "" {
		orgID = oid
	} else {
		orgID = "00000000-0000-0000-0000-000000000000"
	}
	caps, err := h.db.ListCapabilities(r.Context(), orgID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, caps, 200)
}

// CreateSkill adds a new skill

func (h *HTTPHandler) CreateSkill(w http.ResponseWriter, r *http.Request) {
	var s map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		jsonError(w, "invalid body", 400)
		return
	}
	if err := h.db.CreateSkill(r.Context(), s); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

// UpdateSkill updates a skill
func (h *HTTPHandler) UpdateSkill(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var s map[string]interface{}
	json.NewDecoder(r.Body).Decode(&s)
	if err := h.db.UpdateSkill(r.Context(), name, s); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

// DeleteSkill deletes a skill
func (h *HTTPHandler) DeleteSkill(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := h.db.DeleteSkill(r.Context(), name); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

// PublicFriendMessages allows devices to get messages via query token
func (h *HTTPHandler) PublicFriendMessages(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" { jsonError(w, "missing token", 401); return }
	botID, err := h.authHandler.ValidateToken(token)
	if err != nil { jsonError(w, "invalid token: "+err.Error(), 401); return }
	otherID := r.URL.Query().Get("with")
	msgs, err := h.db.GetFriendMessages(r.Context(), botID, otherID, 50, 0)
	if err != nil { jsonError(w, err.Error(), 500); return }
	if msgs == nil { msgs = make([]model.FriendMessage, 0) }
	jsonResp(w, msgs, 200)
}

func (h *HTTPHandler) DeviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	deviceID := getBotID(r)
	if deviceID == "" {
		jsonError(w, "unauthorized", 401)
		return
	}
	// Numeric IDs are regular users, not devices — skip
	if _, err := strconv.Atoi(deviceID); err == nil {
		jsonResp(w, map[string]string{"status": "ok"}, 200)
		return
	}
	// Accept optional ip from body ({"ip":"x.x.x.x"}) or query (?ip=)
	ip := r.URL.Query().Get("ip")
	var req struct {
		IP        string `json:"ip"`
		AgentType string `json:"agent_type"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}
	if ip == "" {
		ip = req.IP
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	var err error
	if ip != "" {
		err = h.db.UpdateDeviceHeartbeat(ctx, deviceID, ip, req.AgentType)
	} else {
		err = h.db.UpdateDeviceLastSeen(ctx, deviceID)
	}
	if err != nil {
		log.Printf("[heartbeat] update failed for %s: %v", deviceID, err)
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) SendFriendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FriendID string `json:"friend_id"`
		Content  string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	botID := getBotID(r)
	id, err := h.db.SaveFriendMessage(r.Context(), botID, req.FriendID, req.Content)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	fm := map[string]interface{}{
		"type": "message", "from": botID, "to": req.FriendID,
		"content": req.Content, "timestamp": time.Now().Unix(),
	}
	data, _ := json.Marshal(fm)
	msg, _ := json.Marshal(ws.WSMessage{Type: "message", Data: data})
	log.Printf("[api] SendFriendMessage: from=%s to=%s, delivering WS", botID, req.FriendID)
	log.Printf("[push] sending to %s", req.FriendID)
	h.hub.SendToBot(req.FriendID, msg)
	log.Printf("[push] sent to %s", req.FriendID)
	// Do NOT echo back to sender - frontend handles local display
	// h.hub.SendToBot(botID, msg)
	jsonResp(w, map[string]interface{}{"status": "ok", "id": id, "content": req.Content, "from": botID, "to": req.FriendID}, 200)
}

func (h *HTTPHandler) GetFriendMessages(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	otherID := r.URL.Query().Get("with")
	log.Printf("[api] GetFriendMessages botID=%s otherID=%s", botID, otherID)
	log.Printf("[api] GetFriendMessages with=%s", r.URL.Query().Get("with")); log.Printf("[api] GetFriendMessages with=%s", r.URL.Query().Get("with")); msgs, err := h.db.GetFriendMessages(r.Context(), botID, otherID, 50, 0)
	if err != nil { jsonError(w, err.Error(), 500); return }
	if msgs == nil { msgs = make([]model.FriendMessage, 0) }
	jsonResp(w, msgs, 200)
}

func (h *HTTPHandler) HandleFriendRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromID string `json:"from_id"`
		Action string `json:"action"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	botID := getBotID(r)
	switch req.Action {
	case "accept":
		h.db.UpdateFriendStatus(r.Context(), botID, req.FromID, "accepted")
		h.db.UpdateFriendStatus(r.Context(), req.FromID, botID, "accepted")
	case "reject":
		h.db.DeleteFriend(r.Context(), botID, req.FromID)
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) ListConnections(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, []map[string]string{{"count": strconv.Itoa(h.hub.ConnectionCount())}}, 200)
}

// GetAgentStatus returns agent/bot current status
func (h *HTTPHandler) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	botID := r.URL.Query().Get("bot_id")
	if botID == "" {
		w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": "bot_id required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	status, err := h.db.GetBotStatus(ctx, botID)
	if err != nil {
		log.Printf("[http] get agent status error: %v", err)
		w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{"error": "internal"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
// ── Agent CRUD ──

func (h *HTTPHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	mid := r.Header.Get("X-Merchant-ID")
	deviceID := r.URL.Query().Get("device_id")
	var (
		agents []db.Agent
		err    error
	)
	if deviceID != "" {
		agents, err = h.db.ListAgentsByDevice(r.Context(), deviceID)
	} else {
		agents, err = h.db.ListAgents(r.Context(), mid)
	}
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, agents, 200)
}

// AgentHeartbeat updates agent heartbeat_at + status (called by device bridge for each running agent)
func (h *HTTPHandler) AgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]
	if botID == "" {
		jsonError(w, "bot_id required", 400)
		return
	}
	if err := h.db.TouchAgentHeartbeat(r.Context(), botID); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]
	agent, err := h.db.GetAgent(r.Context(), botID)
	if err != nil {
		jsonError(w, "not found", 404)
		return
	}
	jsonResp(w, agent, 200)
}

func (h *HTTPHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var a db.Agent
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		jsonError(w, "invalid body", 400)
		return
	}
	if a.BotID == "" || a.DisplayName == "" {
		jsonError(w, "bot_id and display_name required", 400)
		return
	}
	// Check duplicate name
	if exists, err := h.db.AgentNameExists(r.Context(), a.DisplayName); err == nil && exists {
		jsonError(w, "Agent name already exists", 409)
		return
	}
	if a.Status == "" {
		a.Status = "active"
	}
	if a.MerchantID == "" {
		a.MerchantID = r.Header.Get("X-Merchant-ID")
	}
	if a.Model == "" {
		a.Model = "gpt-4"
	}
	if err := h.db.CreateAgent(r.Context(), &a); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	// Auto-add agent as friend for the creating user
	h.db.AddFriend(r.Context(), getBotID(r), a.BotID, "agent")
	jsonResp(w, a, 201)
}

func (h *HTTPHandler) UpdateAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]
	var a db.Agent
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		jsonError(w, "invalid body", 400)
		return
	}
	if err := h.db.UpdateAgent(r.Context(), botID, &a); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]
	if err := h.db.DeleteAgent(r.Context(), botID); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

// Agent Group Bindings

func (h *HTTPHandler) AgentGroups(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]
	groups, err := h.db.GetGroupsForBot(r.Context(), botID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, groups, 200)
}

func (h *HTTPHandler) SetAgentGroups(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]
	var req struct {
		GroupIDs []int64 `json:"group_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", 400)
		return
	}
	if err := h.db.SetBotGroups(r.Context(), botID, req.GroupIDs); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}


func (h *HTTPHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		DeviceKey string `json:"device_key"`
		OS        string `json:"os"`
		AgentType string `json:"agent_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.DeviceKey == "" {
		jsonError(w, "name and device_key required", 400)
		return
	}
	dev, err := h.db.RegisterDevice(r.Context(), req.Name, req.DeviceKey, req.OS, req.AgentType)
	if err != nil {
		jsonError(w, "invalid or used registration code: "+err.Error(), 400)
		return
	}
	// Auto-add device as friend for org admin
	if adminID, err := h.db.GetOrgAdminID(r.Context(), dev.OrgID); err == nil {
		h.db.AutoFriendDevice(r.Context(), adminID, dev.Name)
	} else {
		// Fallback: friend the super admin
		h.db.AutoFriendDevice(r.Context(), "1", dev.Name)
	}
	jsonResp(w, dev, 201)
}

func (h *HTTPHandler) GenerateRegCode(w http.ResponseWriter, r *http.Request) {
	orgID, err := h.db.GetOrgID(r.Context(), parseInt64(getBotID(r)))
	if err != nil {
		jsonError(w, "org not found", 400)
		return
	}
	code, err := h.db.GenerateRegCode(r.Context(), orgID, getBotID(r))
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]interface{}{"code": code, "install_cmd": "curl -s https://ai.12fz.com/install-device.sh | bash -s -- --code=" + code}, 201)
}

func (h *HTTPHandler) DeviceCommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BotID string `json:"bot_id"`
		Cmd   string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BotID == "" || req.Cmd == "" {
		jsonError(w, "bot_id and cmd required", 400)
		return
	}
	cmdID := fmt.Sprintf("%d", time.Now().UnixNano())
	msg, _ := json.Marshal(ws.WSMessage{
		Type: "command",
		Data: mustJSON(map[string]string{"id": cmdID, "cmd": req.Cmd}),
	})
	log.Printf("[cmd] sending to %s: %s", req.BotID, req.Cmd)
	h.hub.SendToBot(req.BotID, msg)
	jsonResp(w, map[string]string{"status": "sent", "cmd_id": cmdID}, 200)
}

func (h *HTTPHandler) DeviceSetup(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		jsonError(w, "missing token", 400)
		return
	}
	_, err := h.db.ValidateDeviceToken(r.Context(), token)
	if err != nil {
		jsonError(w, "invalid token", 401)
		return
	}
	b := make([]byte, 24)
	rand.Read(b)
	key := "sk-dev-" + hex.EncodeToString(b)
	h.db.StoreAPIKey(r.Context(), key, token)
	jsonResp(w, map[string]interface{}{"key": key, "balance": 100000}, 200)
}

// GetDeviceModel returns the model config for a device
func (h *HTTPHandler) GetDeviceModel(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	model, provider, err := h.db.GetDeviceModelConfig(r.Context(), id)
	if err != nil {
		jsonError(w, "device not found", 404)
		return
	}
	jsonResp(w, map[string]string{"model_name": model, "model_provider": provider}, 200)
}

// SetDeviceModel updates the model config for a device
func (h *HTTPHandler) SetDeviceModel(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req struct {
		Model    string `json:"model_name"`
		Provider string `json:"model_provider"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Model == "" {
		jsonError(w, "model_name required", 400)
		return
	}
	if err := h.db.SetDeviceModelConfig(r.Context(), id, req.Model, req.Provider); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}


// ProxyChat forwards /v1/chat/completions to new-api
func (h *HTTPHandler) ProxyChat(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "")
}

// ProxyModels forwards /v1/models to new-api
func (h *HTTPHandler) ProxyModels(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, []map[string]interface{}{}, 200)
}

func (h *HTTPHandler) proxyRequest(w http.ResponseWriter, r *http.Request, target string) {
	start := time.Now()
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Parse model from request
	var reqBody struct {
		Model    string                   `json:"model"`
		Messages []map[string]interface{} `json:"messages"`
	}
	json.Unmarshal(body, &reqBody)

	// Route to model endpoint from DB (real key, not masked)
	endpoint := target
	apiKey := ""
	if ep, k, err := h.db.ProxyGetModelKey(r.Context(), reqBody.Model); err == nil {
		if ep != "" {
			endpoint = ep
		}
		apiKey = k
	}
	// === BILLING: extract org_id + key_id from client API key ===
	orgID := "00000000-0000-0000-0000-000000000000" // fallback
	var keyID int
	clientKey := ExtractTokenFromHeader(r)
	if clientKey != "" {
		if kid, oid, err := h.db.LookupProxyKey(r.Context(), clientKey); err == nil {
			keyID = kid
			orgID = oid
		}
	}

	// === BILLING: pre-check balance ===
	if orgID != "00000000-0000-0000-0000-000000000000" {
		bal, err := h.db.GetOrgBalance(r.Context(), orgID)
		if err == nil && bal <= 0 {
			jsonError(w, "insufficient balance", 402)
			return
		}
	}

	if endpoint == "" {
		jsonError(w, "no endpoint for model: "+reqBody.Model, 400)
		return
	}

	req, _ := http.NewRequest(r.Method, endpoint, io.NopCloser(strings.NewReader(string(body))))
	for k, vs := range r.Header {
		if k == "Authorization" { continue }
		for _, v := range vs { req.Header.Add(k, v) }
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		jsonError(w, "proxy error: "+err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	durationMs := int(time.Since(start).Milliseconds())

	respBody, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Model string `json:"model"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(respBody, &parsed)
	modelName := parsed.Model
	if modelName == "" { modelName = reqBody.Model }
	tokenCount := parsed.Usage.TotalTokens
	if tokenCount == 0 { tokenCount = parsed.Usage.PromptTokens + parsed.Usage.CompletionTokens }

	// Sync log with retry (critical for billing)
	var billingCost float64
	for attempt := 0; attempt < 2; attempt++ {
		c, e := h.db.LogProxyUsage(r.Context(), orgID, keyID, modelName, tokenCount)
		if e == nil { billingCost = c; break }
		if attempt == 1 {
			log.Printf("[proxy] USAGE DROP: model=%s tokens=%d err=%v", modelName, tokenCount, e)
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}
	// Deduct balance if real org
	if billingCost > 0 && orgID != "00000000-0000-0000-0000-000000000000" {
		if err := h.db.ConsumeBalance(r.Context(), orgID, billingCost); err != nil {
			log.Printf("[proxy] BALANCE DEDUCT FAIL: org=%s cost=%.4f err=%v", orgID, billingCost, err)
		}
	}

	for k, vs := range resp.Header {
		for _, v := range vs { w.Header().Add(k, v) }
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
	log.Printf("[proxy] %s -> %s %dms %dtok", reqBody.Model, modelName, durationMs, tokenCount)
}
func (h *HTTPHandler) ListRegCodes(w http.ResponseWriter, r *http.Request) {
	orgID, err := h.db.GetOrgID(r.Context(), parseInt64(getBotID(r)))
	if err != nil {
		jsonError(w, "org not found", 400)
		return
	}
	codes, err := h.db.ListRegCodes(r.Context(), orgID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, codes, 200)
}

func (h *HTTPHandler) RevokeRegCode(w http.ResponseWriter, r *http.Request) {
	orgID, err := h.db.GetOrgID(r.Context(), parseInt64(getBotID(r)))
	if err != nil {
		jsonError(w, "org not found", 400)
		return
	}
	code := mux.Vars(r)["code"]
	if err := h.db.RevokeRegCode(r.Context(), orgID, code); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func (h *HTTPHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	botID := getBotID(r)
	uid, err := strconv.ParseInt(botID, 10, 64)
	if err != nil {
		jsonError(w, "invalid user", 400)
		return
	}
	orgID, err := h.db.GetOrgID(r.Context(), uid)
	if err != nil {
		jsonError(w, "org not found", 400)
		return
	}
	devs, err := h.db.ListDevicesByOrg(r.Context(), orgID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]interface{}{"devices": devs}, 200)
}

func (h *HTTPHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.db.DeleteDevice(r.Context(), id); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "deleted"}, 200)
}

func (h *HTTPHandler) DeviceAgents(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		jsonError(w, "token required", 401)
		return
	}
	dev, err := h.db.ValidateDeviceToken(r.Context(), token)
	if err != nil {
		jsonError(w, "invalid token", 401)
		return
	}
	agents, err := h.db.PendingAgentsByOrg(r.Context(), dev.OrgID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, agents, 200)
}

// PublicListDevices extracts token from header, validates via go.12fz.com, lists devices
func (h *HTTPHandler) PublicListDevices(w http.ResponseWriter, r *http.Request) {
	token := ExtractTokenFromHeader(r)
	if token == "" {
		jsonError(w, "unauthorized", 401)
		return
	}
	orgID, err := h.resolveOrgFromToken(token)
	if err != nil {
		jsonError(w, "auth failed: "+err.Error(), 401)
		return
	}
	devs, err := h.db.ListDevicesByOrg(r.Context(), orgID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]interface{}{"devices": devs}, 200)
}

func (h *HTTPHandler) PublicDeleteDevice(w http.ResponseWriter, r *http.Request) {
	token := ExtractTokenFromHeader(r)
	if token == "" {
		jsonError(w, "unauthorized", 401)
		return
	}
	_, err := h.resolveOrgFromToken(token)
	if err != nil {
		jsonError(w, "auth failed: "+err.Error(), 401)
		return
	}
	id := mux.Vars(r)["id"]
	if err := h.db.DeleteDevice(r.Context(), id); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) resolveOrgFromToken(token string) (string, error) {
	req, _ := http.NewRequest("GET", "https://go.12fz.com/api/sys/home", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var upstream struct {
		Code int `json:"code"`
		Data struct {
			UserInfo struct {
				OrgID    string `json:"org_id"`
				Username string `json:"username"`
			} `json:"userInfo"`
		} `json:"data"`
	}
	json.Unmarshal(body, &upstream)
	if upstream.Data.UserInfo.OrgID == "" {
		// Super admin has no org bound in the zhongtai SSO — fall back to the
		// master org where all system devices live.
		if upstream.Data.UserInfo.Username == "admin" {
			return "00000000-0000-0000-0000-000000000000", nil
		}
		return "", fmt.Errorf("org not found")
	}
	return upstream.Data.UserInfo.OrgID, nil
}

func (h *HTTPHandler) PostDeviceActivity(w http.ResponseWriter, r *http.Request) {
	deviceID := getBotID(r)
	var body struct { Action string `json:"action"`; Detail string `json:"detail"` }
	json.NewDecoder(r.Body).Decode(&body)
	if body.Action == "" { body.Action = "unknown" }
	h.db.LogDeviceActivity(r.Context(), deviceID, body.Action, body.Detail)
	// local_ip 上报：更新设备本地IP
	if body.Action == "local_ip" && body.Detail != "" {
		if err := h.db.SetDeviceLocalIP(r.Context(), deviceID, body.Detail); err != nil {
			// 记录失败但不阻断上报
			h.db.LogDeviceActivity(r.Context(), deviceID, "local_ip_error", err.Error())
		}
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) GetDeviceActivity(w http.ResponseWriter, r *http.Request) {
	deviceID := mux.Vars(r)["id"]
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" { fmt.Sscanf(l, "%d", &limit) }
	activity, _ := h.db.GetDeviceActivity(r.Context(), deviceID, limit)
	if activity == nil { activity = []map[string]interface{}{} }
	jsonResp(w, activity, 200)
}

func (h *HTTPHandler) ToggleDeviceSkills(w http.ResponseWriter, r *http.Request) {
	deviceID := mux.Vars(r)["id"]
	if err := h.db.ToggleDeviceSkills(r.Context(), deviceID); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) ToggleDeviceSoftware(w http.ResponseWriter, r *http.Request) {
	deviceID := mux.Vars(r)["id"]
	if err := h.db.ToggleDeviceSoftware(r.Context(), deviceID); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}
