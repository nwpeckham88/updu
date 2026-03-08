package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/config"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/notifier"
	"github.com/updu/updu/internal/realtime"
	"github.com/updu/updu/internal/scheduler"
	"github.com/updu/updu/internal/storage"
)

func setupAPITest(t *testing.T) (*Server, *storage.DB, func()) {
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
		SessionTTLDays:  7,
		EnableCustomCSS: true,
		AllowLocalhost:  true,
	}

	a := auth.New(db, cfg)
	reg := checker.NewRegistry(true)
	sse := realtime.NewHub()
	n := notifier.New(db)
	sched := scheduler.New(db, reg, sse, n, 5)
	sched.DisableStagger = true

	if err := sched.Start(context.Background()); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	srv := NewServer(db, a, reg, sched, n, sse, cfg)

	cleanup := func() {
		sched.Stop()
		db.Close()
		os.RemoveAll(tempDir)
	}

	return srv, db, cleanup
}

func TestAPI_Health(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/system/health", nil)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", rr.Code)
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestAPI_AuthFlow(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// 1. Register first user (admin)
	regBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "password123",
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for register, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. Login
	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "password123",
	})
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for login, got %d", rr.Code)
	}

	// Capture session cookie
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "updu_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie, got none")
	}

	// 3. Get Session Info
	req = httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for session, got %d", rr.Code)
	}

	// 4. Logout
	req = httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for logout, got %d", rr.Code)
	}
}

func TestAPI_MonitorCRUD(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// Register and login to get admin access
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Create Monitor
	monBody, _ := json.Marshal(map[string]any{
		"name":       "Test Google",
		"type":       "http",
		"config":     map[string]string{"url": "https://google.com"},
		"interval_s": 60,
	})
	req := httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(monBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var created models.Monitor
	json.NewDecoder(rr.Body).Decode(&created)
	if created.ID == "" {
		t.Fatal("expected monitor ID, got empty")
	}

	// 2. List Monitors
	req = httptest.NewRequest("GET", "/api/v1/monitors", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for list, got %d", rr.Code)
	}

	// 3. Get Monitor
	req = httptest.NewRequest("GET", "/api/v1/monitors/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for get, got %d", rr.Code)
	}

	// 4. Update Monitor
	created.Name = "Updated Name"
	updateBody, _ := json.Marshal(created)
	req = httptest.NewRequest("PUT", "/api/v1/monitors/"+created.ID, bytes.NewBuffer(updateBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for update, got %d", rr.Code)
	}

	// 5. Delete Monitor
	req = httptest.NewRequest("DELETE", "/api/v1/monitors/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for delete, got %d", rr.Code)
	}
}

