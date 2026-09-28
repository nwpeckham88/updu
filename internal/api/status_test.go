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

	// 7. Verify built-in WAN zone exists alongside default zone
	req = httptest.NewRequest("GET", "/api/v1/zones", nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/zones, got %d: %s", rec.Code, rec.Body.String())
	}
	var zones []*models.Zone
	if err := json.Unmarshal(rec.Body.Bytes(), &zones); err != nil {
		t.Fatalf("failed to decode zones: %v", err)
	}
	hasWAN := false
	for _, z := range zones {
		if z.ID == "wan" {
			hasWAN = true
			break
		}
	}
	if !hasWAN {
		t.Errorf("expected default 'wan' zone to be present in /api/v1/zones, got: %+v", zones)
	}

	// 8. Test Status Page with active incident and active maintenance
	sp := &models.StatusPage{
		ID:       "sp-main",
		Name:     "Public Status",
		Slug:     "public-status",
		IsPublic: true,
		Groups: []models.StatusPageGroup{
			{Name: "family", MonitorIDs: []string{mFamily.ID}},
		},
	}
	if err := db.CreateStatusPage(ctx, sp); err != nil {
		t.Fatalf("failed to create status page: %v", err)
	}

	// Add an incident
	inc := &models.Incident{
		ID:         "inc-1",
		Title:      "WAN Ingress Glitch",
		Status:     models.IncidentInvestigating,
		Severity:   "major",
		MonitorIDs: []string{mFamily.ID},
		StartedAt:  time.Now(),
		CreatedBy:  "admin",
	}
	if err := db.CreateIncident(ctx, inc); err != nil {
		t.Fatalf("failed to create incident: %v", err)
	}

	// Add maintenance
	mw := &models.MaintenanceWindow{
		ID:         "mw-1",
		Title:      "Router Firmware Upgrade",
		MonitorIDs: []string{mFamily.ID},
		StartsAt:   time.Now().Add(-10 * time.Minute),
		EndsAt:     time.Now().Add(50 * time.Minute),
		CreatedBy:  "admin",
	}
	if err := db.CreateMaintenanceWindow(ctx, mw); err != nil {
		t.Fatalf("failed to create maintenance window: %v", err)
	}

	req = httptest.NewRequest("GET", "/api/v1/status-pages/public-status", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for public status page, got %d: %s", rec.Code, rec.Body.String())
	}
	var spResp struct {
		Page        models.StatusPage           `json:"page"`
		Monitors    []map[string]any            `json:"monitors"`
		Incidents   []*models.Incident          `json:"incidents"`
		Maintenance []*models.MaintenanceWindow `json:"maintenance"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spResp); err != nil {
		t.Fatalf("failed to unmarshal status page response: %v", err)
	}
	if len(spResp.Incidents) != 1 || spResp.Incidents[0].ID != "inc-1" {
		t.Errorf("expected 1 incident in status page response, got %d", len(spResp.Incidents))
	}
	if len(spResp.Maintenance) != 1 || spResp.Maintenance[0].ID != "mw-1" {
		t.Errorf("expected 1 active maintenance in status page response, got %d", len(spResp.Maintenance))
	}
}
