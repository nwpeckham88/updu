package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Test defaults
	os.Clearenv()
	cfg := Load()
	if cfg.Port != 3000 {
		t.Errorf("expected default port 3000, got %d", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", cfg.Host)
	}
	if cfg.PasswordPolicy != PasswordPolicyDefault {
		t.Errorf("expected default password policy %q, got %q", PasswordPolicyDefault, cfg.PasswordPolicy)
	}

	// Test environment variables
	os.Setenv("UPDU_PORT", "4000")
	os.Setenv("UPDU_HOST", "127.0.0.1")
	os.Setenv("UPDU_DB_PATH", "/tmp/test.db")
	os.Setenv("UPDU_PASSWORD_POLICY", PasswordPolicyStrong)

	cfg = Load()
	if cfg.Port != 4000 {
		t.Errorf("expected port 4000, got %d", cfg.Port)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1, got %s", cfg.Host)
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("expected db path /tmp/test.db, got %s", cfg.DBPath)
	}
	if cfg.PasswordPolicy != PasswordPolicyStrong {
		t.Errorf("expected password policy %q, got %q", PasswordPolicyStrong, cfg.PasswordPolicy)
	}

	// Test invalid int fallback
	os.Setenv("UPDU_PORT", "invalid")
	os.Setenv("UPDU_PASSWORD_POLICY", "not-a-policy")
	cfg = Load()
	if cfg.Port != 3000 {
		t.Errorf("expected fallback port 3000 for invalid env, got %d", cfg.Port)
	}
	if cfg.PasswordPolicy != PasswordPolicyDefault {
		t.Errorf("expected invalid password policy to fall back to %q, got %q", PasswordPolicyDefault, cfg.PasswordPolicy)
	}
}

func TestIsSecure(t *testing.T) {
	tests := []struct {
		baseURL  string
		expected bool
	}{
		{"https://example.com", true},
		{"https://monitor.local:8443", true},
		{"http://localhost:3000", false},
		{"http://example.com", false},
		{"", false},
	}

	for _, tc := range tests {
		cfg := &Config{BaseURL: tc.baseURL}
		if got := cfg.IsSecure(); got != tc.expected {
			t.Errorf("IsSecure(%q) = %v, want %v", tc.baseURL, got, tc.expected)
		}
	}
}

func TestAuthSecretAutoGenerate(t *testing.T) {
	os.Clearenv()
	cfg := Load()

	if cfg.AuthSecret == "" {
		t.Error("expected auto-generated auth secret, got empty")
	}
	if len(cfg.AuthSecret) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected 64 char hex secret, got %d chars", len(cfg.AuthSecret))
	}

	// When explicitly set, should use that value
	os.Setenv("UPDU_AUTH_SECRET", "my-custom-secret")
	cfg = Load()
	if cfg.AuthSecret != "my-custom-secret" {
		t.Errorf("expected custom secret, got %q", cfg.AuthSecret)
	}
}