func TestAPI_StatusPageCRUD(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// Login as admin
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Create Status Page
	spBody, _ := json.Marshal(map[string]any{
		"name":      "Public Status",
		"slug":      "public",
		"is_public": true,
	})
	req := httptest.NewRequest("POST", "/api/v1/status-pages", bytes.NewBuffer(spBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var created models.StatusPage
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode created status page: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected status page ID, got empty. Body: %s", rr.Body.String())
	}
	t.Logf("Created Status Page ID: %s", created.ID)

	// 2. List Status Pages
	req = httptest.NewRequest("GET", "/api/v1/status-pages", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for list, got %d", rr.Code)
	}

	// 3. Get Status Page by Slug (Public)
	req = httptest.NewRequest("GET", "/api/v1/status-pages/public", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for public get, got %d", rr.Code)
	}

	// 4. Get Status Page by ID
	req = httptest.NewRequest("GET", "/api/v1/status-pages/"+created.ID+"/detail", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for get by ID, got %d", rr.Code)
	}

	// 5. Update Status Page
	created.Name = "Updated Page"
	updateBody, _ := json.Marshal(created)
	req = httptest.NewRequest("PUT", "/api/v1/status-pages/"+created.ID, bytes.NewBuffer(updateBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for update, got %d", rr.Code)
	}

	// 6. Delete Status Page
	req = httptest.NewRequest("DELETE", "/api/v1/status-pages/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for delete, got %d", rr.Code)
	}
}

func TestAPI_NotificationsCRUD(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// Login as admin
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Create Notification Channel
	ncBody, _ := json.Marshal(map[string]any{
		"name":    "Slack",
		"type":    "webhook",
		"config":  map[string]string{"url": "http://slack.com"},
		"enabled": true,
	})
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBuffer(ncBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Errorf("expected status OK or Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var created models.NotificationChannel
	json.NewDecoder(rr.Body).Decode(&created)

	// 2. List Notification Channels
	req = httptest.NewRequest("GET", "/api/v1/notifications", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for list, got %d", rr.Code)
	}

	// 3. Get Notification Channel
	req = httptest.NewRequest("GET", "/api/v1/notifications/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for get, got %d", rr.Code)
	}

	// 4. Update Notification Channel
	created.Name = "Updated Slack"
	updateBody, _ := json.Marshal(created)
	req = httptest.NewRequest("PUT", "/api/v1/notifications/"+created.ID, bytes.NewBuffer(updateBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for update, got %d", rr.Code)
	}

	// 5. Test Notification Channel
	req = httptest.NewRequest("POST", "/api/v1/notifications/"+created.ID+"/test", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for test, got %d", rr.Code)
	}

	// 6. Delete Notification Channel
	req = httptest.NewRequest("DELETE", "/api/v1/notifications/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for delete, got %d", rr.Code)
	}
}

func TestAPI_DashboardAndUptime(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// Login as admin
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Create a monitor first so dashboard isn't empty
	monBody, _ := json.Marshal(map[string]any{
		"name":   "Test",
		"type":   "http",
		"config": map[string]string{"url": "http://example.com"},
	})
	req := httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(monBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var created models.Monitor
	json.NewDecoder(rr.Body).Decode(&created)

	// 2. Dashboard
	req = httptest.NewRequest("GET", "/api/v1/dashboard", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for dashboard, got %d", rr.Code)
	}

	// 3. Monitor Checks
	req = httptest.NewRequest("GET", "/api/v1/monitors/"+created.ID+"/checks", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for monitor checks, got %d", rr.Code)
	}

	// 4. Monitor Uptime
	req = httptest.NewRequest("GET", "/api/v1/monitors/"+created.ID+"/uptime", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK for monitor uptime, got %d", rr.Code)
	}
}

func TestAPI_ErrorCases(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// 1. Unauthorized access
	req := httptest.NewRequest("GET", "/api/v1/monitors", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status Unauthorized, got %d", rr.Code)
	}

	// 2. Login with bad password
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))

	badLoginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrongpassword"})
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(badLoginBody))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status Unauthorized, got %d", rr.Code)
	}

	// 3. Register with short password
	shortPwBody, _ := json.Marshal(map[string]string{"username": "newuser", "password": "123"})
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(shortPwBody))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status BadRequest, got %d", rr.Code)
	}
}

