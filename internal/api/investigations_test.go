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

func TestMonitorInvestigationEndpoint(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	cookie := registerAndLoginAdmin(t, router)
	monitor := &models.Monitor{
		ID:        "monitor-investigation",
		Name:      "Investigation Target",
		Type:      "http",
		Config:    json.RawMessage(`{"url":"https://example.com"}`),
		Groups:    []string{"Core"},
		IntervalS: 60,
		TimeoutS:  10,
		Retries:   1,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.CreateMonitor(context.Background(), monitor); err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	setReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors/monitor-investigation/investigate",
		bytes.NewBufferString(`{"investigating":true}`),
	)
	setReq.AddCookie(cookie)
	setResp := httptest.NewRecorder()
	router.ServeHTTP(setResp, setReq)

	if setResp.Code != http.StatusOK {
		t.Fatalf("expected set investigation status 200, got %d: %s", setResp.Code, setResp.Body.String())
	}

	var setPayload map[string]any
	if err := json.NewDecoder(setResp.Body).Decode(&setPayload); err != nil {
		t.Fatalf("failed to decode set response: %v", err)
	}
	if setPayload["monitor_id"] != "monitor-investigation" || setPayload["active"] != true {
		t.Fatalf("unexpected set response: %#v", setPayload)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/monitor-investigation", nil)
	getReq.AddCookie(cookie)
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected get monitor status 200, got %d: %s", getResp.Code, getResp.Body.String())
	}

	var monitorPayload map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&monitorPayload); err != nil {
		t.Fatalf("failed to decode monitor response: %v", err)
	}
	investigation, ok := monitorPayload["investigation"].(map[string]any)
	if !ok {
		t.Fatalf("expected monitor response to include investigation, got %#v", monitorPayload["investigation"])
	}
	if investigation["active"] != true || investigation["monitor_id"] != "monitor-investigation" {
		t.Fatalf("unexpected monitor investigation payload: %#v", investigation)
	}

	clearReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors/monitor-investigation/investigate",
		bytes.NewBufferString(`{"investigating":false}`),
	)
	clearReq.AddCookie(cookie)
	clearResp := httptest.NewRecorder()
	router.ServeHTTP(clearResp, clearReq)
	if clearResp.Code != http.StatusOK {
		t.Fatalf("expected clear investigation status 200, got %d: %s", clearResp.Code, clearResp.Body.String())
	}

	getReq = httptest.NewRequest(http.MethodGet, "/api/v1/monitors/monitor-investigation", nil)
	getReq.AddCookie(cookie)
	getResp = httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected get monitor status 200 after clear, got %d: %s", getResp.Code, getResp.Body.String())
	}
	monitorPayload = map[string]any{}
	if err := json.NewDecoder(getResp.Body).Decode(&monitorPayload); err != nil {
		t.Fatalf("failed to decode monitor response after clear: %v", err)
	}
	if _, exists := monitorPayload["investigation"]; exists {
		t.Fatalf("expected investigation to be omitted after clear, got %#v", monitorPayload["investigation"])
	}
}

func TestMonitorInvestigationEndpointRequiresAdmin(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	_, viewerCookie := setupAdminAndViewer(t, srv)

	monitor := &models.Monitor{
		ID:        "monitor-viewer-investigation",
		Name:      "Viewer Investigation Target",
		Type:      "http",
		Config:    json.RawMessage(`{"url":"https://example.com"}`),
		Groups:    []string{"Core"},
		IntervalS: 60,
		TimeoutS:  10,
		Retries:   1,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.CreateMonitor(context.Background(), monitor); err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors/monitor-viewer-investigation/investigate",
		bytes.NewBufferString(`{"investigating":true}`),
	)
	req.AddCookie(viewerCookie)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected viewer investigation status 403, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestMonitorInvestigationHiddenFromViewerResponses(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	adminCookie, viewerCookie := setupAdminAndViewer(t, srv)

	monitor := &models.Monitor{
		ID:        "monitor-hidden-investigation",
		Name:      "Hidden Investigation Target",
		Type:      "http",
		Config:    json.RawMessage(`{"url":"https://example.com"}`),
		Groups:    []string{"Core"},
		IntervalS: 60,
		TimeoutS:  10,
		Retries:   1,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.CreateMonitor(context.Background(), monitor); err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	setReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors/monitor-hidden-investigation/investigate",
		bytes.NewBufferString(`{"investigating":true}`),
	)
	setReq.AddCookie(adminCookie)
	setResp := httptest.NewRecorder()
	router.ServeHTTP(setResp, setReq)
	if setResp.Code != http.StatusOK {
		t.Fatalf("expected admin investigation status 200, got %d: %s", setResp.Code, setResp.Body.String())
	}

	adminGetReq := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/monitor-hidden-investigation", nil)
	adminGetReq.AddCookie(adminCookie)
	adminGetResp := httptest.NewRecorder()
	router.ServeHTTP(adminGetResp, adminGetReq)
	var adminPayload map[string]any
	if err := json.NewDecoder(adminGetResp.Body).Decode(&adminPayload); err != nil {
		t.Fatalf("failed to decode admin monitor response: %v", err)
	}
	if _, exists := adminPayload["investigation"]; !exists {
		t.Fatalf("expected admin response to include investigation")
	}

	viewerGetReq := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/monitor-hidden-investigation", nil)
	viewerGetReq.AddCookie(viewerCookie)
	viewerGetResp := httptest.NewRecorder()
	router.ServeHTTP(viewerGetResp, viewerGetReq)
	var viewerPayload map[string]any
	if err := json.NewDecoder(viewerGetResp.Body).Decode(&viewerPayload); err != nil {
		t.Fatalf("failed to decode viewer monitor response: %v", err)
	}
	if _, exists := viewerPayload["investigation"]; exists {
		t.Fatalf("expected viewer response to omit investigation, got %#v", viewerPayload["investigation"])
	}
}

func TestCreateMonitorEnabledDefaults(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	cookie := registerAndLoginAdmin(t, router)

	tests := []struct {
		name        string
		enabled     *bool
		wantEnabled bool
	}{
		{name: "omitted enabled defaults disabled", wantEnabled: false},
		{name: "explicit disabled remains disabled", enabled: ptr(false), wantEnabled: false},
		{name: "explicit enabled remains enabled", enabled: ptr(true), wantEnabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]any{
				"name":       tt.name,
				"type":       "http",
				"groups":     []string{"Core"},
				"interval_s": 60,
				"config":     map[string]any{"url": "https://example.com"},
			}
			if tt.enabled != nil {
				payload["enabled"] = *tt.enabled
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("failed to marshal monitor payload: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/monitors", bytes.NewBuffer(body))
			req.AddCookie(cookie)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusCreated {
				t.Fatalf("expected create monitor status 201, got %d: %s", resp.Code, resp.Body.String())
			}

			var created map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
				t.Fatalf("failed to decode monitor response: %v", err)
			}
			if created["enabled"] != tt.wantEnabled {
				t.Fatalf("expected monitor enabled=%v, got %#v", tt.wantEnabled, created["enabled"])
			}
		})
	}
}

func ptr[T any](value T) *T {
	return &value
}
