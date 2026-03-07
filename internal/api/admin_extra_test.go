package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPI_AdminExtra(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// 1. Setup Admin
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 2. Notification Channels API
	ncBody, _ := json.Marshal(map[string]any{
		"name": "Slack", "type": "slack", "config": map[string]string{"url": "http://slack.com"}, "enabled": true,
	})
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBuffer(ncBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create channel, got %d: %s", rr.Code, rr.Body.String())
	}
	var nc struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rr.Body).Decode(&nc)

	// List Channels
	req = httptest.NewRequest("GET", "/api/v1/notifications", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for list channels, got %d", rr.Code)
	}

	// Update Channel
	ncUpBody, _ := json.Marshal(map[string]any{"name": "Slack Updated", "enabled": false})
	req = httptest.NewRequest("PUT", "/api/v1/notifications/"+nc.ID, bytes.NewBuffer(ncUpBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for update channel, got %d", rr.Code)
	}

	// Delete Channel
	req = httptest.NewRequest("DELETE", "/api/v1/notifications/"+nc.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for delete channel, got %d", rr.Code)
	}

	// 3. Status Pages API (Extra)
	spBody, _ := json.Marshal(map[string]any{
		"name": "Public Status", "slug": "public", "is_public": true, "monitors": []string{},
	})
	req = httptest.NewRequest("POST", "/api/v1/status-pages", bytes.NewBuffer(spBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create status page, got %d", rr.Code)
	}
	var sp struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rr.Body).Decode(&sp)

	// List Status Pages
	req = httptest.NewRequest("GET", "/api/v1/status-pages", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for list status pages, got %d", rr.Code)
	}

	// Update Status Page
	spUpBody, _ := json.Marshal(map[string]any{"name": "Private Status", "is_public": false})
	req = httptest.NewRequest("PUT", "/api/v1/status-pages/"+sp.ID, bytes.NewBuffer(spUpBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for update status page, got %d", rr.Code)
	}

	// Delete Status Page
	req = httptest.NewRequest("DELETE", "/api/v1/status-pages/"+sp.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for delete status page, got %d", rr.Code)
	}
}

func TestAPI_AdminErrorPaths(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()
	router := srv.Router()

	// Setup Admin
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Invalid JSON in POST
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBuffer([]byte("{invalid")))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code == http.StatusOK {
		t.Error("expected error for invalid JSON")
	}

	// 2. Missing fields in Monitor
	req = httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer([]byte(`{"name":""}`)))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code == http.StatusOK {
		t.Error("expected error for empty monitor name")
	}

	// 3. Non-existent IDs
	req = httptest.NewRequest("GET", "/api/v1/incidents/ghost", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for ghost incident, got %d", rr.Code)
	}
}
