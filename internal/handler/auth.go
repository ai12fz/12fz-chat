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
}

func NewAuthHandler(jwtSecret, adminID, adminPass string) *AuthHandler {
	return &AuthHandler{
		jwtSecret: jwtSecret,
		adminID:   adminID,
		adminPass: adminPass,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
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

// ExtractTokenFromHeader extracts Bearer token from Authorization header
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