func TestAPI_IncidentCRUD(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Create Incident
	incBody, _ := json.Marshal(map[string]any{
		"title":    "Outage",
		"status":   "active",
		"severity": "critical",
	})
	req := httptest.NewRequest("POST", "/api/v1/incidents", bytes.NewBuffer(incBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected status Created, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. List Incidents
	req = httptest.NewRequest("GET", "/api/v1/incidents", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", rr.Code)
	}
}

func TestAPI_MaintenanceCRUD(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Create Maintenance
	mwBody, _ := json.Marshal(map[string]any{
		"title":     "Upgrade",
		"starts_at": time.Now(),
		"ends_at":   time.Now().Add(time.Hour),
	})
	req := httptest.NewRequest("POST", "/api/v1/maintenance", bytes.NewBuffer(mwBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected status Created, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. List Maintenance
	req = httptest.NewRequest("GET", "/api/v1/maintenance", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", rr.Code)
	}
}

func TestAPI_MiscCoverage(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Get non-existent monitor
	req := httptest.NewRequest("GET", "/api/v1/monitors/nonexistent", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent monitor, got %d", rr.Code)
	}

	// 2. Get non-existent notification channel
	req = httptest.NewRequest("GET", "/api/v1/notifications/nonexistent", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent notify, got %d", rr.Code)
	}

	// 3. Create Status Page with missing slug
	spBody, _ := json.Marshal(map[string]any{"name": "No Slug"})
	req = httptest.NewRequest("POST", "/api/v1/status-pages", bytes.NewBuffer(spBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing slug, got %d", rr.Code)
	}
}

func TestAPI_IncidentFullCRUD(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Create
	incBody, _ := json.Marshal(map[string]any{
		"title":    "Database Down",
		"status":   "investigating",
		"severity": "critical",
	})
	req := httptest.NewRequest("POST", "/api/v1/incidents", bytes.NewBuffer(incBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var created models.Incident
	json.NewDecoder(rr.Body).Decode(&created)

	// 2. Get
	req = httptest.NewRequest("GET", "/api/v1/incidents/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// 3. Update
	created.Status = "resolved"
	updateBody, _ := json.Marshal(created)
	req = httptest.NewRequest("PUT", "/api/v1/incidents/"+created.ID, bytes.NewBuffer(updateBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// 4. Delete
	req = httptest.NewRequest("DELETE", "/api/v1/incidents/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAPI_MaintenanceFullCRUD(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Create
	mwBody, _ := json.Marshal(map[string]any{
		"title":     "Weekly Backup",
		"starts_at": time.Now(),
		"ends_at":   time.Now().Add(time.Hour),
	})
	req := httptest.NewRequest("POST", "/api/v1/maintenance", bytes.NewBuffer(mwBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var created models.MaintenanceWindow
	json.NewDecoder(rr.Body).Decode(&created)

	// 2. Get
	req = httptest.NewRequest("GET", "/api/v1/maintenance/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// 3. Update
	created.Title = "Daily Backup"
	updateBody, _ := json.Marshal(created)
	req = httptest.NewRequest("PUT", "/api/v1/maintenance/"+created.ID, bytes.NewBuffer(updateBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// 4. Delete
	req = httptest.NewRequest("DELETE", "/api/v1/maintenance/"+created.ID, nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAPI_HeartbeatPing(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	// Setup a heartbeat in DB with a token
	h := &models.Heartbeat{
		Slug:      "test-job",
		Token:     "secret-token",
		ExpectedS: 60,
	}
	db.UpsertHeartbeat(context.Background(), h)

	// 1. Ping with valid token (query param)
	req := httptest.NewRequest("POST", "/api/v1/heartbeat/test-job?token=secret-token", nil)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid token, got %d", rr.Code)
	}

	// 2. Ping with valid token (Authorization header)
	req = httptest.NewRequest("POST", "/api/v1/heartbeat/test-job", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rr = httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid header token, got %d", rr.Code)
	}

	// 3. Ping with invalid token
	req = httptest.NewRequest("POST", "/api/v1/heartbeat/test-job?token=wrong", nil)
	rr = httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", rr.Code)
	}

	// 4. Ping with missing token
	req = httptest.NewRequest("POST", "/api/v1/heartbeat/test-job", nil)
	rr = httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing token, got %d", rr.Code)
	}

	// 5. Ping non-existent
	req = httptest.NewRequest("POST", "/api/v1/heartbeat/ghost?token=any", nil)
	rr = httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// helper to register admin + login, then register viewer + login
func setupAdminAndViewer(t *testing.T, srv *Server) (adminCookie, viewerCookie *http.Cookie) {
	t.Helper()
	router := srv.Router()

	// Register admin (first user)
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))

	// Login admin
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	adminCookie = rrLogin.Result().Cookies()[0]

	// Admin registers a viewer
	viewerBody, _ := json.Marshal(map[string]string{"username": "viewer1", "password": "password123"})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(viewerBody))
	req.AddCookie(adminCookie)
	router.ServeHTTP(httptest.NewRecorder(), req)

	// Login viewer
	viewerLogin, _ := json.Marshal(map[string]string{"username": "viewer1", "password": "password123"})
	rrViewerLogin := httptest.NewRecorder()
	router.ServeHTTP(rrViewerLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(viewerLogin)))
	viewerCookie = rrViewerLogin.Result().Cookies()[0]

	return adminCookie, viewerCookie
}

func TestAPI_AdminEnforcement(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	adminCookie, viewerCookie := setupAdminAndViewer(t, srv)
	router := srv.Router()

	// Admin routes that viewers should NOT access (403 Forbidden)
	adminOnlyRoutes := []struct {
		method string
		path   string
		body   any
	}{
		{"POST", "/api/v1/monitors", map[string]any{"name": "Test", "type": "http", "config": map[string]string{"url": "http://example.com"}}},
		{"POST", "/api/v1/incidents", map[string]any{"title": "Test", "status": "investigating", "severity": "minor"}},
		{"POST", "/api/v1/maintenance", map[string]any{"title": "Test", "starts_at": time.Now(), "ends_at": time.Now().Add(time.Hour)}},
		{"POST", "/api/v1/notifications", map[string]any{"name": "Test", "type": "webhook", "config": map[string]string{"url": "http://example.com"}, "enabled": true}},
		{"POST", "/api/v1/settings", map[string]string{"key": "val"}},
		{"GET", "/api/v1/admin/users", nil},
		{"GET", "/api/v1/settings", nil},
		{"GET", "/api/v1/system/metrics", nil},
		{"GET", "/api/v1/system/backup", nil},
	}

	for _, tc := range adminOnlyRoutes {
		var bodyBuf *bytes.Buffer
		if tc.body != nil {
			b, _ := json.Marshal(tc.body)
			bodyBuf = bytes.NewBuffer(b)
		} else {
			bodyBuf = bytes.NewBuffer(nil)
		}

		// Viewer should get 403
		req := httptest.NewRequest(tc.method, tc.path, bodyBuf)
		req.AddCookie(viewerCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Errorf("[VIEWER] %s %s: expected 403, got %d", tc.method, tc.path, rr.Code)
		}

		// Admin should NOT get 403
		if tc.body != nil {
			b, _ := json.Marshal(tc.body)
			bodyBuf = bytes.NewBuffer(b)
		} else {
			bodyBuf = bytes.NewBuffer(nil)
		}
		req = httptest.NewRequest(tc.method, tc.path, bodyBuf)
		req.AddCookie(adminCookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code == http.StatusForbidden {
			t.Errorf("[ADMIN] %s %s: got 403 unexpectedly", tc.method, tc.path)
		}
	}
}

func TestAPI_PasswordChange(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// Register + login
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	// 1. Wrong current password
	wrongBody, _ := json.Marshal(map[string]string{
		"current_password": "wrongpassword",
		"new_password":     "newpassword123",
	})
	req := httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewBuffer(wrongBody))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong current password, got %d", rr.Code)
	}

	// 2. New password too short
	shortBody, _ := json.Marshal(map[string]string{
		"current_password": "password123",
		"new_password":     "short",
	})
	req = httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewBuffer(shortBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short new password, got %d", rr.Code)
	}

	// 3. Valid password change
	validBody, _ := json.Marshal(map[string]string{
		"current_password": "password123",
		"new_password":     "newpassword123",
	})
	req = httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewBuffer(validBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid password change, got %d: %s", rr.Code, rr.Body.String())
	}

	// 4. Login with OLD password should fail
	oldLogin, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(oldLogin))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with old password after change, got %d", rr.Code)
	}

	// 5. Login with NEW password should succeed
	newLogin, _ := json.Marshal(map[string]string{"username": "admin", "password": "newpassword123"})
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(newLogin))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with new password, got %d", rr.Code)
	}
}

func TestAPI_CustomCSS(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// 1. Empty CSS by default
	req := httptest.NewRequest("GET", "/api/v1/custom.css", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Errorf("expected text/css content-type, got %s", ct)
	}
	if rr.Body.String() != "" {
		t.Errorf("expected empty CSS, got %q", rr.Body.String())
	}

	// 2. Set custom CSS via settings (admin)
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody)))
	sessionCookie := rrLogin.Result().Cookies()[0]

	cssContent := ":root { --color-primary: red; }"
	settingsBody, _ := json.Marshal(map[string]string{"custom_css": cssContent})
	req = httptest.NewRequest("POST", "/api/v1/settings", bytes.NewBuffer(settingsBody))
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for save settings, got %d", rr.Code)
	}

	// 3. Retrieve custom CSS (public, no auth)
	req = httptest.NewRequest("GET", "/api/v1/custom.css", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Body.String() != cssContent {
		t.Errorf("expected CSS %q, got %q", cssContent, rr.Body.String())
	}
}

func TestAPI_LoginRateLimit(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// Register a user
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))

	// Make 5 login attempts (all should be processed, even if wrong)
	for i := 0; i < 5; i++ {
		badLogin, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrongpassword"})
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(badLogin))
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("attempt %d: expected 401, got %d", i+1, rr.Code)
		}
	}

	// 6th attempt should be rate limited
	badLogin, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrongpassword"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(badLogin))
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 on 6th attempt, got %d", rr.Code)
	}

	// Different IP should still work
	badLogin2, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrongpassword"})
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(badLogin2))
	req.RemoteAddr = "10.0.0.1:55555"
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code == http.StatusTooManyRequests {
		t.Errorf("different IP should not be rate limited")
	}
}

