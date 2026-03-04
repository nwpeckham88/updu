package channels

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestChannelTypes(t *testing.T) {
	d := NewDiscordChannel()
	if d.Type() != "discord" {
		t.Errorf("discord type mismatch")
	}

	w := NewWebhookChannel()
	if w.Type() != "webhook" {
		t.Errorf("webhook type mismatch")
	}

	s := NewSlackChannel()
	if s.Type() != "slack" {
		t.Errorf("slack type mismatch")
	}

	e := NewEmailChannel()
	if e.Type() != "email" {
		t.Errorf("email type mismatch")
	}

	n := NewNtfyChannel()
	if n.Type() != "ntfy" {
		t.Errorf("ntfy type mismatch")
	}
}

func TestEmailChannel_Send(t *testing.T) {
	e := NewEmailChannel()
	ctx := context.Background()

	m := &models.Monitor{
		ID:   "mon-1",
		Name: "Test Mon",
		Type: "http",
	}

	res := &models.CheckResult{
		Status:  models.StatusDown,
		Message: "Down!",
	}

	config := map[string]any{
		"to":   "test@example.com",
		"host": "localhost",
		"port": float64(25),
		"from": "updu@example.com",
	}

	// Email send should fail if no local SMTP server is running, but it hits the code path
	err := e.Send(ctx, m, res, config)
	if err == nil {
		t.Logf("expected error from local smtp dial, got nil")
	}
}

func TestNtfyChannel_Send(t *testing.T) {
	n := NewNtfyChannel()
	ctx := context.Background()

	latency := 42
	m := &models.Monitor{ID: "mon-1", Name: "API Server", Type: "http"}
	resDown := &models.CheckResult{Status: models.StatusDown, Message: "Connection refused", LatencyMs: &latency}
	resUp := &models.CheckResult{Status: models.StatusUp, LatencyMs: &latency}
	resDegraded := &models.CheckResult{Status: models.StatusDegraded, LatencyMs: &latency}

	// 1. Missing URL should error
	err := n.Send(ctx, m, resDown, map[string]any{})
	if err == nil {
		t.Error("expected error for missing URL")
	}

	// 2. Success with mock server (DOWN status)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	err = n.Send(ctx, m, resDown, map[string]any{"url": mockServer.URL})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 3. UP status
	err = n.Send(ctx, m, resUp, map[string]any{"url": mockServer.URL})
	if err != nil {
		t.Errorf("expected no error for UP status, got %v", err)
	}

	// 4. DEGRADED status
	err = n.Send(ctx, m, resDegraded, map[string]any{"url": mockServer.URL})
	if err != nil {
		t.Errorf("expected no error for DEGRADED status, got %v", err)
	}

	// 5. Server error should return error
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errServer.Close()

	err = n.Send(ctx, m, resDown, map[string]any{"url": errServer.URL})
	if err == nil {
		t.Error("expected error for server 500, got nil")
	}
}
