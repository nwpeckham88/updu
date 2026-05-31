package checker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestSablierChecker_Validate(t *testing.T) {
	c := &SablierChecker{}

	if c.Type() != "sablier" {
		t.Fatalf("expected type sablier, got %q", c.Type())
	}

	valid := []byte(`{"url":"http://sablier.internal:6660","service_name":"media"}`)
	if err := c.Validate(valid); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	if err := c.Validate([]byte(`{"service_name":"media"}`)); err == nil {
		t.Fatal("expected error when url is missing")
	}

	if err := c.Validate([]byte(`{"url":"http://sablier.internal:6660"}`)); err == nil {
		t.Fatal("expected error when service_name is missing")
	}

	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Fatal("expected error for malformed json")
	}
}

func TestSablierChecker_Check(t *testing.T) {
	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	tests := []struct {
		name               string
		responseStatusCode int
		responseBody       string
		wantStatus         models.MonitorStatus
		wantMessage        string
		wantPath           string
		wantMetadata       string
	}{
		{
			name:               "ready services are up",
			responseStatusCode: http.StatusOK,
			responseBody:       `{"name":"media","status":"ready","replicas":1,"spec":{"replicas":1,"ttl":"5m"}}`,
			wantStatus:         models.StatusUp,
			wantMessage:        "ready",
			wantPath:           "/api/services/media",
			wantMetadata:       "ready",
		},
		{
			name:               "sleeping services stay up without waking",
			responseStatusCode: http.StatusOK,
			responseBody:       `{"name":"media","status":"sleeping","replicas":0,"spec":{"replicas":1,"ttl":"5m"}}`,
			wantStatus:         models.StatusUp,
			wantMessage:        "sleeping",
			wantPath:           "/api/services/media",
			wantMetadata:       "sleeping",
		},
		{
			name:               "sleeping with nonzero replicas is degraded",
			responseStatusCode: http.StatusOK,
			responseBody:       `{"name":"media","status":"sleeping","replicas":1,"spec":{"replicas":1,"ttl":"5m"}}`,
			wantStatus:         models.StatusDegraded,
			wantMessage:        "sleeping but has",
			wantPath:           "/api/services/media",
			wantMetadata:       "sleeping",
		},
		{
			name:               "starting services are pending",
			responseStatusCode: http.StatusOK,
			responseBody:       `{"name":"media","status":"starting","replicas":0,"spec":{"replicas":1,"ttl":"5m"}}`,
			wantStatus:         models.StatusPending,
			wantMessage:        "starting",
			wantPath:           "/api/services/media",
			wantMetadata:       "starting",
		},
		{
			name:               "ready with zero replicas is degraded",
			responseStatusCode: http.StatusOK,
			responseBody:       `{"name":"media","status":"ready","replicas":0,"spec":{"replicas":1,"ttl":"5m"}}`,
			wantStatus:         models.StatusDegraded,
			wantMessage:        "0 replicas",
			wantPath:           "/api/services/media",
			wantMetadata:       "ready",
		},
		{
			name:               "unknown sablier state is down",
			responseStatusCode: http.StatusOK,
			responseBody:       `{"name":"media","status":"paused","replicas":0,"spec":{"replicas":1,"ttl":"5m"}}`,
			wantStatus:         models.StatusDown,
			wantMessage:        "unknown sablier state",
			wantPath:           "/api/services/media",
			wantMetadata:       "paused",
		},
		{
			name:               "http errors are down",
			responseStatusCode: http.StatusServiceUnavailable,
			responseBody:       `{"error":"unavailable"}`,
			wantStatus:         models.StatusDown,
			wantMessage:        "HTTP 503",
			wantPath:           "/api/services/media",
		},
		{
			name:               "malformed json is down",
			responseStatusCode: http.StatusOK,
			responseBody:       `{bad`,
			wantStatus:         models.StatusDown,
			wantMessage:        "invalid JSON response",
			wantPath:           "/api/services/media",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.wantPath {
					t.Fatalf("expected request path %q, got %q", tt.wantPath, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer ts.Close()

			checker := &SablierChecker{}
			monitor := &models.Monitor{
				ID:       "sablier-media",
				TimeoutS: 5,
				Config: json.RawMessage(
					`{"url":"` + ts.URL + `/","service_name":"media"}`,
				),
			}

			result, err := checker.Check(ctx, monitor)
			if err != nil {
				t.Fatalf("check failed: %v", err)
			}

			if result.Status != tt.wantStatus {
				t.Fatalf("expected status %s, got %s (%s)", tt.wantStatus, result.Status, result.Message)
			}

			if !strings.Contains(result.Message, tt.wantMessage) {
				t.Fatalf("expected message to contain %q, got %q", tt.wantMessage, result.Message)
			}

			if tt.wantMetadata != "" && !strings.Contains(string(result.Metadata), tt.wantMetadata) {
				t.Fatalf("expected metadata to contain %q, got %s", tt.wantMetadata, result.Metadata)
			}
		})
	}
}
