package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/updu/updu/internal/storage"
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

func TestAPI_OpenAPIEndpoint(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	doc := decodeOpenAPIDoc(t, srv.Router())

	if doc.OpenAPI == "" {
		t.Fatal("expected openapi field")
	}
	if len(doc.Info) == 0 {
		t.Fatal("expected info block")
	}
	if len(doc.Paths) == 0 {
		t.Fatal("expected non-empty paths")
	}

	assertOpenAPICoverage(t, doc.Paths, map[string][]string{
		"/api/v1/auth/login":                 {"post"},
		"/api/v1/auth/register":              {"post"},
		"/api/v1/auth/setup":                 {"get"},
		"/api/v1/auth/providers":             {"get"},
		"/api/v1/status-pages/{slug}/unlock": {"post"},
		"/api/v1/status-pages/{slug}":        {"get"},
		"/api/v1/heartbeat/{slug}":           {"post"},
		"/heartbeat/{token}":                 {"get", "post", "put"},
		"/api/v1/system/health":              {"get"},
		"/healthz":                           {"get"},
		"/api/v1/openapi.json":               {"get"},
		"/api/v1/metrics":                    {"get"},
		"/api/v1/custom.css":                 {"get"},
		"/api/v1/events":                     {"get"},
		"/api/v1/auth/logout":                {"post"},
		"/api/v1/auth/session":               {"get"},
		"/api/v1/auth/password":              {"put"},
		"/api/v1/monitors":                   {"get", "post"},
		"/api/v1/monitors/test":              {"post"},
		"/api/v1/monitors/{id}":              {"get", "put", "delete"},
		"/api/v1/monitors/{id}/checks":       {"get"},
		"/api/v1/monitors/{id}/events":       {"get"},
		"/api/v1/monitors/{id}/uptime":       {"get"},
		"/api/v1/dashboard":                  {"get"},
		"/api/v1/stats":                      {"get"},
		"/api/v1/events/history":             {"get"},
		"/api/v1/status-pages":               {"get", "post"},
		"/api/v1/status-pages/{id}/detail":   {"get"},
		"/api/v1/status-pages/{id}":          {"put", "delete"},
		"/api/v1/notifications":              {"get", "post"},
		"/api/v1/notifications/{id}":         {"get", "put", "delete"},
		"/api/v1/notifications/{id}/test":    {"post"},
		"/api/v1/incidents":                  {"get", "post"},
		"/api/v1/incidents/{id}":             {"get", "put", "delete"},
		"/api/v1/maintenance":                {"get", "post"},
		"/api/v1/maintenance/{id}":           {"get", "put", "delete"},
		"/api/v1/groups":                     {"get"},
		"/api/v1/groups/{name}":              {"put", "delete"},
		"/api/v1/admin/users":                {"get"},
		"/api/v1/admin/users/{id}/role":      {"put"},
		"/api/v1/admin/users/{id}":           {"delete"},
		"/api/v1/admin/api-tokens":           {"get", "post"},
		"/api/v1/admin/api-tokens/{id}":      {"delete"},
		"/api/v1/audit-logs":                 {"get"},
		"/api/v1/settings":                   {"get", "post"},
		"/api/v1/system/metrics":             {"get"},
		"/api/v1/system/backup":              {"get", "post"},
		"/api/v1/system/export/yaml":         {"get"},
		"/api/v1/system/version":             {"get"},
		"/api/v1/system/update":              {"post"},
	})
}

func decodeOpenAPIDoc(t *testing.T, router http.Handler) struct {
	OpenAPI string                    `json:"openapi"`
	Info    map[string]any            `json:"info"`
	Paths   map[string]map[string]any `json:"paths"`
} {
	t.Helper()

	req := httptest.NewRequest("GET", "/api/v1/openapi.json", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}

	var doc struct {
		OpenAPI string                    `json:"openapi"`
		Info    map[string]any            `json:"info"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&doc); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	return doc
}

func assertOpenAPICoverage(t *testing.T, paths map[string]map[string]any, expected map[string][]string) {
	t.Helper()

	for path, methods := range expected {
		pathItem, ok := paths[path]
		if !ok {
			t.Fatalf("expected path %q in OpenAPI document", path)
		}

		for _, method := range methods {
			if _, ok := pathItem[method]; !ok {
				t.Fatalf("expected %s %s in OpenAPI document", strings.ToUpper(method), path)
			}
		}
	}
}

func TestAPI_TokenScopes(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	ctx := context.Background()
	admin, err := srv.auth.Register(ctx, "admin-token", "password123")
	if err != nil {
		t.Fatalf("failed to register admin: %v", err)
	}

	readToken := seedAPIToken(t, db, admin.ID, "read-token", "read")
	writeToken := seedAPIToken(t, db, admin.ID, "write-token", "write")

	t.Run("read token can read but not write", func(t *testing.T) {
		readReq := httptest.NewRequest("GET", "/api/v1/monitors", nil)
		readReq.Header.Set("Authorization", "Bearer "+readToken)
		readRR := httptest.NewRecorder()
		srv.Router().ServeHTTP(readRR, readReq)

		if readRR.Code != http.StatusOK {
			t.Fatalf("expected read token GET to succeed, got %d: %s", readRR.Code, readRR.Body.String())
		}

		createBody := strings.NewReader(`{"name":"Token Monitor","type":"http","config":{"url":"https://example.com"},"interval_s":60}`)
		writeReq := httptest.NewRequest("POST", "/api/v1/monitors", createBody)
		writeReq.Header.Set("Authorization", "Bearer "+readToken)
		writeRR := httptest.NewRecorder()
		srv.Router().ServeHTTP(writeRR, writeReq)

		if writeRR.Code != http.StatusForbidden {
			t.Fatalf("expected read token POST to be forbidden, got %d: %s", writeRR.Code, writeRR.Body.String())
		}
	})

	t.Run("write token can mutate", func(t *testing.T) {
		createBody := strings.NewReader(`{"name":"Writable Monitor","type":"http","config":{"url":"https://example.com"},"interval_s":60}`)
		writeReq := httptest.NewRequest("POST", "/api/v1/monitors", createBody)
		writeReq.Header.Set("Authorization", "Bearer "+writeToken)
		writeRR := httptest.NewRecorder()
		srv.Router().ServeHTTP(writeRR, writeReq)

		if writeRR.Code != http.StatusCreated {
			t.Fatalf("expected write token POST to succeed, got %d: %s", writeRR.Code, writeRR.Body.String())
		}
	})

	t.Run("write token is blocked from session-only admin routes", func(t *testing.T) {
		adminReq := httptest.NewRequest("GET", "/api/v1/settings", nil)
		adminReq.Header.Set("Authorization", "Bearer "+writeToken)
		adminRR := httptest.NewRecorder()
		srv.Router().ServeHTTP(adminRR, adminReq)

		if adminRR.Code != http.StatusForbidden {
			t.Fatalf("expected write token GET /settings to be forbidden, got %d: %s", adminRR.Code, adminRR.Body.String())
		}
	})
}

func seedAPIToken(t *testing.T, db *storage.DB, userID, tokenValue, scope string) string {
	t.Helper()

	hash := sha256.Sum256([]byte(tokenValue))
	_, err := db.ExecContext(
		context.Background(),
		`INSERT INTO api_tokens (id, name, token_hash, prefix, scope, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fmt.Sprintf("tok_%s", scope),
		fmt.Sprintf("%s token", scope),
		fmt.Sprintf("%x", hash[:]),
		tokenValue[:min(8, len(tokenValue))],
		scope,
		userID,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("failed to seed api token: %v", err)
	}

	return tokenValue
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
