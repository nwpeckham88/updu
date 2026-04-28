package realtime

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

const adminRole = "admin"

// Event is a server-sent event.
type Event struct {
	Type      string `json:"type"`
	Data      any    `json:"data"`
	AdminOnly bool   `json:"-"`
}

// Hub manages SSE client connections and broadcasts events.
type Hub struct {
	clients    map[chan Event]string
	mu         sync.RWMutex
	maxClients int
}

// NewHub creates a new SSE hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[chan Event]string),
		maxClients: 100, // Limit concurrent SSE connections
	}
}

// Broadcast sends an event to all connected clients.
func (h *Hub) Broadcast(e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch, role := range h.clients {
		if e.AdminOnly && role != adminRole {
			continue
		}
		select {
		case ch <- e:
		default:
			// Client buffer full, skip
		}
	}
}

// ServeHTTP handles SSE connections.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.ServeHTTPWithRole(w, r, "")
}

// ServeHTTPWithRole handles SSE connections and records the caller role for event filtering.
func (h *Hub) ServeHTTPWithRole(w http.ResponseWriter, r *http.Request, role string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Enforce connection limit
	h.mu.Lock()
	if len(h.clients) >= h.maxClients {
		h.mu.Unlock()
		http.Error(w, "too many SSE connections", http.StatusServiceUnavailable)
		return
	}
	ch := make(chan Event, 4)
	h.clients[ch] = role
	clientCount := len(h.clients)
	h.mu.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
		close(ch)
	}()

	slog.Debug("SSE client connected", "clients", clientCount, "role", role)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ch:
			data, err := json.Marshal(event.Data)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()
		}
	}
}

// ClientCount returns the current number of connected SSE clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
