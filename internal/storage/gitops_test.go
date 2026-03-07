package storage

import (
	"context"
	"os"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestSyncMonitors(t *testing.T) {
	dbPath := "test_sync.db"
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	// Initial sync
	monitors := []*models.Monitor{
		{
			ID:        "m1",
			Name:      "Monitor 1",
			Type:      "http",
			Groups:    []string{"Core"},
			IntervalS: 60,
			CreatedBy: "gitops",
			Config:    []byte("{}"),
		},
		{
			ID:        "m2",
			Name:      "Monitor 2",
			Type:      "tcp",
			Groups:    []string{"Infra"},
			IntervalS: 60,
			CreatedBy: "gitops",
			Config:    []byte("{}"),
		},
	}

	if err := db.SyncMonitors(ctx, monitors); err != nil {
		t.Fatalf("SyncMonitors failed: %v", err)
	}

	// Verify
	got, err := db.ListMonitors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 monitors, got %d", len(got))
	}

	// Update sync (change m1, remove m2, add m3)
	monitors = []*models.Monitor{
		{
			ID:        "m1",
			Name:      "Monitor 1 Updated",
			Type:      "http",
			Groups:    []string{"Core", "New"},
			IntervalS: 30,
			CreatedBy: "gitops",
			Config:    []byte("{}"),
		},
		{
			ID:        "m3",
			Name:      "Monitor 3",
			Type:      "ping",
			Groups:    []string{"Core"},
			IntervalS: 60,
			CreatedBy: "gitops",
			Config:    []byte("{}"),
		},
	}

	if err := db.SyncMonitors(ctx, monitors); err != nil {
		t.Fatalf("SyncMonitors failed: %v", err)
	}

	// Verify
	got, err = db.ListMonitors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 monitors (orphans are not deleted), got %d", len(got))
	}

	// Check m1 update
	var m1 *models.Monitor
	for _, m := range got {
		if m.ID == "m1" {
			m1 = m
		}
	}
	if m1 == nil || m1.Name != "Monitor 1 Updated" || m1.IntervalS != 30 || len(m1.Groups) != 2 {
		t.Errorf("m1 not updated correctly: %+v", m1)
	}

	// Check deterministic ID generator
	id1 := generateDeterministicID("https://example.com", "http")
	id2 := generateDeterministicID("https://example.com", "http")
	if id1 != id2 {
		t.Error("deterministic ID failed")
	}
}
