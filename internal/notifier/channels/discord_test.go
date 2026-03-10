package channels

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestDiscordChannel_Send(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewDiscordChannel()
	monitor := &models.Monitor{Name: "Test"}
	event := &models.Event{
		Status:    models.StatusUp,
		Message:   "OK",
		CreatedAt: time.Now(),
	}
	config := map[string]any{"url": ts.URL}

	err := c.Send(context.WithValue(context.Background(), AllowLocalhostKey, true), monitor, event, config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test missing URL
	err = c.Send(context.WithValue(context.Background(), AllowLocalhostKey, true), monitor, event, map[string]any{})
	if err == nil {
		t.Error("expected error for missing URL, got nil")
	}

	// Test non-2xx response
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts2.Close()
	err = c.Send(context.WithValue(context.Background(), AllowLocalhostKey, true), monitor, event, map[string]any{"url": ts2.URL})
	if err == nil {
		t.Error("expected error for non-2xx, got nil")
	}
}
