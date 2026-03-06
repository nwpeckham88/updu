package storage

import (
	"context"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestEventQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	monitorID := "mon-event-test"

	// Create a monitor first
	m := &models.Monitor{
		ID:        monitorID,
		Name:      "Event Test Monitor",
		Type:      "http",
		Config:    []byte(`{}`),
		CreatedBy: "user-1",
	}
	if err := db.CreateMonitor(ctx, m); err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	// 1. CreateEvent
	now := time.Now().Truncate(time.Second) // SQLite dates might lose sub-second precision
	e1 := &models.Event{
		MonitorID: monitorID,
		Status:    models.StatusDown,
		Message:   "Connection refused",
		CreatedAt: now.Add(-10 * time.Minute),
	}
	if err := db.CreateEvent(ctx, e1); err != nil {
		t.Fatalf("failed to create event e1: %v", err)
	}

	e2 := &models.Event{
		MonitorID: monitorID,
		Status:    models.StatusUp,
		Message:   "OK",
		CreatedAt: now,
	}
	if err := db.CreateEvent(ctx, e2); err != nil {
		t.Fatalf("failed to create event e2: %v", err)
	}

	// 2. ListRecentEvents
	events, err := db.ListRecentEvents(ctx, 5)
	if err != nil {
		t.Fatalf("failed to list recent events: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 recent events, got %d", len(events))
	}
	// e2 was created more recently, should be first
	if events[0].Status != models.StatusUp {
		t.Errorf("expected newest event to be Up, got %s", events[0].Status)
	}

	// 3. ListEventsByMonitor
	monitorEvents, err := db.ListEventsByMonitor(ctx, monitorID, 10)
	if err != nil {
		t.Fatalf("failed to list events by monitor: %v", err)
	}
	if len(monitorEvents) != 2 {
		t.Errorf("expected 2 events for monitor %s, got %d", monitorID, len(monitorEvents))
	}

	// 4. ListEventsByMonitor with limit
	limitedEvents, err := db.ListEventsByMonitor(ctx, monitorID, 1)
	if err != nil {
		t.Fatalf("failed to list events by monitor with limit: %v", err)
	}
	if len(limitedEvents) != 1 {
		t.Errorf("expected 1 event with limit 1, got %d", len(limitedEvents))
	}
	if limitedEvents[0].Status != models.StatusUp {
		t.Errorf("expected the most recent event from limit 1")
	}
}
