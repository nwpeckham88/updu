package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/notifier"
	"github.com/updu/updu/internal/realtime"
	"github.com/updu/updu/internal/storage"
)

func setupSchedulerTest(t *testing.T) (*Scheduler, *storage.DB, func()) {
	tempDir, err := os.MkdirTemp("", "updu-sched-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Enable debug logging for tests
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	reg := checker.NewRegistry(true, nil)
	sse := realtime.NewHub()
	n := notifier.New(db)

	sched := New(db, reg, sse, n, 5)
	sched.DisableStagger = true

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return sched, db, cleanup
}

func TestScheduler_Lifecycle(t *testing.T) {
	sched, db, cleanup := setupSchedulerTest(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched.Start(ctx)
	defer sched.Stop()

	// Setup a mock monitor
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	m := &models.Monitor{
		ID:        "mon-1",
		Name:      "Test",
		Type:      "http",
		Config:    json.RawMessage(`{"url":"` + ts.URL + `"}`),
		IntervalS: 1,
		TimeoutS:  1,
		Enabled:   true,
		CreatedBy: "user-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.CreateMonitor(ctx, m)
	sched.AddMonitor(ctx, m)

	// Run check synchronously
	sched.RunCheckSync(ctx, m)

	// Verify check result was stored with retry
	var checks []*models.CheckResult
	for i := 0; i < 20; i++ {
		checks, _ = db.GetRecentChecks(ctx, "mon-1", 1)
		if len(checks) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if len(checks) == 0 {
		t.Error("expected at least one check result to be stored")
	}
}

func TestScheduler_AddRemove(t *testing.T) {
	sched, _, cleanup := setupSchedulerTest(t)
	defer cleanup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched.Start(ctx)
	defer sched.Stop()

	m := &models.Monitor{ID: "mon-1", Name: "Test", Type: "http", IntervalS: 60}

	sched.AddMonitor(ctx, m)
	if len(sched.monitors) != 1 {
		t.Errorf("expected 1 monitor, got %d", len(sched.monitors))
	}

	sched.RemoveMonitor("mon-1")
	if len(sched.monitors) != 0 {
		t.Errorf("expected 0 monitors, got %d", len(sched.monitors))
	}
}

func TestScheduler_StartStop(t *testing.T) {
	_, db, cleanup := setupSchedulerTest(t)
	defer cleanup()

	// Add dummy monitors
	ctx := context.Background()
	m1 := &models.Monitor{ID: "m1", Name: "M1", Type: "http", Config: []byte(`{"url":"http://test"}`), Enabled: true, IntervalS: 60}
	m2 := &models.Monitor{ID: "m2", Name: "M2", Type: "ping", Config: []byte(`{"host":"8.8.8.8"}`), Enabled: false, IntervalS: 60}
	db.CreateMonitor(ctx, m1)
	db.CreateMonitor(ctx, m2)

	// Since New() requires parameters we get from setupSchedulerTest, we just recreate it manually
	reg := checker.NewRegistry(true, nil)
	sse := realtime.NewHub()
	n := notifier.New(db)

	sched := New(db, reg, sse, n, 5)

	err := sched.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Wait briefly to allow processing
	time.Sleep(50 * time.Millisecond)

	sched.Stop()

	// Should be able to call Stop safely again (optional safety check)
	sched.Stop()
}

func TestScheduler_ReloadMonitor(t *testing.T) {
	sched, _, cleanup := setupSchedulerTest(t)
	defer cleanup()
	ctx := context.Background()

	m1 := &models.Monitor{ID: "mon-1", Name: "M1", Type: "http", Config: []byte(`{}`), Enabled: true, IntervalS: 60}

	// Start scheduler to ensure goroutines are running
	sched.Start(ctx)
	defer sched.Stop()

	// Initial add
	sched.AddMonitor(ctx, m1)

	// Modified monitor
	m2 := *m1
	m2.Name = "M1-Updated"

	// Reload
	sched.ReloadMonitor(ctx, &m2)

	// Verify
	sched.mu.RLock()
	task, exists := sched.monitors["mon-1"]
	sched.mu.RUnlock()

	if !exists {
		t.Fatal("expected monitor to exist after reload")
	}
	if task.monitor.Name != "M1-Updated" {
		t.Errorf("expected monitor name to be updated, got %s", task.monitor.Name)
	}

	// Reload non-existent monitor
	m3 := &models.Monitor{ID: "mon-nonexistent", Name: "New", Type: "ping", Config: []byte(`{}`), Enabled: true, IntervalS: 60}
	sched.ReloadMonitor(ctx, m3)

	sched.mu.RLock()
	_, exists3 := sched.monitors["mon-nonexistent"]
	sched.mu.RUnlock()

	if !exists3 {
		t.Fatal("expected nonexistent monitor to be added on reload")
	}
}
