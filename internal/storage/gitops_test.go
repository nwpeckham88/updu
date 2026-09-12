package storage

import (
	"context"
	"os"
	"path/filepath"
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

func TestSyncServices(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "updu-sync-services-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	services := []*models.Service{
		{
			ID:     "srv-gitops-1",
			Name:   "GitOps Service",
			Type:   "web",
			ZoneID: "homelab",
			Groups: []string{"Internal"},
			Tags:   []string{"ha"},
			Endpoints: []*models.ServiceEndpoint{
				{
					ID:         "ep-gitops-lan",
					Name:       "LAN Endpoint",
					ScopeID:    "lan",
					TargetType: "http",
					Config:     []byte(`{"url":"http://192.168.1.100:80"}`),
					IntervalS:  10,
					TimeoutS:   5,
					IsPrimary:  false,
				},
				{
					ID:         "ep-gitops-wan",
					Name:       "Public Endpoint",
					ScopeID:    "public",
					TargetType: "http",
					Config:     []byte(`{"url":"https://example.com"}`),
					IntervalS:  30,
					TimeoutS:   10,
					IsPrimary:  true,
				},
			},
		},
	}

	if err := db.SyncServices(ctx, services); err != nil {
		t.Fatalf("SyncServices failed: %v", err)
	}

	// 1. Verify service exists
	svc, err := db.GetService(ctx, "srv-gitops-1")
	if err != nil || svc == nil {
		t.Fatalf("expected service to be created, err=%v, svc=%v", err, svc)
	}
	if svc.Name != "GitOps Service" || svc.ZoneID != "homelab" {
		t.Errorf("unexpected service data: %+v", svc)
	}
	if len(svc.Endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(svc.Endpoints))
	}

	// 2. Verify corresponding monitor was automatically created for scheduler
	mon, err := db.GetMonitor(ctx, "srv-gitops-1")
	if err != nil || mon == nil {
		t.Fatalf("expected monitor to be auto-created for scheduler, err=%v, mon=%v", err, mon)
	}
	if mon.Name != "GitOps Service" || mon.IntervalS != 30 {
		t.Errorf("unexpected monitor data: %+v", mon)
	}
}

