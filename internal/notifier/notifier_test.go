package notifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/notifier/channels"
	"github.com/updu/updu/internal/storage"
)

func setupNotifierTest(t *testing.T) (*Notifier, *storage.DB, func()) {
	tempDir, err := os.MkdirTemp("", "updu-notifier-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	notifier := New(db)
	notifier.Register(channels.NewWebhookChannel())

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return notifier, db, cleanup
}

func TestNotifier_Notify(t *testing.T) {
	notifier, db, cleanup := setupNotifierTest(t)
	defer cleanup()

	ctx := context.WithValue(context.Background(), channels.AllowLocalhostKey, true)

	// Setup a mock webhook server
	received := make(chan bool, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Create a notification channel in DB
	nc := &models.NotificationChannel{
		ID:      "chan-1",
		Name:    "Test Webhook",
		Type:    "webhook",
		Config:  map[string]any{"url": ts.URL},
		Enabled: true,
	}
	if err := db.CreateNotificationChannel(ctx, nc); err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}

	// Create a disabled channel
	db.CreateNotificationChannel(ctx, &models.NotificationChannel{
		ID: "chan-disabled", Name: "Disabled", Type: "webhook", Config: map[string]any{"url": ts.URL}, Enabled: false,
	})

	// Create an unknown channel type
	db.CreateNotificationChannel(ctx, &models.NotificationChannel{
		ID: "chan-unknown", Name: "Unknown", Type: "unknown", Config: map[string]any{}, Enabled: true,
	})

	monitor := &models.Monitor{ID: "mon-1", Name: "Test Monitor"}
	event := &models.Event{Status: models.StatusDown, Message: "Down!"}

	notifier.Notify(ctx, monitor, event)

	// Wait for notification (it runs in a goroutine)
	select {
	case <-received:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}

	// Wait slightly to let the error paths and goroutines log their errors for coverage
	time.Sleep(100 * time.Millisecond)
}

func TestNotifier_Register(t *testing.T) {
	notifier := New(nil)
	channel := channels.NewWebhookChannel()
	notifier.Register(channel)

	if _, ok := notifier.channels["webhook"]; !ok {
		t.Error("failed to register webhook channel")
	}
}
