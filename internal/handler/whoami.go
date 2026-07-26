package handler

import (
	"encoding/json"
	"io"
	"net/http"
)

func (h *HTTPHandler) WhoAmI(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest("GET", "https://go.12fz.com/api/sys/home", nil)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		jsonError(w, "upstream: "+err.Error(), 502)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	var upstream struct {
		Code int `json:"code"`
		Data struct {
			UserInfo struct {
				NickName string `json:"nickName"`
				UserID   string `json:"user_id"`
				Username string `json:"username"`
				OrgID    string `json:"org_id"`
			} `json:"userInfo"`
		} `json:"data"`
	}
	json.Unmarshal(body, &upstream)

	uid := upstream.Data.UserInfo.UserID
	if uid == "" {
		uid = upstream.Data.UserInfo.Username
	}
	nick := upstream.Data.UserInfo.NickName
	if nick == "" {
		nick = upstream.Data.UserInfo.Username
	}

	jsonResp(w, map[string]interface{}{
		"user_id":  uid,
		"nickname": nick,
		"username": upstream.Data.UserInfo.Username,
		"org_id":   upstream.Data.UserInfo.OrgID,
	}, 200)
}
