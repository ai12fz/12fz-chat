package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port       int
	PGConnStr  string
	JWTSecret  string
	AdminBotID string
	AdminPass  string
	BotTokens  map[string]string // bot_id -> token
	UploadDir  string            // 图片上传保存目录
	SSOSecret  string            // SSO共享密钥（多系统打通）
}

// PGConnString method to satisfy db.Connect interface
func (c *Config) PGConnString() string { return c.PGConnStr }

func Load() *Config {
	bt := make(map[string]string)
	if bots := getEnv("BOT_TOKENS", ""); bots != "" {
		for _, pair := range strings.Split(bots, ",") {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				bt[parts[0]] = parts[1]
			}
		}
	}
	return &Config{
		Port:       getEnvInt("PORT", 8081),
		PGConnStr:  getEnv("PG_CONN", "postgresql://gong3:***@localhost:5432/suzao?sslmode=disable"),
		JWTSecret:  getEnv("JWT_SECRET", "12fz-chat-secret-2026"),
		AdminBotID: getEnv("ADMIN_BOT_ID", "admin"),
		AdminPass:  getEnv("ADMIN_PASS", "admin123"),
		UploadDir:  getEnv("UPLOAD_DIR", "/var/www/chat.12fz.com/uploads"),
		SSOSecret:  getEnv("SSO_SECRET", "12fz-sso-2026"),
		BotTokens:  bt,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}
