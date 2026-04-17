package config

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	PasswordPolicyOff        = "off"
	PasswordPolicyDefault    = "default"
	PasswordPolicyStrong     = "strong"
	PasswordPolicyVerySecure = "very_secure"
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
	PasswordPolicy string

	// OIDC (optional)
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	OIDCAutoRegister bool

	// Scheduler
	WorkerPoolSize int
	MinIntervalS   int

	// GitOps / Fetch
	ConfURL    string
	ConfPath   string
	ConfigPath string

	EnableCustomCSS   bool
	AllowLocalhost    bool
	TrustedProxyCIDRs []string

	// Metrics
	MetricsToken string
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
		PasswordPolicy: NormalizePasswordPolicy(envOr("UPDU_PASSWORD_POLICY", PasswordPolicyDefault)),

		OIDCIssuer:       envOr("UPDU_OIDC_ISSUER", ""),
		OIDCClientID:     envOr("UPDU_OIDC_CLIENT_ID", ""),
		OIDCClientSecret: envOr("UPDU_OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:  envOr("UPDU_OIDC_REDIRECT_URL", ""),
		OIDCAutoRegister: envBool("UPDU_OIDC_AUTO_REGISTER", true),

		WorkerPoolSize: envInt("UPDU_WORKER_POOL_SIZE", 0), // 0 = auto
		MinIntervalS:   envInt("UPDU_MIN_INTERVAL_S", 30),

		ConfURL:  os.Getenv("UPDU_CONF_URL"),
		ConfPath: os.Getenv("UPDU_CONF_PATH"),

		EnableCustomCSS:   envBool("UPDU_ENABLE_CUSTOM_CSS", false),
		AllowLocalhost:    envBool("UPDU_ALLOW_LOCALHOST", false),
		TrustedProxyCIDRs: envCSV("UPDU_TRUSTED_PROXY_CIDRS"),

		MetricsToken: envOr("UPDU_METRICS_TOKEN", ""),
	}

	// 1. Discover and load from updu.conf (if it exists)
	cfg.ConfigPath = discoverConfigPath()
	if cfg.ConfigPath != "" {
		yCfg, err := ParseYAMLConfig(cfg.ConfigPath)
		if err == nil {
			// Override defaults with YAML (but env vars still have higher priority)
			// Actually, we should load YAML and then apply env vars on top.
			// Let's re-apply env vars after loading YAML to ensure correct precedence.
			applyYAML(cfg, yCfg)
		}
	}

	// 2. Re-apply environment variables to ensure they have the HIGHEST priority
	applyEnvOverrides(cfg)
	cfg.PasswordPolicy = NormalizePasswordPolicy(cfg.PasswordPolicy)

	// Auto-generate auth secret if not set
	if cfg.AuthSecret == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		cfg.AuthSecret = hex.EncodeToString(b)
		slog.Warn("UPDU_AUTH_SECRET not set — auto-generated random secret (sessions will not persist across restarts)")
	}

	return cfg
}