func TestAPI_ViewerRegistration(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	// Register admin (first user)
	regBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regBody)))

	// Non-admin trying to register should be denied
	viewerReg, _ := json.Marshal(map[string]string{"username": "hacker", "password": "password123"})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(viewerReg))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin register, got %d", rr.Code)
	}

	// Reset rate limiter before the helper which makes multiple requests
	srv.loginMu.Lock()
	srv.loginAttempts = make(map[string]*loginEntry)
	srv.loginMu.Unlock()

	// Login viewer, try to register — should also fail
	adminCookie, viewerCookie := setupAdminAndViewer(t, srv)
	_ = adminCookie

	anotherUser, _ := json.Marshal(map[string]string{"username": "another", "password": "password123"})
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(anotherUser))
	req.AddCookie(viewerCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer register attempt, got %d", rr.Code)
	}
}

func TestAPI_GroupsAndUserManagement(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	adminCookie, _ := setupAdminAndViewer(t, srv)
	router := srv.Router()

	// List groups
	req := httptest.NewRequest("GET", "/api/v1/groups", nil)
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for list groups, got %d", rr.Code)
	}

	// List users
	req = httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for list users, got %d", rr.Code)
	}

	// Get system metrics
	req = httptest.NewRequest("GET", "/api/v1/system/metrics", nil)
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for metrics, got %d", rr.Code)
	}

	// Export backup
	req = httptest.NewRequest("GET", "/api/v1/system/backup", nil)
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for export backup, got %d", rr.Code)
	}

	// Import backup (empty config)
	importBody, _ := json.Marshal(map[string]any{"monitors": []any{}})
	req = httptest.NewRequest("POST", "/api/v1/system/backup", bytes.NewBuffer(importBody))
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for import backup, got %d", rr.Code)
	}
}

