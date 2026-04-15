package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvHelpers(t *testing.T) {
	os.Setenv("TEST_STRING", "hello")
	if envOr("TEST_STRING", "fallback") != "hello" {
		t.Error("envOr failed for string")
	}
	if envOr("NONEXISTENT", "fallback") != "fallback" {
		t.Error("envOr failed for fallback")
	}

	os.Setenv("TEST_INT", "123")
	if envInt("TEST_INT", 0) != 123 {
		t.Error("envInt failed for int")
	}
	os.Setenv("TEST_INT", "invalid")
	if envInt("TEST_INT", 456) != 456 {
		t.Error("envInt failed for fallback")
	}

	os.Setenv("TEST_BOOL", "true")
	if !envBool("TEST_BOOL", false) {
		t.Error("envBool failed for true")
	}
	os.Setenv("TEST_BOOL", "1")
	if !envBool("TEST_BOOL", false) {
		t.Error("envBool failed for 1")
	}
	os.Setenv("TEST_BOOL", "yes")
	if !envBool("TEST_BOOL", false) {
		t.Error("envBool failed for yes")
	}
	os.Setenv("TEST_BOOL", "false")
	if envBool("TEST_BOOL", true) {
		t.Error("envBool failed for false")
	}
}

func TestDiscoverConfigPath(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp dir to test relative path
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// 1. Check current directory
	os.WriteFile("updu.conf", []byte(""), 0644)
	if p := discoverConfigPath(); p != "updu.conf" {
		t.Errorf("expected updu.conf, got %s", p)
	}
	os.Remove("updu.conf")

	// 2. Check UPDU_CONFIG_PATH
	confPath := filepath.Join(tempDir, "custom.conf")
	os.WriteFile(confPath, []byte(""), 0644)
	os.Setenv("UPDU_CONFIG_PATH", confPath)
	if p := discoverConfigPath(); p != confPath {
		t.Errorf("expected %s, got %s", confPath, p)
	}

	// 2b. Check UPDU_CONFIG_PATH as directory
	os.Remove(confPath)
	os.Mkdir(confPath, 0755)
	innerPath := filepath.Join(confPath, "updu.conf")
	os.WriteFile(innerPath, []byte(""), 0644)
	if p := discoverConfigPath(); p != innerPath {
		t.Errorf("expected inner updu.conf, got %s", p)
	}
	os.Setenv("UPDU_CONFIG_PATH", "")

	// 3. Check UPDU_BASE_PATH
	os.Setenv("UPDU_BASE_PATH", confPath)
	if p := discoverConfigPath(); p != innerPath {
		t.Errorf("expected base_path/updu.conf, got %s", p)
	}
	os.Setenv("UPDU_BASE_PATH", "")
}

func TestApplyEnvOverrides(t *testing.T) {
	cfg := &Config{}
	os.Setenv("UPDU_HOST", "1.1.1.1")
	os.Setenv("UPDU_PORT", "9999")
	os.Setenv("UPDU_SESSION_TTL_DAYS", "30")
	os.Setenv("UPDU_OIDC_AUTO_REGISTER", "true")
	os.Setenv("UPDU_WORKER_POOL_SIZE", "10")
	os.Setenv("UPDU_MIN_INTERVAL_S", "15")
	os.Setenv("UPDU_TRUSTED_PROXY_CIDRS", "127.0.0.1/32, 10.0.0.0/8")

	applyEnvOverrides(cfg)

	if cfg.Host != "1.1.1.1" {
		t.Errorf("expected 1.1.1.1, got %s", cfg.Host)
	}
	if cfg.Port != 9999 {
		t.Errorf("expected 9999, got %d", cfg.Port)
	}
	if cfg.SessionTTLDays != 30 {
		t.Errorf("expected 30, got %d", cfg.SessionTTLDays)
	}
	if !cfg.OIDCAutoRegister {
		t.Error("expected OIDCAutoRegister true")
	}
	if cfg.WorkerPoolSize != 10 {
		t.Errorf("expected 10, got %d", cfg.WorkerPoolSize)
	}
	if cfg.MinIntervalS != 15 {
		t.Errorf("expected 15, got %d", cfg.MinIntervalS)
	}
	if len(cfg.TrustedProxyCIDRs) != 2 || cfg.TrustedProxyCIDRs[0] != "127.0.0.1/32" || cfg.TrustedProxyCIDRs[1] != "10.0.0.0/8" {
		t.Fatalf("unexpected trusted proxy cidrs: %#v", cfg.TrustedProxyCIDRs)
	}
}

