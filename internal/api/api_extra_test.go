package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPI_MonitorGaps(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()
	router := srv.Router()

	// Setup: Need one admin user for authed routes
	ctx := context.Background()
	_, err := srv.auth.Register(ctx, "admin123", "password123")
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	session, _ := srv.auth.Login(ctx, "admin123", "password123", "agent", "1.2.3.4")
	cookie := &http.Cookie{Name: "updu_session", Value: session.ID}

	// 1. List Monitors (Empty)
	req := httptest.NewRequest("GET", "/api/v1/monitors", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// 2. Health Check
	req = httptest.NewRequest("GET", "/api/v1/system/health", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// 3. Setup Check
	req = httptest.NewRequest("GET", "/api/v1/auth/setup", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var setupResp struct {
		SetupRequired bool `json:"setup_required"`
	}
	json.NewDecoder(rr.Body).Decode(&setupResp)
	if setupResp.SetupRequired {
		t.Error("expected setup required to be false after registering admin")
	}

	// 4. Auth Providers
	req = httptest.NewRequest("GET", "/api/v1/auth/providers", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// 5. Test maxBody middleware (implicitly via any POST route)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"a", "password":"b"}`))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req) // Should just be unauthorized/bad request but not crash
}

func TestAPI_SetupCheck_Initial(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()
	router := srv.Router()

	req := httptest.NewRequest("GET", "/api/v1/auth/setup", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var setupResp struct {
		SetupRequired bool `json:"setup_required"`
	}
	json.NewDecoder(rr.Body).Decode(&setupResp)
	if !setupResp.SetupRequired {
		t.Error("expected setup required to be true initially")
	}
}
