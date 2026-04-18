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

func TestTransactionChecker(t *testing.T) {
	c := &TransactionChecker{}
	if c.Type() != "transaction" {
		t.Fatalf("Type() = %q, want %q", c.Type(), "transaction")
	}

	if err := c.Validate([]byte(`{"steps":[{"url":"https://example.com"}]}`)); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
	if err := c.Validate([]byte(`{"steps":[]}`)); err == nil {
		t.Fatal("expected error for empty steps")
	}
	if err := c.Validate([]byte(`{"steps":[{}]}`)); err == nil {
		t.Fatal("expected error for step missing url")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"abc123"}`))
		case "/cart":
			if r.Header.Get("Authorization") != "Bearer abc123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"items":["demo-plan"]}`))
		case "/checkout":
			if r.Header.Get("Authorization") != "Bearer abc123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"total":42}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	t.Run("executes a multi-step flow with extracted variables", func(t *testing.T) {
		monitor := &models.Monitor{
			ID: "txn-up",
			Config: json.RawMessage(`{
				"steps": [
					{
						"method": "POST",
						"url": "` + server.URL + `/login",
						"expected_status": 200,
						"extract": {"access_token": "token"}
					},
					{
						"method": "GET",
						"url": "` + server.URL + `/cart",
						"headers": {"Authorization": "Bearer {{access_token}}"},
						"expected_status": 200,
						"expected_body": "demo-plan"
					},
					{
						"method": "GET",
						"url": "` + server.URL + `/checkout",
						"headers": {"Authorization": "Bearer {{access_token}}"},
						"expected_status": 200,
						"expected_body": "total"
					}
				]
			}`),
			TimeoutS: 5,
		}

		res, err := c.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusUp {
			t.Fatalf("expected up, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "all 3 step(s) passed") {
			t.Fatalf("expected success message, got %q", res.Message)
		}
		if len(res.Metadata) == 0 {
			t.Fatal("expected metadata to be populated")
		}
	})

	t.Run("fails on the step that misses its expected body", func(t *testing.T) {
		monitor := &models.Monitor{
			ID: "txn-down",
			Config: json.RawMessage(`{
				"steps": [
					{
						"method": "POST",
						"url": "` + server.URL + `/login",
						"expected_status": 200,
						"extract": {"access_token": "token"}
					},
					{
						"method": "GET",
						"url": "` + server.URL + `/cart",
						"headers": {"Authorization": "Bearer {{access_token}}"},
						"expected_status": 200,
						"expected_body": "missing-item"
					}
				]
			}`),
			TimeoutS: 5,
		}

		res, err := c.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusDown {
			t.Fatalf("expected down, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "step 2") {
			t.Fatalf("expected step-specific error, got %q", res.Message)
		}
	})

	t.Run("fails when a referenced variable was never extracted", func(t *testing.T) {
		monitor := &models.Monitor{
			ID: "txn-missing-var",
			Config: json.RawMessage(`{
				"steps": [
					{
						"method": "POST",
						"url": "` + server.URL + `/login",
						"expected_status": 200
					},
					{
						"method": "GET",
						"url": "` + server.URL + `/cart",
						"headers": {"Authorization": "Bearer {{access_token}}"},
						"expected_status": 200
					}
				]
			}`),
			TimeoutS: 5,
		}

		res, err := c.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusDown {
			t.Fatalf("expected down, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, `step 2: undefined variable "access_token"`) {
			t.Fatalf("expected missing-variable failure at step 2, got %q", res.Message)
		}
	})
}
