package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port      int
	PGConnStr string
	JWTSecret string
}

func Load() *Config {
	return &Config{
		Port:      getEnvInt("PORT", 8081),
		PGConnStr: getEnv("PG_CONN", "postgresql://gong3:Cx99w06020354@localhost:5432/suzao?sslmode=disable"),
		JWTSecret: getEnv("JWT_SECRET", "12fz-chat-secret-2026"),
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
