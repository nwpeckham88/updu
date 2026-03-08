//go:build oidc

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/config"
	"github.com/updu/updu/internal/notifier"
	"github.com/updu/updu/internal/realtime"
	"github.com/updu/updu/internal/scheduler"
	"github.com/updu/updu/internal/storage"
)

func setupOIDCAuthTest(t *testing.T, configured bool) (*Server, *storage.DB, func()) {
	tempDir, err := os.MkdirTemp("", "updu-api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	cfg := &config.Config{
		SessionTTLDays: 7,
	}

	if configured {
		cfg.OIDCIssuer = "https://mock.issuer.com"
		cfg.OIDCClientID = "mock-client"
		cfg.OIDCClientSecret = "mock-secret"
		cfg.OIDCRedirectURL = "http://localhost/callback"
	}

	a := auth.New(db, cfg)
	reg := checker.NewRegistry()
	sse := realtime.NewHub()
	n := notifier.New(db)
	sched := scheduler.New(db, reg, sse, n, 5)
	sched.DisableStagger = true

	srv := NewServer(db, a, reg, sched, n, sse, cfg)

	cleanup := func() {
		sched.Stop()
		db.Close()
		os.RemoveAll(tempDir)
	}

	return srv, db, cleanup
}

func TestAPI_OIDC_ProvidersEndpoint(t *testing.T) {
	// 1. Without OIDC configured
	srv, _, cleanup := setupOIDCAuthTest(t, false)
	req := httptest.NewRequest("GET", "/api/v1/auth/providers", nil)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp map[string]bool
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["oidc"] != false {
		t.Errorf("expected oidc=false, got %v", resp["oidc"])
	}
	cleanup()

	// 2. With OIDC configured
	srv, _, cleanup = setupOIDCAuthTest(t, true)
	defer cleanup()

	req = httptest.NewRequest("GET", "/api/v1/auth/providers", nil)
	rr = httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["oidc"] != true {
		t.Errorf("expected oidc=true, got %v", resp["oidc"])
	}
}

func TestAPI_OIDC_Login_NotConfigured(t *testing.T) {
	srv, _, cleanup := setupOIDCAuthTest(t, false)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/auth/oidc/login", nil)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	// Should 404 since it's not fully configured
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestAPI_OIDC_Callback_NotConfigured(t *testing.T) {
	srv, _, cleanup := setupOIDCAuthTest(t, false)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/auth/oidc/callback", nil)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestAPI_OIDC_Login_Initialization(t *testing.T) {
	// Initialize with fake values, which will trigger an error from go-oidc when it
	// tries to contact the discovery endpoint
	// https://mock.issuer.com/.well-known/openid-configuration
	srv, _, cleanup := setupOIDCAuthTest(t, true)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/auth/oidc/login", nil)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	// Will fail to initialize the OIDC provider (connection refused/dns error since mock.issuer.com doesn't exist)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when issuer fails discovery, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "error") {
		t.Errorf("expected error message, got %s", rr.Body.String())
	}
}

func TestAPI_OIDC_Callback_MissingStateCookie(t *testing.T) {
	srv, _, cleanup := setupOIDCAuthTest(t, true)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/auth/oidc/callback?state=somestate&code=xyz", nil)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when missing state cookie, got %d", rr.Code)
	}
}

func TestAPI_OIDC_Callback_StateMismatch(t *testing.T) {
	srv, _, cleanup := setupOIDCAuthTest(t, true)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/auth/oidc/callback?state=wrongstate&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: "oidc_state", Value: "correctstate"})
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for state mismatch, got %d", rr.Code)
	}
}
