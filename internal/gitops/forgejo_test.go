package gitops

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/config"
	"github.com/updu/updu/internal/storage"
)

func TestVerifySignature(t *testing.T) {
	secret := "super-secret-hmac-key"
	payload := []byte(`{"ref":"refs/heads/main","repository":{"name":"homelab-monitors"}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := hex.EncodeToString(mac.Sum(nil))

	// 1. Valid HMAC
	if !VerifySignature(payload, secret, validSig, "") {
		t.Error("expected valid signature to pass")
	}

	// 2. Invalid HMAC
	if VerifySignature(payload, secret, "invalid_hex", "") {
		t.Error("expected invalid signature to fail")
	}

	// 3. Valid token header
	if !VerifySignature(payload, secret, "", secret) {
		t.Error("expected valid token header to pass")
	}

	// 4. Invalid token header
	if VerifySignature(payload, secret, "", "wrong-token") {
		t.Error("expected wrong token to fail")
	}
}

func TestResolveRawURL(t *testing.T) {
	cfg := &config.GitOpsConfig{
		ServerURL:  "https://git.kn8design.com",
		Repository: "nathan/homelab-monitors",
		Branch:     "main",
		Path:       "updu.conf",
	}

	rawURL, err := ResolveRawURL(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://git.kn8design.com/api/v1/repos/nathan/homelab-monitors/raw/updu.conf?ref=main"
	if rawURL != expected {
		t.Errorf("expected %q, got %q", expected, rawURL)
	}

	// Explicit override
	cfg.RawURL = "https://custom.example/raw.yaml"
	rawURL, _ = ResolveRawURL(cfg)
	if rawURL != "https://custom.example/raw.yaml" {
		t.Errorf("expected custom URL override, got %q", rawURL)
	}
}

func TestSyncConfig(t *testing.T) {
	tempDir := t.TempDir()
	db, err := storage.Open(filepath.Join(tempDir, "gitops_test.db"))
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	yamlContent := `
monitors:
  - id: gitops-svc
    name: GitOps Managed Service
    type: http
    interval: 30s
    groups: [family]
    config:
      url: https://example.com

services:
  - id: gitops-app
    name: GitOps App
    type: web
    groups: [infra]
    endpoints:
      - name: Primary Web
        type: http
        config:
          url: https://example.com/app
`

	reg := checker.NewRegistry(true)
	res, err := SyncConfig(ctx, []byte(yamlContent), reg, db, nil)
	if err != nil {
		t.Fatalf("failed to sync config: %v", err)
	}

	if res.MonitorsSynced != 1 {
		t.Errorf("expected 1 monitor synced, got %d", res.MonitorsSynced)
	}
	if res.ServicesSynced != 1 {
		t.Errorf("expected 1 service synced, got %d", res.ServicesSynced)
	}

	// Verify in DB
	m, err := db.GetMonitor(ctx, "gitops-svc")
	if err != nil || m == nil {
		t.Fatalf("expected monitor to exist in DB: %v", err)
	}
	if m.Name != "GitOps Managed Service" || len(m.Groups) != 1 || m.Groups[0] != "family" {
		t.Errorf("unexpected monitor in DB: %+v", m)
	}
}
