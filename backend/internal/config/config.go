package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds runtime settings loaded from environment variables.
type Config struct {
	AppEnv        string
	HTTPPort      string
	DatabaseURL   string
	CORSOrigins   []string
	JWTSecret     string
	JWTTTL        time.Duration
	AdminUsername string
	AdminPassword string
	InternalToken string
	// FreshnessTTL is how long a camera frame stays trustworthy for the API.
	FreshnessTTL time.Duration
}

// Load reads .env files (without overriding real env) and builds Config.
func Load() (*Config, error) {
	// Local run from the repository root or from backend/.
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		AppEnv:      getenv("APP_ENV", "dev"),
		HTTPPort:    getenv("HTTP_PORT", "8080"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		CORSOrigins: splitList(getenv("CORS_ORIGINS",
			"http://localhost:5173,http://127.0.0.1:5173")),
		JWTSecret:     strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AdminUsername: getenv("ADMIN_USERNAME", "admin"),
		AdminPassword: strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")),
		InternalToken: strings.TrimSpace(os.Getenv("INTERNAL_TOKEN")),
	}

	ttl, err := time.ParseDuration(getenv("JWT_TTL", "24h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_TTL must be a duration like 24h: %w", err)
	}
	cfg.JWTTTL = ttl

	freshnessSeconds, err := strconv.Atoi(getenv("FRESHNESS_TTL_SECONDS", "15"))
	if err != nil || freshnessSeconds <= 0 {
		return nil, fmt.Errorf("FRESHNESS_TTL_SECONDS must be a positive integer")
	}
	cfg.FreshnessTTL = time.Duration(freshnessSeconds) * time.Second

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

// StatusTTL returns the configured freshness window, with a safe default.
func (c *Config) StatusTTL() time.Duration {
	if c.FreshnessTTL <= 0 {
		return 15 * time.Second
	}
	return c.FreshnessTTL
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
