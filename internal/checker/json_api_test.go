package checker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestJSONAPIChecker(t *testing.T) {
	c := &JSONAPIChecker{}
	if c.Type() != "json" {
		t.Error("type error")
	}

	if err := c.Validate([]byte(`{"url": "http://test", "field": "status", "expected_value": "ok"}`)); err != nil {
		t.Error(err)
	}
	if err := c.Validate([]byte(`{"field": "status", "expected_value": "ok"}`)); err == nil {
		t.Error("expected err no url")
	}
	if err := c.Validate([]byte(`{"url": "http://test", "expected_value": "ok"}`)); err == nil {
		t.Error("expected err no field")
	}
	if err := c.Validate([]byte(`{"url": "http://test", "field": "status"}`)); err == nil {
		t.Error("expected err no expected val")
	}
	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Error("expected err bad json")
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/404" {
			w.WriteHeader(404)
			return
		}
		if r.URL.Path == "/badjson" {
			w.Write([]byte(`{bad`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "data": {"health": "good"}}`))
	}))
	defer s.Close()

	ctx := context.Background()
	monitor := &models.Monitor{
		Config: json.RawMessage(`{"url": "` + s.URL + `", "field": "status", "expected_value": "ok"}`),
	}
	res, _ := c.Check(ctx, monitor)
	if res.Status != models.StatusUp {
		t.Errorf("expected up, got %v: %v", res.Status, res.Message)
	}

	monitor.Config = json.RawMessage(`{"url": "` + s.URL + `", "field": "data.health", "expected_value": "good"}`)
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusUp {
		t.Errorf("expected up, got %v: %v", res.Status, res.Message)
	}

	// Field not found
	monitor.Config = json.RawMessage(`{"url": "` + s.URL + `", "field": "data.missing", "expected_value": "good"}`)
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Error("expected down")
	}

	// Field mismatch
	monitor.Config = json.RawMessage(`{"url": "` + s.URL + `", "field": "status", "expected_value": "bad"}`)
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Error("expected down")
	}

	// HTTP error 404
	monitor.Config = json.RawMessage(`{"url": "` + s.URL + `/404", "field": "status", "expected_value": "ok"}`)
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Error("expected down")
	}

	// Bad JSON response
	monitor.Config = json.RawMessage(`{"url": "` + s.URL + `/badjson", "field": "status", "expected_value": "ok"}`)
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Error("expected down")
	}

	// Invalid URL (trigger go http client error)
	monitor.Config = json.RawMessage(`{"url": "http://127.0.0.1:0", "field": "status", "expected_value": "ok"}`)
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Error("expected down")
	}

	// Invalid monitor Config JSON
	monitor.Config = json.RawMessage(`{bad`)
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Error("expected down")
	}
}
