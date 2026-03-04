package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestAPI_SystemMetrics(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// 1. Setup Admin User and get session cookie
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 2. Create some monitors
	monReq1, _ := json.Marshal(map[string]any{
		"name": "M1", "type": "http", "config": map[string]string{"url": "http://a.com"}, "enabled": true,
	})
	req := httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(monReq1))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var m1 models.Monitor
	json.NewDecoder(rr.Body).Decode(&m1)

	monReq2, _ := json.Marshal(map[string]any{
		"name": "M2", "type": "ping", "config": map[string]string{"host": "8.8.8.8"}, "enabled": false, // paused
	})
	req = httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(monReq2))
	req.AddCookie(sessionCookie)
	rrM2 := httptest.NewRecorder()
	router.ServeHTTP(rrM2, req)
	if rrM2.Code != http.StatusCreated {
		t.Fatalf("failed creating m2: %v", rrM2.Body.String())
	}
	var m2 models.Monitor
	json.NewDecoder(rrM2.Body).Decode(&m2)
	_, _ = srv.db.ExecContext(context.Background(), "UPDATE monitors SET enabled = 0 WHERE id = ?", m2.ID)

	// 3. Insert fake check results in DB to simulate statuses
	latency := 45
	err := srv.db.InsertCheckResult(context.Background(), &models.CheckResult{
		MonitorID: m1.ID,
		Status:    models.StatusUp,
		LatencyMs: &latency,
		CheckedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("failed inserting fake result: %v", err)
	}

	// 4. Create a fake active incident
	incidentReq, _ := json.Marshal(map[string]any{
		"title": "Outage", "severity": "major", "status": "investigating", "monitor_ids": []string{m1.ID},
	})
	req = httptest.NewRequest("POST", "/api/v1/incidents", bytes.NewBuffer(incidentReq))
	req.AddCookie(sessionCookie)
	rrInc := httptest.NewRecorder()
	router.ServeHTTP(rrInc, req)
	if rrInc.Code != http.StatusCreated {
		t.Fatalf("failed creating incident, got %d", rrInc.Code)
	}

	// 5. Test Metrics Endpoint
	req = httptest.NewRequest("GET", "/api/v1/system/metrics", nil)
	req.AddCookie(sessionCookie)
	rrMetrics := httptest.NewRecorder()
	router.ServeHTTP(rrMetrics, req)

	if rrMetrics.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rrMetrics.Code)
	}

	var metrics models.SystemMetrics
	if err := json.NewDecoder(rrMetrics.Body).Decode(&metrics); err != nil {
		t.Fatalf("failed decoding metrics: %v", err)
	}

	if metrics.TotalMonitors != 2 {
		t.Errorf("expected 2 total monitors, got %d", metrics.TotalMonitors)
	}
	if metrics.MonitorsPaused != 1 {
		t.Errorf("expected 1 paused monitor, got %d", metrics.MonitorsPaused)
	}
	if metrics.MonitorsUp != 1 {
		t.Errorf("expected 1 up monitor, got %d", metrics.MonitorsUp)
	}
	if metrics.ActiveIncidents != 1 {
		t.Errorf("expected 1 active incident, got %d", metrics.ActiveIncidents)
	}
}
