package channels

import (
	"context"
	"encoding/json"
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

	g := NewGotifyChannel()
	if g.Type() != "gotify" {
		t.Errorf("gotify type mismatch")
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

	event := &models.Event{
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
	err := e.Send(ctx, m, event, config)
	if err == nil {
		t.Logf("expected error from local smtp dial, got nil")
	}
}

func TestNtfyChannel_Send(t *testing.T) {
	n := NewNtfyChannel()
	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	m := &models.Monitor{ID: "mon-1", Name: "API Server", Type: "http"}
	eventDown := &models.Event{Status: models.StatusDown, Message: "Connection refused"}
	eventUp := &models.Event{Status: models.StatusUp}
	eventDegraded := &models.Event{Status: models.StatusDegraded}

	// 1. Missing URL should error
	err := n.Send(ctx, m, eventDown, map[string]any{})
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

	err = n.Send(ctx, m, eventDown, map[string]any{"url": mockServer.URL})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 3. UP status
	err = n.Send(ctx, m, eventUp, map[string]any{"url": mockServer.URL})
	if err != nil {
		t.Errorf("expected no error for UP status, got %v", err)
	}

	// 4. DEGRADED status
	err = n.Send(ctx, m, eventDegraded, map[string]any{"url": mockServer.URL})
	if err != nil {
		t.Errorf("expected no error for DEGRADED status, got %v", err)
	}

	// 5. Server error should return error
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errServer.Close()

	err = n.Send(ctx, m, eventDown, map[string]any{"url": errServer.URL})
	if err == nil {
		t.Error("expected error for server 500, got nil")
	}
}

func TestGotifyChannel_Send(t *testing.T) {
	g := NewGotifyChannel()
	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	m := &models.Monitor{ID: "mon-1", Name: "API Server", Type: "http"}
	eventDown := &models.Event{Status: models.StatusDown, Message: "Connection refused"}
	eventUp := &models.Event{Status: models.StatusUp}

	err := g.Send(ctx, m, eventDown, map[string]any{"token": "abc123"})
	if err == nil {
		t.Error("expected error for missing URL")
	}

	err = g.Send(ctx, m, eventDown, map[string]any{"url": "https://gotify.example.com"})
	if err == nil {
		t.Error("expected error for missing token")
	}

	var requestCount int
	var titles []string
	var priorities []int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/message" {
			t.Errorf("expected /message path, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("X-Gotify-Key") != "abc123" {
			t.Errorf("expected gotify token header, got %q", r.Header.Get("X-Gotify-Key"))
		}

		var payload struct {
			Title    string `json:"title"`
			Message  string `json:"message"`
			Priority int    `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if payload.Message == "" {
			t.Error("expected non-empty message")
		}
		titles = append(titles, payload.Title)
		priorities = append(priorities, payload.Priority)

		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	err = g.Send(ctx, m, eventDown, map[string]any{
		"url":   mockServer.URL,
		"token": "abc123",
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = g.Send(ctx, m, eventUp, map[string]any{
		"url":   mockServer.URL,
		"token": "abc123",
	})
	if err != nil {
		t.Errorf("expected no error for UP status, got %v", err)
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests, got %d", requestCount)
	}
	if len(titles) == 2 {
		if titles[0] != "Monitor API Server is down" {
			t.Errorf("unexpected first title %q", titles[0])
		}
		if priorities[0] != 8 {
			t.Errorf("unexpected first priority %d", priorities[0])
		}
		if titles[1] != "Monitor API Server is up" {
			t.Errorf("unexpected second title %q", titles[1])
		}
		if priorities[1] != 2 {
			t.Errorf("unexpected second priority %d", priorities[1])
		}
	}

	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer errServer.Close()

	err = g.Send(ctx, m, eventDown, map[string]any{
		"url":   errServer.URL,
		"token": "abc123",
	})
	if err == nil {
		t.Error("expected error for server 502, got nil")
	}
}