func applyYAML(cfg *Config, yCfg *YAMLConfig) {
	if yCfg.Host != "" {
		cfg.Host = yCfg.Host
	}
	if yCfg.Port != 0 {
		cfg.Port = yCfg.Port
	}
	if yCfg.BaseURL != "" {
		cfg.BaseURL = yCfg.BaseURL
	}
	if yCfg.DBPath != "" {
		cfg.DBPath = yCfg.DBPath
	}
	if yCfg.LogLevel != "" {
		cfg.LogLevel = yCfg.LogLevel
	}
	if yCfg.AuthSecret != "" {
		cfg.AuthSecret = yCfg.AuthSecret
	}
	if yCfg.SessionTTLDays != 0 {
		cfg.SessionTTLDays = yCfg.SessionTTLDays
	}
	if yCfg.AdminUser != "" {
		cfg.AdminUser = yCfg.AdminUser
	}
	if yCfg.AdminPassword != "" {
		cfg.AdminPassword = yCfg.AdminPassword
	}
	if yCfg.PasswordPolicy != "" {
		cfg.PasswordPolicy = NormalizePasswordPolicy(yCfg.PasswordPolicy)
	}
	if yCfg.OIDCIssuer != "" {
		cfg.OIDCIssuer = yCfg.OIDCIssuer
	}
	if yCfg.OIDCClientID != "" {
		cfg.OIDCClientID = yCfg.OIDCClientID
	}
	if yCfg.OIDCClientSecret != "" {
		cfg.OIDCClientSecret = yCfg.OIDCClientSecret
	}
	if yCfg.OIDCRedirectURL != "" {
		cfg.OIDCRedirectURL = yCfg.OIDCRedirectURL
	}
	if yCfg.OIDCAutoRegister != nil {
		cfg.OIDCAutoRegister = *yCfg.OIDCAutoRegister
	}
	if yCfg.WorkerPoolSize != 0 {
		cfg.WorkerPoolSize = yCfg.WorkerPoolSize
	}
	if yCfg.MinIntervalS != 0 {
		cfg.MinIntervalS = yCfg.MinIntervalS
	}
	if yCfg.ConfURL != "" {
		cfg.ConfURL = yCfg.ConfURL
	}
	if yCfg.ConfPath != "" {
		cfg.ConfPath = yCfg.ConfPath
	}
	if yCfg.EnableCustomCSS != nil {
		cfg.EnableCustomCSS = *yCfg.EnableCustomCSS
	}
	if len(yCfg.TrustedProxyCIDRs) > 0 {
		cfg.TrustedProxyCIDRs = append([]string(nil), yCfg.TrustedProxyCIDRs...)
	}
	if yCfg.MetricsToken != "" {
		cfg.MetricsToken = yCfg.MetricsToken
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("UPDU_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("UPDU_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Port = n
		}
	}
	if v := os.Getenv("UPDU_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("UPDU_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("UPDU_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("UPDU_AUTH_SECRET"); v != "" {
		cfg.AuthSecret = v
	}
	if v := os.Getenv("UPDU_SESSION_TTL_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.SessionTTLDays = n
		}
	}
	if v := os.Getenv("UPDU_ADMIN_USER"); v != "" {
		cfg.AdminUser = v
	}
	if v := os.Getenv("UPDU_ADMIN_PASSWORD"); v != "" {
		cfg.AdminPassword = v
	}
	if v := os.Getenv("UPDU_PASSWORD_POLICY"); v != "" {
		cfg.PasswordPolicy = NormalizePasswordPolicy(v)
	}
	if v := os.Getenv("UPDU_OIDC_ISSUER"); v != "" {
		cfg.OIDCIssuer = v
	}
	if v := os.Getenv("UPDU_OIDC_CLIENT_ID"); v != "" {
		cfg.OIDCClientID = v
	}
	if v := os.Getenv("UPDU_OIDC_CLIENT_SECRET"); v != "" {
		cfg.OIDCClientSecret = v
	}
	if v := os.Getenv("UPDU_OIDC_REDIRECT_URL"); v != "" {
		cfg.OIDCRedirectURL = v
	}
	if v := os.Getenv("UPDU_OIDC_AUTO_REGISTER"); v != "" {
		cfg.OIDCAutoRegister = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("UPDU_WORKER_POOL_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WorkerPoolSize = n
		}
	}
	if v := os.Getenv("UPDU_MIN_INTERVAL_S"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MinIntervalS = n
		}
	}
	if v := os.Getenv("UPDU_CONF_URL"); v != "" && cfg.ConfURL == "" {
		cfg.ConfURL = v
	}
	if v := os.Getenv("UPDU_CONF_PATH"); v != "" && cfg.ConfPath == "" {
		cfg.ConfPath = v
	}
	if v := os.Getenv("UPDU_ENABLE_CUSTOM_CSS"); v != "" {
		cfg.EnableCustomCSS = strings.ToLower(v) == "true" || v == "1" || v == "yes"
	}
	if v := os.Getenv("UPDU_TRUSTED_PROXY_CIDRS"); v != "" {
		cfg.TrustedProxyCIDRs = envCSV("UPDU_TRUSTED_PROXY_CIDRS")
	}
	if v := os.Getenv("UPDU_METRICS_TOKEN"); v != "" {
		cfg.MetricsToken = v
	}
}

func discoverConfigPath() string {
	// 1. Check current directory
	if _, err := os.Stat("updu.conf"); err == nil {
		return "updu.conf"
	}

	// 2. Check UPDU_CONFIG_PATH (priority env var)
	if p := os.Getenv("UPDU_CONFIG_PATH"); p != "" {
		if stat, err := os.Stat(p); err == nil {
			if !stat.IsDir() {
				return p
			}
			// If it's a directory, check for updu.conf inside it
			joined := filepath.Join(p, "updu.conf")
			if _, err := os.Stat(joined); err == nil {
				return joined
			}
		}
	}

	// 3. Check UPDU_BASE_PATH
	if p := os.Getenv("UPDU_BASE_PATH"); p != "" {
		joined := filepath.Join(p, "updu.conf")
		if _, err := os.Stat(joined); err == nil {
			return joined
		}
	}

	return ""
}

// IsSecure returns true when the base URL uses HTTPS.
func (c *Config) IsSecure() bool {
	return strings.HasPrefix(c.BaseURL, "https://")
}

// IsTrustedProxy reports whether remoteAddr belongs to an explicitly trusted reverse proxy.
func (c *Config) IsTrustedProxy(remoteAddr string) bool {
	if len(c.TrustedProxyCIDRs) == 0 {
		return false
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil {
		remoteAddr = host
	}

	ip := net.ParseIP(strings.TrimSpace(remoteAddr))
	if ip == nil {
		return false
	}

	return c.IsTrustedProxyIP(ip)
}

// IsTrustedProxyIP reports whether ip belongs to an explicitly trusted reverse proxy.
func (c *Config) IsTrustedProxyIP(ip net.IP) bool {
	if len(c.TrustedProxyCIDRs) == 0 || ip == nil {
		return false
	}

	for _, candidate := range c.TrustedProxyCIDRs {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if exact := net.ParseIP(candidate); exact != nil && exact.Equal(ip) {
			return true
		}
		if _, network, parseErr := net.ParseCIDR(candidate); parseErr == nil && network.Contains(ip) {
			return true
		}
	}

	return false
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

func envCSV(key string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func NormalizePasswordPolicy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", PasswordPolicyDefault:
		return PasswordPolicyDefault
	case PasswordPolicyOff:
		return PasswordPolicyOff
	case PasswordPolicyStrong:
		return PasswordPolicyStrong
	case PasswordPolicyVerySecure, "very-secure", "verysecure", "very secure":
		return PasswordPolicyVerySecure
	default:
		return PasswordPolicyDefault
	}
}
