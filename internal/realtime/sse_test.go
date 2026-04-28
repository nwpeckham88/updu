package realtime

import (
	"context"
	"net/http/httptest"
	"strings"
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

func TestHub_AdminOnlyBroadcastFiltersViewer(t *testing.T) {
	h := NewHub()

	adminRecorder, adminCancel, adminDone := serveTestClient(t, h, "admin")
	viewerRecorder, viewerCancel, viewerDone := serveTestClient(t, h, "viewer")

	time.Sleep(50 * time.Millisecond)
	if h.ClientCount() != 2 {
		t.Fatalf("expected 2 clients, got %d", h.ClientCount())
	}

	h.Broadcast(Event{Type: "secret", Data: "admin-only", AdminOnly: true})
	h.Broadcast(Event{Type: "public", Data: "visible"})
	time.Sleep(50 * time.Millisecond)

	adminCancel()
	viewerCancel()
	<-adminDone
	<-viewerDone

	adminBody := adminRecorder.Body.String()
	viewerBody := viewerRecorder.Body.String()
	if !strings.Contains(adminBody, "admin-only") {
		t.Fatalf("expected admin client to receive admin-only event, got %q", adminBody)
	}
	if strings.Contains(viewerBody, "admin-only") {
		t.Fatalf("expected viewer client not to receive admin-only event, got %q", viewerBody)
	}
	if !strings.Contains(viewerBody, "visible") {
		t.Fatalf("expected viewer client to receive public event, got %q", viewerBody)
	}
}

func serveTestClient(t *testing.T, h *Hub, role string) (*httptest.ResponseRecorder, context.CancelFunc, chan bool) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	done := make(chan bool)
	go func() {
		h.ServeHTTPWithRole(recorder, req, role)
		done <- true
	}()

	return recorder, cancel, done
}
