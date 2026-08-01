package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// SsoExchange handles POST /api/sso/exchange
//
// The marketplace portal (go.12fz.com) embeds the chat UI (ai.12fz.com) via
// iframe and passes its own JWT (localStorage jwtToken) as ?token=. That JWT
// is not signed with this server's jwtSecret, so it cannot pass ValidateToken.
// This endpoint validates the marketplace JWT upstream via
// go.12fz.com/api/sys/home and, when valid, issues a local chat JWT that the
// frontend stores and uses for all subsequent /api and /ws calls.
func (h *HTTPHandler) SsoExchange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}
	if body.Token == "" {
		jsonError(w, "missing token", 400)
		return
	}

	// Validate the marketplace token upstream (same proxy pattern as WhoAmI).
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

	upstreamBody, err := io.ReadAll(resp.Body)
	if err != nil {
		jsonError(w, "upstream read: "+err.Error(), 502)
		return
	}

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
	if err := json.Unmarshal(upstreamBody, &upstream); err != nil {
		jsonError(w, "upstream parse: "+err.Error(), 502)
		return
	}

	// Anonymous/expired marketplace tokens resolve to userInfo with empty
	// user_id (default admin placeholder) — only real users get a non-empty id.
	uid := upstream.Data.UserInfo.UserID
	if uid == "" || upstream.Code != 0 {
		jsonError(w, "invalid token", 401)
		return
	}

	// Issue a local chat JWT bound to the marketplace user id.
	jwt, err := h.authHandler.SignJWT(uid, 24*time.Hour)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]interface{}{"token": jwt}, 200)
}
