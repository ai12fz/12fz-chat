package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Simple JWT implementation without external dependencies

type AuthHandler struct {
	jwtSecret string
	adminID   string
	adminPass string
	ssoSecret string
	botTokens map[string]string // bot_id -> pre-shared token
}

func NewAuthHandler(jwtSecret, adminID, adminPass, ssoSecret string, botTokens map[string]string) *AuthHandler {
	return &AuthHandler{
		jwtSecret: jwtSecret,
		adminID:   adminID,
		adminPass: adminPass,
		ssoSecret: ssoSecret,
		botTokens: botTokens,
	}
}

type LoginRequest struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer int    `json:"captcha_answer"`
}

type LoginResponse struct {
	Token  string `json:"token"`
	BotID  string `json:"bot_id"`
	Expire int64  `json:"expire"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}

	// Verify captcha
	if req.CaptchaID != "" {
		if !getCaptchaStore().Verify(req.CaptchaID, req.CaptchaAnswer) {
			jsonError(w, "验证码错误", 400)
			return
		}
	}

	if req.Username != h.adminID || req.Password != h.adminPass {
		jsonError(w, "invalid credentials", 401)
		return
	}

	expire := time.Now().Add(24 * time.Hour).Unix()
	token, err := h.generateToken(h.adminID, expire)
	if err != nil {
		jsonError(w, "token error", 500)
		return
	}

	jsonResp(w, LoginResponse{
		Token:  token,
		BotID:  h.adminID,
		Expire: expire,
	}, 200)
}

func (h *AuthHandler) generateToken(botID string, expire int64) (string, error) {
	header := base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadData, _ := json.Marshal(map[string]interface{}{
		"bot_id": botID,
		"exp":    expire,
	})
	payload := base64URLEncode(payloadData)
	sig := hmacSHA256(header+"."+payload, h.jwtSecret)
	return header + "." + payload + "." + sig, nil
}

// ValidateToken parses and validates a JWT token, returns bot_id
func (h *AuthHandler) ValidateToken(token string) (string, error) {
	// Check bot pre-shared tokens first
	for botID, botToken := range h.botTokens {
		if token == botToken {
			return botID, nil
		}
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	// Verify signature
	expectedSig := hmacSHA256(parts[0]+"."+parts[1], h.jwtSecret)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return "", fmt.Errorf("invalid signature")
	}

	// Decode payload
	payloadJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid payload: %w", err)
	}

	var payload struct {
		BotID string `json:"bot_id"`
		Exp   int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return "", fmt.Errorf("invalid payload json: %w", err)
	}

	if time.Now().Unix() > payload.Exp {
		return "", fmt.Errorf("token expired")
	}

	return payload.BotID, nil
}

// ── SSO Login ──

type SSOLoginRequest struct {
	Source       string `json:"source"`        // e.g. "erp", "wp", "future_system"
	UserID       string `json:"user_id"`       // e.g. "suzao", "admin"
	DisplayName  string `json:"display_name"`  // e.g. "数造科技", "管理员"
	SSOSecret    string `json:"sso_secret"`    // shared secret key
}

type SSOLoginResponse struct {
	Token      string `json:"token"`
	BotID      string `json:"bot_id"`
	Source     string `json:"source"`
	SourceName string `json:"source_name"`
	Expire     int64  `json:"expire"`
}

func (h *AuthHandler) SSOLogin(w http.ResponseWriter, r *http.Request) {
	var req SSOLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}

	// Validate required fields
	if req.Source == "" || req.UserID == "" || req.SSOSecret == "" {
		jsonError(w, "missing required fields (source, user_id, sso_secret)", 400)
		return
	}

	// Validate SSO secret
	if req.SSOSecret != h.ssoSecret {
		jsonError(w, "invalid sso_secret", 401)
		return
	}

	// Build bot_id: just the user_id (same user from different systems = same chat user)
	botID := req.UserID

	// Generate short-lived token (2 hours)
	expire := time.Now().Add(2 * time.Hour).Unix()
	token, err := h.generateSSOToken(botID, req.Source, req.DisplayName, expire)
	if err != nil {
		jsonError(w, "token error", 500)
		return
	}

	jsonResp(w, SSOLoginResponse{
		Token:      token,
		BotID:      botID,
		Source:     req.Source,
		SourceName: req.DisplayName,
		Expire:     expire,
	}, 200)
}

func (h *AuthHandler) generateSSOToken(botID, source, name string, expire int64) (string, error) {
	header := base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadData, _ := json.Marshal(map[string]interface{}{
		"bot_id": botID,
		"source": source,
		"name":   name,
		"exp":    expire,
	})
	payload := base64URLEncode(payloadData)
	sig := hmacSHA256(header+"."+payload, h.jwtSecret)
	return header + "." + payload + "." + sig, nil
}
func ExtractTokenFromHeader(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func base64URLDecode(s string) ([]byte, error) {
	// Add padding
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

func hmacSHA256(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return base64URLEncode(mac.Sum(nil))
}
