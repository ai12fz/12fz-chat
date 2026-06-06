package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port       int
	PGConnStr  string
	JWTSecret  string
	AdminBotID string
	AdminPass  string
}

// PGConnStr method to satisfy db.Connect interface
func (c *Config) PGConnStr() string { return c.PGConnStr }

func Load() *Config {
	return &Config{
		Port:       getEnvInt("PORT", 8081),
		PGConnStr:  getEnv("PG_CONN", "postgresql://gong3:***@localhost:5432/suzao?sslmode=disable"),
		JWTSecret:  getEnv("JWT_SECRET", "12fz-chat-secret-2026"),
		AdminBotID: getEnv("ADMIN_BOT_ID", "admin"),
		AdminPass:  getEnv("ADMIN_PASS", "admin123"),
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
