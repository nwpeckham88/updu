package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestTicketQueries(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	svcID := "svc_123"
	monID := "mon_456"
	t1 := &models.Ticket{
		ID:          "tkt_1",
		Title:       "Cannot stream Jellyfin",
		Description: "Player buffers endlessly on smart TV",
		Status:      models.TicketStatusOpen,
		Severity:    "high",
		ServiceID:   &svcID,
		ServiceName: "Jellyfin",
		MonitorID:   &monID,
		CreatedBy:   "alice",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	t2 := &models.Ticket{
		ID:          "tkt_2",
		Title:       "Immich upload slow",
		Description: "Photos taking forever to sync",
		Status:      models.TicketStatusOpen,
		Severity:    "medium",
		CreatedBy:   "bob",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.CreateTicket(ctx, t1); err != nil {
		t.Fatalf("CreateTicket t1 failed: %v", err)
	}
	if err := db.CreateTicket(ctx, t2); err != nil {
		t.Fatalf("CreateTicket t2 failed: %v", err)
	}

	// Test GetTicket
	got, err := db.GetTicket(ctx, "tkt_1")
	if err != nil {
		t.Fatalf("GetTicket failed: %v", err)
	}
	if got == nil || got.Title != "Cannot stream Jellyfin" || got.ServiceName != "Jellyfin" {
		t.Fatalf("unexpected ticket: %+v", got)
	}

	// Test ListTickets all
	all, err := db.ListTickets(ctx, "")
	if err != nil {
		t.Fatalf("ListTickets failed: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 tickets, got %d", len(all))
	}

	// Test ListTickets filtered by user
	aliceTickets, err := db.ListTickets(ctx, "alice")
	if err != nil {
		t.Fatalf("ListTickets for alice failed: %v", err)
	}
	if len(aliceTickets) != 1 || aliceTickets[0].ID != "tkt_1" {
		t.Fatalf("expected 1 ticket for alice, got %d", len(aliceTickets))
	}

	// Test UpdateTicket
	now := time.Now()
	t1.Status = models.TicketStatusResolved
	t1.ResolvedAt = &now
	if err := db.UpdateTicket(ctx, t1); err != nil {
		t.Fatalf("UpdateTicket failed: %v", err)
	}

	gotUpdated, err := db.GetTicket(ctx, "tkt_1")
	if err != nil {
		t.Fatalf("GetTicket after update failed: %v", err)
	}
	if gotUpdated.Status != models.TicketStatusResolved || gotUpdated.ResolvedAt == nil {
		t.Fatalf("expected resolved status, got %+v", gotUpdated)
	}

	// Test DeleteTicket
	if err := db.DeleteTicket(ctx, "tkt_2"); err != nil {
		t.Fatalf("DeleteTicket failed: %v", err)
	}
	gotDeleted, err := db.GetTicket(ctx, "tkt_2")
	if err != nil {
		t.Fatalf("GetTicket after delete error: %v", err)
	}
	if gotDeleted != nil {
		t.Fatalf("expected nil after delete, got %+v", gotDeleted)
	}
}
