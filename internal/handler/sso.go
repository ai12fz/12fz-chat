package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// SsoExchange validates a marketplace JWT (from go.12fz.com) via the
// upstream /api/sys/home endpoint and exchanges it for a local chat JWT.
// The mailbox iframe in go.12fz.com loads ai.12fz.com/chat/?token=<jwt>;
// the frontend then POSTs this token here to get a chat-native token.
func (h *HTTPHandler) SsoExchange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		jsonError(w, "missing token", 400)
		return
	}

	// Validate the marketplace token upstream (go.12fz.com = marketplace).
	// Effective logins return userInfo.user_id non-empty; anonymous or
	// invalid tokens return user_id == "" (default admin shell).
	req, err := http.NewRequest("GET", "https://go.12fz.com/api/sys/home", nil)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	req.Header.Set("Authorization", "Bearer "+body.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		jsonError(w, "upstream: "+err.Error(), 502)
		return
	}
	defer resp.Body.Close()

	upstreamBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(upstreamBody)
		return
	}

	var upstream struct {
		Code int `json:"code"`
		Data struct {
			UserInfo struct {
				UserID   string `json:"user_id"`
				Username string `json:"username"`
			} `json:"userInfo"`
		} `json:"data"`
	}
	json.Unmarshal(upstreamBody, &upstream)

	uid := upstream.Data.UserInfo.UserID
	if uid == "" {
		jsonError(w, "invalid token", 401)
		return
	}

	jwt, err := h.authHandler.SignJWT(uid, 24*time.Hour)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]interface{}{"token": jwt}, 200)
}