func TestAPI_UserRoleAndDelete(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	adminCookie, _ := setupAdminAndViewer(t, srv)
	router := srv.Router()

	// List users to get viewer ID
	req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var users []struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	json.NewDecoder(rr.Body).Decode(&users)

	var viewerID string
	for _, u := range users {
		if u.Username == "viewer1" {
			viewerID = u.ID
			break
		}
	}
	if viewerID == "" {
		t.Fatal("viewer user not found")
	}

	// Promote viewer to admin
	roleBody, _ := json.Marshal(map[string]string{"role": "admin"})
	req = httptest.NewRequest("PUT", "/api/v1/admin/users/"+viewerID+"/role", bytes.NewBuffer(roleBody))
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for role update, got %d: %s", rr.Code, rr.Body.String())
	}

	// Demote back to viewer
	roleBody2, _ := json.Marshal(map[string]string{"role": "viewer"})
	req = httptest.NewRequest("PUT", "/api/v1/admin/users/"+viewerID+"/role", bytes.NewBuffer(roleBody2))
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for demote, got %d", rr.Code)
	}

	// Invalid role
	badRole, _ := json.Marshal(map[string]string{"role": "superuser"})
	req = httptest.NewRequest("PUT", "/api/v1/admin/users/"+viewerID+"/role", bytes.NewBuffer(badRole))
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid role, got %d", rr.Code)
	}

	// Admin can't delete themselves
	var adminID string
	for _, u := range users {
		if u.Username == "admin" {
			adminID = u.ID
			break
		}
	}
	req = httptest.NewRequest("DELETE", "/api/v1/admin/users/"+adminID, nil)
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for self-delete, got %d", rr.Code)
	}

	// Admin can delete viewer
	req = httptest.NewRequest("DELETE", "/api/v1/admin/users/"+viewerID, nil)
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for delete user, got %d", rr.Code)
	}
}

