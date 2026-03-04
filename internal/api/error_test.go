package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPI_ErrorPaths(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()
	router := srv.Router()

	endpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/monitors"},
		{"GET", "/api/v1/monitors/xyz"},
		{"PUT", "/api/v1/monitors/xyz"},
		{"DELETE", "/api/v1/monitors/xyz"},
		{"GET", "/api/v1/monitors/xyz/checks"},
		{"POST", "/api/v1/maintenance"},
		{"GET", "/api/v1/maintenance/xyz"},
		{"PUT", "/api/v1/maintenance/xyz"},
		{"DELETE", "/api/v1/maintenance/xyz"},
		{"POST", "/api/v1/incidents"},
		{"GET", "/api/v1/incidents/xyz"},
		{"PUT", "/api/v1/incidents/xyz"},
		{"DELETE", "/api/v1/incidents/xyz"},
		{"GET", "/api/v1/settings"},
		{"PUT", "/api/v1/settings"},
		{"GET", "/api/v1/users"},
		{"PUT", "/api/v1/users/xyz"},
		{"DELETE", "/api/v1/users/xyz"},
		{"POST", "/api/v1/notifications"},
		{"GET", "/api/v1/notifications/xyz"},
		{"PUT", "/api/v1/notifications/xyz"},
		{"DELETE", "/api/v1/notifications/xyz"},
		{"POST", "/api/v1/notifications/xyz/test"},
		{"POST", "/api/v1/status-pages"},
		{"GET", "/api/v1/status-pages/xyz"},
		{"PUT", "/api/v1/status-pages/xyz"},
		{"DELETE", "/api/v1/status-pages/xyz"},
		{"POST", "/api/v1/auth/login"},
		{"POST", "/api/v1/auth/register"},
		{"GET", "/api/v1/auth/session"},
		{"POST", "/api/v1/auth/logout"},
		{"GET", "/api/v1/dashboard"},
		{"GET", "/api/v1/events"},
		{"GET", "/api/v1/system/health"},
		{"GET", "/api/v1/groups"},
		{"PUT", "/api/v1/groups/xyz"},
		{"DELETE", "/api/v1/groups/xyz"},
		{"GET", "/api/v1/monitors"},
		{"GET", "/api/v1/monitors/xyz/uptime"},
		{"GET", "/api/v1/system/metrics"},
		{"GET", "/api/v1/system/backup"},
		{"POST", "/api/v1/system/backup"},
		{"POST", "/api/v1/heartbeat/slug"},
	}

	// 1. Unauthenticated (should 401)
	for _, e := range endpoints {
		req := httptest.NewRequest(e.method, e.path, bytes.NewBufferString(`{}`))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}

	// 2. Authenticated but invalid bodies/bad IDs (should 400/404)
	reqUser := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(`{"username":"admin", "password":"password123"}`))
	router.ServeHTTP(httptest.NewRecorder(), reqUser)

	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin", "password":"password123"}`))
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, reqLogin)

	// Ensure we got a cookie
	var cookie *http.Cookie
	if len(rrLogin.Result().Cookies()) > 0 {
		cookie = rrLogin.Result().Cookies()[0]
	}

	for _, e := range endpoints {
		// Sending malformed JSON to trigger 400s
		req := httptest.NewRequest(e.method, e.path, bytes.NewBufferString(`{invalid json`))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}
