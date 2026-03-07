package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPI_MonitorGaps(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()
	router := srv.Router()

	// 1. List Monitors (Empty)
	req := httptest.NewRequest("GET", "/api/v1/monitors", nil)
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
	if !setupResp.SetupRequired {
		t.Error("expected setup required to be true initially")
	}
}
