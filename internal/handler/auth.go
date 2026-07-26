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

type AuthHandler struct {
	jwtSecret string
	botTokens map[string]string
}

func NewAuthHandler(jwtSecret string, botTokens map[string]string) *AuthHandler {
	return &AuthHandler{
		jwtSecret: jwtSecret,
		botTokens: botTokens,
	}
}

func (h *AuthHandler) ValidateToken(token string) (string, error) {
	for botID, botToken := range h.botTokens {
		if token == botToken {
			return botID, nil
		}
	}
	if strings.HasPrefix(token, "session-") {
		return token[8:], nil
	}
	// Validate JWT (from go.12fz.com or local)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}
	expectedSig := hmacSHA256(parts[0]+"."+parts[1], h.jwtSecret)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return "", fmt.Errorf("invalid signature")
	}
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