func TestAPI_GroupCRUD(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	adminCookie, _ := setupAdminAndViewer(t, srv)
	router := srv.Router()

	// Create a monitor with a group
	monBody, _ := json.Marshal(map[string]any{
		"name":       "Grouped",
		"type":       "http",
		"group":      "production",
		"config":     map[string]string{"url": "http://example.com"},
		"interval_s": 60,
	})
	req := httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(monBody))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// List groups — should include "production"
	req = httptest.NewRequest("GET", "/api/v1/groups", nil)
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// Rename group
	renameBody, _ := json.Marshal(map[string]string{"name": "prod"})
	req = httptest.NewRequest("PUT", "/api/v1/groups/production", bytes.NewBuffer(renameBody))
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for rename, got %d: %s", rr.Code, rr.Body.String())
	}

	// Rename with missing name
	emptyBody, _ := json.Marshal(map[string]string{})
	req = httptest.NewRequest("PUT", "/api/v1/groups/prod", bytes.NewBuffer(emptyBody))
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", rr.Code)
	}

	// Delete group
	req = httptest.NewRequest("DELETE", "/api/v1/groups/prod", nil)
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for delete, got %d", rr.Code)
	}
}

func TestAPI_SettingsCRUD(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	adminCookie, _ := setupAdminAndViewer(t, srv)
	router := srv.Router()

	// Save settings
	settingsBody, _ := json.Marshal(map[string]string{
		"site_name": "My Monitor",
		"theme":     "dark",
	})
	req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewBuffer(settingsBody))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Read settings back
	req = httptest.NewRequest("GET", "/api/v1/settings", nil)
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var settings map[string]string
	json.NewDecoder(rr.Body).Decode(&settings)
	if settings["site_name"] != "My Monitor" {
		t.Errorf("expected site_name 'My Monitor', got %q", settings["site_name"])
	}

	// Invalid body
	req = httptest.NewRequest("POST", "/api/v1/settings", bytes.NewBufferString("not json"))
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", rr.Code)
	}
}

func TestAPI_InvalidBodies(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	adminCookie, _ := setupAdminAndViewer(t, srv)
	router := srv.Router()

	invalidRoutes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/monitors"},
		{"POST", "/api/v1/incidents"},
		{"POST", "/api/v1/maintenance"},
		{"POST", "/api/v1/notifications"},
		{"POST", "/api/v1/status-pages"},
	}

	for _, tc := range invalidRoutes {
		// Invalid JSON
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString("not json"))
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("[INVALID JSON] %s %s: expected 400, got %d", tc.method, tc.path, rr.Code)
		}
	}

	// Reset rate limiter before testing login/register with invalid bodies
	srv.loginMu.Lock()
	srv.loginAttempts = make(map[string]*loginEntry)
	srv.loginMu.Unlock()

	// Login with invalid JSON
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString("not json"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid login body, got %d", rr.Code)
	}

	// Register with invalid JSON
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString("not json"))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid register body, got %d", rr.Code)
	}

	// Register with short username
	shortUser, _ := json.Marshal(map[string]string{"username": "ab", "password": "password123"})
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(shortUser))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short username, got %d", rr.Code)
	}

	// Password change with invalid JSON
	req = httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewBufferString("not json"))
	req.AddCookie(adminCookie)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid password change body, got %d", rr.Code)
	}
}
