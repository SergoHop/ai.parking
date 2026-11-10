package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds runtime settings loaded from environment variables.
type Config struct {
	AppEnv        string
	HTTPPort      string
	DatabaseURL   string
	CORSOrigins   []string
	JWTSecret     string
	InternalToken string
}

// Load reads .env files (without overriding real env) and builds Config.
func Load() (*Config, error) {
	// Local run from the repository root or from backend/.
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		AppEnv:        getenv("APP_ENV", "dev"),
		HTTPPort:      getenv("HTTP_PORT", "8080"),
		DatabaseURL:   strings.TrimSpace(os.Getenv("DATABASE_URL")),
		CORSOrigins:   splitList(getenv("CORS_ORIGINS", "http://localhost:5173")),
		JWTSecret:     strings.TrimSpace(os.Getenv("JWT_SECRET")),
		InternalToken: strings.TrimSpace(os.Getenv("INTERNAL_TOKEN")),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func (c *Config) IsDev() bool {
	return c.AppEnv == "dev"
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
