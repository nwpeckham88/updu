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

func TestAPI_EventsCRUD(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// Register and login to get admin access
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	if len(rrLogin.Result().Cookies()) == 0 {
		t.Fatalf("failed to login and get cookie")
	}
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Create a monitor first
	monBody, _ := json.Marshal(map[string]any{
		"name":       "Event Test",
		"type":       "http",
		"config":     map[string]string{"url": "https://example.com"},
		"interval_s": 60,
	})
	req := httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(monBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status Created, got %d", rr.Code)
	}

	var createdMonitor models.Monitor
	if err := json.NewDecoder(rr.Body).Decode(&createdMonitor); err != nil {
		t.Fatalf("failed to decode monitor")
	}

	// 2. Insert some events directly to DB
	e1 := &models.Event{
		MonitorID: createdMonitor.ID,
		Status:    models.StatusDown,
		Message:   "Offline",
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}
	db.CreateEvent(context.Background(), e1)

	e2 := &models.Event{
		MonitorID: createdMonitor.ID,
		Status:    models.StatusUp,
		Message:   "Online",
		CreatedAt: time.Now(),
	}
	db.CreateEvent(context.Background(), e2)

	// 3. Test handleListEvents (History)
	req = httptest.NewRequest("GET", "/api/v1/events/history?limit=10", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK for history, got %d", rr.Code)
	}
	var history []models.Event
	json.NewDecoder(rr.Body).Decode(&history)
	if len(history) != 2 {
		t.Errorf("expected 2 global events in history, got %d", len(history))
	} else if history[0].Status != models.StatusUp {
		t.Errorf("expected newest event to be Up, got %s", history[0].Status)
	}

	// 4. Test handleListMonitorEvents
	req = httptest.NewRequest("GET", "/api/v1/monitors/"+createdMonitor.ID+"/events?limit=1", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK for monitor events, got %d", rr.Code)
	}
	var monitorEvents []models.Event
	json.NewDecoder(rr.Body).Decode(&monitorEvents)
	if len(monitorEvents) != 1 {
		t.Errorf("expected 1 monitor event due to limit, got %d", len(monitorEvents))
	} else if monitorEvents[0].Status != models.StatusUp {
		t.Errorf("expected newest event due to ordering, got %s", monitorEvents[0].Status)
	}
}
