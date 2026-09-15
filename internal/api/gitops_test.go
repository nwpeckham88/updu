package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitOpsWebhookAndAPI(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	// Configure GitOps in server config
	srv.config.GitOps.Enabled = true
	srv.config.GitOps.Provider = "forgejo"
	srv.config.GitOps.ServerURL = "https://git.kn8design.com"
	srv.config.GitOps.Repository = "nathan/homelab-monitors"
	srv.config.GitOps.Branch = "main"
	srv.config.GitOps.Path = "updu.conf"
	srv.config.GitOps.WebhookSecret = "forgejo-test-secret"

	router := srv.Router()
	adminCookie := registerAndLoginAdmin(t, router)

	// 1. Check GitOps status as admin
	req := httptest.NewRequest("GET", "/api/v1/gitops/status", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for gitops status, got %d: %s", rec.Code, rec.Body.String())
	}
	var statusResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("failed to unmarshal status: %v", err)
	}
	if statusResp["enabled"] != true || statusResp["repository"] != "nathan/homelab-monitors" {
		t.Errorf("unexpected gitops status: %+v", statusResp)
	}

	// 2. Webhook with invalid signature should fail 401
	payload := []byte(`{"ref":"refs/heads/main"}`)
	req = httptest.NewRequest("POST", "/api/v1/gitops/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Forgejo-Signature", "invalid-signature")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid webhook signature, got %d", rec.Code)
	}

	// 3. Webhook with valid signature for non-target branch should be ignored
	nonMainPayload := []byte(`{"ref":"refs/heads/dev"}`)
	mac := hmac.New(sha256.New, []byte("forgejo-test-secret"))
	mac.Write(nonMainPayload)
	sig := hex.EncodeToString(mac.Sum(nil))

	req = httptest.NewRequest("POST", "/api/v1/gitops/webhook", bytes.NewReader(nonMainPayload))
	req.Header.Set("X-Forgejo-Signature", sig)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (ignored), got %d: %s", rec.Code, rec.Body.String())
	}
	var ignoredResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &ignoredResp)
	if ignoredResp["ref"] != "refs/heads/dev" {
		t.Errorf("expected ignored message, got %+v", ignoredResp)
	}
}