func TestApplyYAML(t *testing.T) {
	cfg := &Config{}

	// Helper to create *bool
	truePtr := true

	yCfg := &YAMLConfig{
		Host:              "yaml-host",
		Port:              8888,
		BaseURL:           "https://yaml.local",
		DBPath:            "/path/to/yaml.db",
		LogLevel:          "debug",
		AuthSecret:        "yaml-secret",
		SessionTTLDays:    14,
		AdminUser:         "yaml-admin",
		AdminPassword:     "yaml-password",
		OIDCIssuer:        "https://yaml-oidc",
		OIDCClientID:      "yaml-client",
		OIDCClientSecret:  "yaml-secret",
		OIDCRedirectURL:   "https://yaml-redirect",
		OIDCAutoRegister:  &truePtr,
		WorkerPoolSize:    20,
		MinIntervalS:      60,
		TrustedProxyCIDRs: []string{"127.0.0.1/32", "10.0.0.0/8"},
		ConfURL:           "https://yaml-conf",
		ConfPath:          "yaml-conf-path",
	}

	applyYAML(cfg, yCfg)

	if cfg.Host != "yaml-host" {
		t.Error("Host mismatch")
	}
	if cfg.Port != 8888 {
		t.Error("Port mismatch")
	}
	if cfg.BaseURL != "https://yaml.local" {
		t.Error("BaseURL mismatch")
	}
	if cfg.DBPath != "/path/to/yaml.db" {
		t.Error("DBPath mismatch")
	}
	if cfg.LogLevel != "debug" {
		t.Error("LogLevel mismatch")
	}
	if cfg.AuthSecret != "yaml-secret" {
		t.Error("AuthSecret mismatch")
	}
	if cfg.SessionTTLDays != 14 {
		t.Error("SessionTTLDays mismatch")
	}
	if cfg.AdminUser != "yaml-admin" {
		t.Error("AdminUser mismatch")
	}
	if cfg.AdminPassword != "yaml-password" {
		t.Error("AdminPassword mismatch")
	}
	if cfg.OIDCIssuer != "https://yaml-oidc" {
		t.Error("OIDCIssuer mismatch")
	}
	if cfg.OIDCClientID != "yaml-client" {
		t.Error("OIDCClientID mismatch")
	}
	if cfg.OIDCClientSecret != "yaml-secret" {
		t.Error("OIDCClientSecret mismatch")
	}
	if cfg.OIDCRedirectURL != "https://yaml-redirect" {
		t.Error("OIDCRedirectURL mismatch")
	}
	if !cfg.OIDCAutoRegister {
		t.Error("OIDCAutoRegister mismatch")
	}
	if cfg.WorkerPoolSize != 20 {
		t.Error("WorkerPoolSize mismatch")
	}
	if cfg.MinIntervalS != 60 {
		t.Error("MinIntervalS mismatch")
	}
	if len(cfg.TrustedProxyCIDRs) != 2 || cfg.TrustedProxyCIDRs[0] != "127.0.0.1/32" || cfg.TrustedProxyCIDRs[1] != "10.0.0.0/8" {
		t.Error("TrustedProxyCIDRs mismatch")
	}
	if cfg.ConfURL != "https://yaml-conf" {
		t.Error("ConfURL mismatch")
	}
	if cfg.ConfPath != "yaml-conf-path" {
		t.Error("ConfPath mismatch")
	}
}

func TestIsTrustedProxy(t *testing.T) {
	cfg := &Config{TrustedProxyCIDRs: []string{"127.0.0.1/32", "10.0.0.0/8", "::1/128"}}

	if !cfg.IsTrustedProxy("127.0.0.1:8080") {
		t.Fatal("expected loopback proxy to be trusted")
	}
	if !cfg.IsTrustedProxy("10.2.3.4:8080") {
		t.Fatal("expected RFC1918 proxy to be trusted")
	}
	if !cfg.IsTrustedProxy("[::1]:8080") {
		t.Fatal("expected IPv6 loopback proxy to be trusted")
	}
	if cfg.IsTrustedProxy("192.168.1.10:8080") {
		t.Fatal("unexpected trust for proxy outside configured ranges")
	}
}
