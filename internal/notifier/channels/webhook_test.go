package channels

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestWebhookChannel_Send(t *testing.T) {
	c := NewWebhookChannel()
	ctx := context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	monitor := &models.Monitor{ID: "mon-1", Name: "Test"}
	event := &models.Event{Status: models.StatusUp, Message: "OK"}
	config := map[string]any{
		"url":    ts.URL,
		"method": "POST",
		"headers": map[string]any{
			"X-Custom": "Value",
		},
	}

	err := c.Send(ctx, monitor, event, config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Test missing URL
	err = c.Send(ctx, monitor, event, map[string]any{})
	if err == nil {
		t.Error("expected error for missing URL")
	}

	// Test non-2xx status
	ts404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts404.Close()

	err = c.Send(ctx, monitor, event, map[string]any{"url": ts404.URL})
	if err == nil {
		t.Error("expected error for 404 status")
	}
}
