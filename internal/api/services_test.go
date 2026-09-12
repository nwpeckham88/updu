package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/models"
)

func TestServicesAPI(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	cookie := registerAndLoginAdmin(t, router)

	// 1. Create a Service
	svcPayload := models.Service{
		Name:   "Vaultwarden App",
		Type:   models.ServiceTypeWeb,
		ZoneID: "default",
		Groups: []string{"Security"},
		Tags:   []string{"core"},
		Endpoints: []*models.ServiceEndpoint{
			{
				Name:       "Public Domain",
				ScopeID:    models.ScopePublic,
				TargetType: "http",
				Config:     json.RawMessage(`{"url":"https://example.com"}`),
				IsPrimary:  true,
			},
			{
				Name:       "Internal LAN",
				ScopeID:    models.ScopeLAN,
				TargetType: "tcp",
				Config:     json.RawMessage(`{"host":"127.0.0.1","port":80}`),
				IsPrimary:  false,
			},
		},
	}
	body, _ := json.Marshal(svcPayload)

	req := httptest.NewRequest("POST", "/api/v1/services", bytes.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/services returned %d: %s", rec.Code, rec.Body.String())
	}

	var created models.Service
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshaling created service: %v", err)
	}
	if created.ID == "" || created.Name != "Vaultwarden App" {
		t.Errorf("unexpected created service: %+v", created)
	}

	// 2. List Services
	req = httptest.NewRequest("GET", "/api/v1/services", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/services returned %d: %s", rec.Code, rec.Body.String())
	}

	var services []*models.Service
	if err := json.Unmarshal(rec.Body.Bytes(), &services); err != nil {
		t.Fatalf("unmarshaling services: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("expected 1 service, got %d", len(services))
	}

	// 3. Get Service by ID
	req = httptest.NewRequest("GET", "/api/v1/services/"+created.ID, nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/services/{id} returned %d: %s", rec.Code, rec.Body.String())
	}

	// 4. Probe Service Endpoints
	req = httptest.NewRequest("POST", "/api/v1/services/"+created.ID+"/probe", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/services/{id}/probe returned %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Get Network Topology
	req = httptest.NewRequest("GET", "/api/v1/topology", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/topology returned %d: %s", rec.Code, rec.Body.String())
	}
	var topo models.NetworkTopology
	if err := json.Unmarshal(rec.Body.Bytes(), &topo); err != nil {
		t.Fatalf("unmarshaling topology: %v", err)
	}
	if len(topo.Nodes) == 0 || len(topo.Services) == 0 {
		t.Errorf("topology empty: nodes=%d, services=%d", len(topo.Nodes), len(topo.Services))
	}

	// 6. List Zones and Scopes
	req = httptest.NewRequest("GET", "/api/v1/zones", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/v1/zones returned %d", rec.Code)
	}

	req = httptest.NewRequest("GET", "/api/v1/scopes", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/v1/scopes returned %d", rec.Code)
	}

	// 7. Delete Service
	req = httptest.NewRequest("DELETE", "/api/v1/services/"+created.ID, nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("DELETE /api/v1/services/{id} returned %d", rec.Code)
	}
}

func TestLLMsTxt(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()

	for _, path := range []string{"/.well-known/llms.txt", "/llms.txt"} {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s returned %d, expected 200", path, rec.Code)
		}
		if body := rec.Body.String(); !strings.Contains(body, "# updu") || !strings.Contains(body, "mcp") {
			t.Errorf("GET %s unexpected body content: %s", path, body)
		}
	}
}

func TestTLSCertificatesAPI(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	// Spin up a test TLS server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	router := srv.Router()
	cookie := registerAndLoginAdmin(t, router)

	// 1. Audit Live TLS Handshake (test passing full URL scheme to verify normalization)
	reqBody, _ := json.Marshal(map[string]any{
		"host": ts.URL, // e.g. "https://127.0.0.1:port"
	})

	req := httptest.NewRequest("POST", "/api/v1/certificates/test", bytes.NewReader(reqBody))
	req.AddCookie(cookie)
	// Guardrail: Allow localhost in test
	req = req.WithContext(context.WithValue(req.Context(), checker.AllowLocalhostKey, true))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/certificates/test returned %d: %s", rec.Code, rec.Body.String())
	}

	var audited models.TLSCertificate
	if err := json.Unmarshal(rec.Body.Bytes(), &audited); err != nil {
		t.Fatalf("unmarshaling tested certificate: %v", err)
	}
	if audited.ID == "" || audited.Domain == "" {
		t.Errorf("expected valid certificate output, got %+v", audited)
	}

	// 2. Verify certificate was persisted to database
	req = httptest.NewRequest("GET", "/api/v1/certificates", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/certificates returned %d: %s", rec.Code, rec.Body.String())
	}

	var certs []*models.TLSCertificate
	if err := json.Unmarshal(rec.Body.Bytes(), &certs); err != nil {
		t.Fatalf("unmarshaling certificates: %v", err)
	}
	if len(certs) == 0 {
		t.Fatalf("expected at least 1 persisted certificate, got 0")
	}
}

func TestTLSCertificateSSRFBlocked(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()
	srv.config.AllowLocalhost = false

	// Spin up a test TLS server on localhost
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	router := srv.Router()
	cookie := registerAndLoginAdmin(t, router)

	// Attempt audit to localhost WITHOUT AllowLocalhost context or config override
	reqBody, _ := json.Marshal(map[string]any{
		"host": ts.URL,
	})

	req := httptest.NewRequest("POST", "/api/v1/certificates/test", bytes.NewReader(reqBody))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Should be blocked by SSRF defense (400 Bad Request)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected SSRF block (400), got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "loopback") && !strings.Contains(rec.Body.String(), "SSRF") && !strings.Contains(rec.Body.String(), "blocked") && !strings.Contains(rec.Body.String(), "not allowed") {
		t.Logf("SSRF block message: %s", rec.Body.String())
	}
}

func TestEmptyEndpointsAndCertificatesJSON(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	cookie := registerAndLoginAdmin(t, router)

	// 1. Empty certificates list returns [] not null
	req := httptest.NewRequest("GET", "/api/v1/certificates", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/certificates returned %d: %s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("expected empty array '[]', got %s", rec.Body.String())
	}

	// 2. Empty services list returns [] not null
	req = httptest.NewRequest("GET", "/api/v1/services", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/services returned %d: %s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("expected empty array '[]', got %s", rec.Body.String())
	}

	// 3. Empty topology has non-null slices
	req = httptest.NewRequest("GET", "/api/v1/topology", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/topology returned %d: %s", rec.Code, rec.Body.String())
	}
	var topo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &topo); err != nil {
		t.Fatalf("unmarshaling topology: %v", err)
	}
	for _, key := range []string{"nodes", "scopes", "zones", "services", "edges"} {
		v, exists := topo[key]
		if !exists || v == nil {
			t.Errorf("topology key %q is nil or missing in JSON response: %s", key, rec.Body.String())
		}
	}
}


