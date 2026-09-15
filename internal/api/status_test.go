package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestStatusAPI(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	adminCookie := registerAndLoginAdmin(t, router)

	ctx := t.Context()
	// Create two monitors: one family, one admin-only
	mFamily := &models.Monitor{
		ID:        "mon-jellyfin",
		Name:      "Jellyfin Media",
		Type:      "http",
		Config:    json.RawMessage(`{"url":"https://jellyfin.example.com"}`),
		Groups:    []string{"family"},
		IntervalS: 60,
		TimeoutS:  10,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mAdmin := &models.Monitor{
		ID:        "mon-adguard",
		Name:      "AdGuard DNS",
		Type:      "http",
		Config:    json.RawMessage(`{"url":"https://dns.example.com"}`),
		Groups:    []string{"admin"},
		IntervalS: 60,
		TimeoutS:  10,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.CreateMonitor(ctx, mFamily); err != nil {
		t.Fatalf("failed to create family monitor: %v", err)
	}
	if err := db.CreateMonitor(ctx, mAdmin); err != nil {
		t.Fatalf("failed to create admin monitor: %v", err)
	}

	// Insert check results
	lat := 14
	_ = db.InsertCheckResult(ctx, &models.CheckResult{
		MonitorID: mFamily.ID,
		Status:    models.StatusUp,
		LatencyMs: &lat,
		Message:   "HTTP 200 OK",
		CheckedAt: time.Now(),
	})

	// 1. Query single status by ID as admin
	req := httptest.NewRequest("GET", "/api/v1/status/mon-jellyfin", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var single SingleStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &single); err != nil {
		t.Fatalf("failed to unmarshal status response: %v", err)
	}
	if single.Name != "Jellyfin Media" || single.Status != "up" || *single.LatencyMs != 14 {
		t.Errorf("unexpected status response: %+v", single)
	}

	// 2. Query single status by case-insensitive name
	req = httptest.NewRequest("GET", "/api/v1/status/Jellyfin%20Media", nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 by name, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Batch query as admin (should see both)
	req = httptest.NewRequest("GET", "/api/v1/status", nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var batch BatchStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &batch); err != nil {
		t.Fatalf("failed to unmarshal batch response: %v", err)
	}
	if len(batch.Services) != 2 {
		t.Errorf("expected 2 services for admin, got %d", len(batch.Services))
	}

	// 4. Batch query with ?group=family
	req = httptest.NewRequest("GET", "/api/v1/status?group=family", nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &batch); err != nil {
		t.Fatalf("failed to unmarshal batch response: %v", err)
	}
	if len(batch.Services) != 1 || batch.Services[0].ID != "mon-jellyfin" {
		t.Errorf("expected 1 family service, got %+v", batch.Services)
	}

	// 5. Test SVG Badge endpoint
	req = httptest.NewRequest("GET", "/api/v1/status/mon-jellyfin/badge", nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for badge, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<svg") || !strings.Contains(rec.Body.String(), "up") {
		t.Errorf("expected svg containing 'up', got %s", rec.Body.String())
	}

	// 6. Test with API Token via query param ?token=...
	tokenReq := struct {
		Name  string               `json:"name"`
		Scope models.APITokenScope `json:"scope"`
	}{Name: "bento-key", Scope: models.APITokenScopeRead}
	tokenBody, _ := json.Marshal(tokenReq)
	req = httptest.NewRequest("POST", "/api/v1/admin/api-tokens", bytes.NewReader(tokenBody))
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to create API token: %d %s", rec.Code, rec.Body.String())
	}
	var createdToken models.APITokenSecret
	_ = json.Unmarshal(rec.Body.Bytes(), &createdToken)

	// Now query status with query parameter ?token=... (no cookies)
	req = httptest.NewRequest("GET", "/api/v1/status/mon-jellyfin?token="+createdToken.Token, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with query token, got %d: %s", rec.Code, rec.Body.String())
	}
}
