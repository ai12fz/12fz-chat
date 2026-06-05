package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ai12fz/12fz-chat/internal/db"
	"github.com/gorilla/mux"
)

type HTTPHandler struct {
	db *db.DB
}

func NewHTTPHandler(database *db.DB) *HTTPHandler {
	return &HTTPHandler{db: database}
}

func (h *HTTPHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		CreatedBy string `json:"created_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	group, err := h.db.CreateGroup(r.Context(), req.Name, req.CreatedBy)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, group, 201)
}

func (h *HTTPHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.db.ListGroups(r.Context())
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

func (h *HTTPHandler) AddFriend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		FriendID string `json:"friend_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	if err := h.db.AddFriend(r.Context(), req.UserID, req.FriendID); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 201)
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

func (h *HTTPHandler) GetMyGroups(w http.ResponseWriter, r *http.Request) {
	botID := r.URL.Query().Get("bot_id")
	if botID == "" {
		jsonError(w, "missing bot_id", 400)
		return
	}
	groups, err := h.db.GetUserGroups(r.Context(), botID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, groups, 200)
}

// Health check
func (h *HTTPHandler) Health(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{"status": "ok", "service": "12fz-chat"}, 200)
}

func jsonResp(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	jsonResp(w, map[string]string{"error": msg}, status)
}

// POST /api/messages  {group_id, sender_id, content}
func (h *HTTPHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupID  int64  `json:"group_id"`
		SenderID string `json:"sender_id"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	msg, err := h.db.CreateAndReturnMessage(r.Context(), req.GroupID, req.SenderID, req.Content)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, msg, 201)
}
