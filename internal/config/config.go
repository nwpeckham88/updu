package config

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	// Server
	Host    string
	Port    int
	BaseURL string

	// Database
	DBPath string

	// Logging
	LogLevel string

	// Auth
	AuthSecret     string
	SessionTTLDays int
	AdminUser      string
	AdminPassword  string

	// OIDC (optional)
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	OIDCAutoRegister bool

	// Scheduler
	WorkerPoolSize int
	MinIntervalS   int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		Host:    envOr("UPDU_HOST", "0.0.0.0"),
		Port:    envInt("UPDU_PORT", 3000),
		BaseURL: envOr("UPDU_BASE_URL", "http://localhost:3000"),

		DBPath: envOr("UPDU_DB_PATH", "./data/updu.db"),

		LogLevel: envOr("UPDU_LOG_LEVEL", "info"),

		AuthSecret:     envOr("UPDU_AUTH_SECRET", ""),
		SessionTTLDays: envInt("UPDU_SESSION_TTL_DAYS", 7),
		AdminUser:      envOr("UPDU_ADMIN_USER", ""),
		AdminPassword:  envOr("UPDU_ADMIN_PASSWORD", ""),

		OIDCIssuer:       envOr("UPDU_OIDC_ISSUER", ""),
		OIDCClientID:     envOr("UPDU_OIDC_CLIENT_ID", ""),
		OIDCClientSecret: envOr("UPDU_OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:  envOr("UPDU_OIDC_REDIRECT_URL", ""),
		OIDCAutoRegister: envBool("UPDU_OIDC_AUTO_REGISTER", true),

		WorkerPoolSize: envInt("UPDU_WORKER_POOL_SIZE", 0), // 0 = auto
		MinIntervalS:   envInt("UPDU_MIN_INTERVAL_S", 30),
	}

	// Auto-generate auth secret if not set
	if cfg.AuthSecret == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		cfg.AuthSecret = hex.EncodeToString(b)
		slog.Warn("UPDU_AUTH_SECRET not set — auto-generated random secret (sessions will not persist across restarts)")
	}

	return cfg
}

// IsSecure returns true when the base URL uses HTTPS.
func (c *Config) IsSecure() bool {
	return strings.HasPrefix(c.BaseURL, "https://")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		v = strings.ToLower(v)
		return v == "true" || v == "1" || v == "yes"
	}
	return fallback
}
