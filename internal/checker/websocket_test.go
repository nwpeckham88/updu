package checker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/updu/updu/internal/models"
	"nhooyr.io/websocket"
)

func TestWebSocketChecker(t *testing.T) {
	c := &WebSocketChecker{}
	if c.Type() != "websocket" {
		t.Errorf("Type() = %v, want websocket", c.Type())
	}

	// Test Validate
	validCfg := []byte(`{"url": "wss://echo.websocket.org"}`)
	if err := c.Validate(validCfg); err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if err := c.Validate([]byte(`{}`)); err == nil {
		t.Errorf("Expected error for empty url")
	}
	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Errorf("Expected error for bad json")
	}

	// Test Check
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
	}))
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

	monitor := &models.Monitor{
		ID:        "ws-1",
		IntervalS: 60,
		Config:    json.RawMessage(`{"url": "` + wsURL + `"}`),
	}

	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)
	res, err := c.Check(ctx, monitor)
	if err != nil {
		t.Errorf("Expected nil err, got %v", err)
	}
	if res.Status != models.StatusUp {
		t.Errorf("Expected StatusUp, got %v - %s", res.Status, res.Message)
	}

	// Test Down
	monitor.Config = json.RawMessage(`{"url": "ws://localhost:12345"}`)
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Errorf("Expected StatusDown, got %v", res.Status)
	}

	// Test invalid config in Check
	monitor.Config = json.RawMessage(`{bad`)
	res, _ = c.Check(ctx, monitor)
	if res.Message != "Invalid monitor configuration" {
		t.Errorf("Expected config error message, got %v", res.Message)
	}
}
