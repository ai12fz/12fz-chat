package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	PlatformDSN string
	Port      int
	PGConnStr string
	JWTSecret string
	BotTokens map[string]string
	DocsDir   string
}

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
		Port:      getEnvInt("PORT", 8081),
		PlatformDSN: getEnv("PLATFORM_DSN", getEnv("PG_CONN", "postgresql://app_zhongtai:@localhost:5432/12fzsj?sslmode=disable")),
		PGConnStr: getEnv("PG_CONN", "postgresql://app_zhongtai:@localhost:5432/12fzsj?sslmode=disable"),
		JWTSecret: getEnv("JWT_SECRET", "12fz-chat-secret-2026"),
		BotTokens: bt,
		DocsDir:   getEnv("DOCS_DIR", "/root/12fzwebsocket/docs"),
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
