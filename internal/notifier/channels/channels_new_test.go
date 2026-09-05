package channels_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/notifier/channels"
)

func TestNewNotificationChannels(t *testing.T) {
	var receivedPath string
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	mon := &models.Monitor{
		ID:   "mon-test",
		Name: "Test Target",
		Type: "http",
	}
	event := &models.Event{
		MonitorID: "mon-test",
		Status:    models.StatusDown,
		Message:   "Connection refused",
		CreatedAt: time.Now(),
	}

	ctx := context.WithValue(context.Background(), channels.AllowLocalhostKey, true)

	// 1. Telegram
	tg := channels.NewTelegramChannel()
	if tg.Type() != "telegram" {
		t.Errorf("expected telegram, got %s", tg.Type())
	}
	// Missing token test
	err := tg.Send(ctx, mon, event, map[string]any{})
	if err == nil {
		t.Error("expected error for missing token")
	}

	// 2. Pushover
	po := channels.NewPushoverChannel()
	if po.Type() != "pushover" {
		t.Errorf("expected pushover, got %s", po.Type())
	}
	err = po.Send(ctx, mon, event, map[string]any{})
	if err == nil {
		t.Error("expected error for missing user_key")
	}

	// 3. Matrix
	mx := channels.NewMatrixChannel()
	if mx.Type() != "matrix" {
		t.Errorf("expected matrix, got %s", mx.Type())
	}
	err = mx.Send(ctx, mon, event, map[string]any{
		"homeserver_url": server.URL,
		"room_id":        "!testroom:matrix.org",
		"access_token":   "syt_test_token",
	})
	if err != nil {
		t.Errorf("matrix Send failed: %v", err)
	}
	if receivedAuth != "Bearer syt_test_token" {
		t.Errorf("expected matrix auth header, got %s", receivedAuth)
	}

	// 4. Apprise
	apprise := channels.NewAppriseChannel()
	if apprise.Type() != "apprise" {
		t.Errorf("expected apprise, got %s", apprise.Type())
	}
	err = apprise.Send(ctx, mon, event, map[string]any{
		"url": server.URL + "/notify",
	})
	if err != nil {
		t.Errorf("apprise Send failed: %v", err)
	}
	if receivedPath != "/notify" {
		t.Errorf("expected /notify, got %s", receivedPath)
	}
}
