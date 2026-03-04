package realtime

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHub_Broadcast(t *testing.T) {
	h := NewHub()

	// Create a mock client
	w := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)

	done := make(chan bool)
	go func() {
		h.ServeHTTP(w, req)
		done <- true
	}()

	// Wait for client to be registered
	time.Sleep(50 * time.Millisecond)
	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", h.ClientCount())
	}

	// Broadcast an event
	h.Broadcast(Event{Type: "test", Data: "hello"})

	// Stop the server
	cancel()
	<-done

	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", h.ClientCount())
	}
}

func TestHub_ClientCount(t *testing.T) {
	h := NewHub()
	if h.ClientCount() != 0 {
		t.Errorf("expected 0, got %d", h.ClientCount())
	}
}
