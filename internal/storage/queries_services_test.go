package storage_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/storage"
)

func setupTestDB(t *testing.T) (*storage.DB, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "updu-test-services-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("opening test db: %v", err)
	}

	if err := db.Migrate(context.Background()); err != nil {
		db.Close()
		os.RemoveAll(dir)
		t.Fatalf("migrating test db: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}
	return db, cleanup
}

func TestServicesCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 1. Create Service with endpoints
	svc := &models.Service{
		ID:     "srv-vaultwarden",
		Name:   "Vaultwarden",
		Type:   models.ServiceTypeWeb,
		ZoneID: "default",
		Groups: []string{"Security", "Self-Hosted"},
		Tags:   []string{"critical"},
		Endpoints: []*models.ServiceEndpoint{
			{
				ID:         "ep-lan",
				Name:       "LAN Container",
				ScopeID:    models.ScopeLAN,
				TargetType: "http",
				Config:     json.RawMessage(`{"url":"http://192.168.1.50:8080"}`),
				IntervalS:  15,
				IsPrimary:  false,
			},
			{
				ID:         "ep-wan",
				Name:       "Public Domain",
				ScopeID:    models.ScopePublic,
				TargetType: "http",
				Config:     json.RawMessage(`{"url":"https://vault.kn8.design"}`),
				IntervalS:  30,
				IsPrimary:  true,
			},
		},
	}

	if err := db.CreateService(ctx, svc); err != nil {
		t.Fatalf("CreateService failed: %v", err)
	}

	// 2. Get Service
	got, err := db.GetService(ctx, "srv-vaultwarden")
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected service, got nil")
	}
	if got.Name != "Vaultwarden" || len(got.Endpoints) != 2 {
		t.Errorf("service mismatch: name=%s, endpoints=%d", got.Name, len(got.Endpoints))
	}

	// 3. Record Endpoint Checks
	lat := 12
	code := 200
	check := &models.EndpointCheck{
		ServiceID:  "srv-vaultwarden",
		EndpointID: "ep-wan",
		NodeID:     "local",
		Status:     models.StatusUp,
		LatencyMs:  &lat,
		StatusCode: &code,
		Message:    "OK",
	}
	if err := db.RecordEndpointCheck(ctx, check); err != nil {
		t.Fatalf("RecordEndpointCheck failed: %v", err)
	}

	// 4. List endpoints and verify check enrichment
	endpoints, err := db.ListServiceEndpoints(ctx, "srv-vaultwarden")
	if err != nil {
		t.Fatalf("ListServiceEndpoints failed: %v", err)
	}
	var primaryFound bool
	for _, ep := range endpoints {
		if ep.ID == "ep-wan" {
			primaryFound = true
			if ep.Status != models.StatusUp || ep.LastLatency == nil || *ep.LastLatency != 12 {
				t.Errorf("endpoint enrichment mismatch: status=%s, latency=%v", ep.Status, ep.LastLatency)
			}
		}
	}
	if !primaryFound {
		t.Error("primary endpoint ep-wan not found")
	}

	// 5. Update Service
	got.Name = "Vaultwarden Pro"
	if err := db.UpdateService(ctx, got); err != nil {
		t.Fatalf("UpdateService failed: %v", err)
	}
	updated, _ := db.GetService(ctx, "srv-vaultwarden")
	if updated.Name != "Vaultwarden Pro" {
		t.Errorf("expected updated name, got %s", updated.Name)
	}

	// 6. Delete Service
	if err := db.DeleteService(ctx, "srv-vaultwarden"); err != nil {
		t.Fatalf("DeleteService failed: %v", err)
	}
	deleted, _ := db.GetService(ctx, "srv-vaultwarden")
	if deleted != nil {
		t.Error("expected service to be deleted")
	}
}

func TestTLSCertificates(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	cert := &models.TLSCertificate{
		ID:             "cert-kn8-design",
		Domain:         "vault.kn8.design",
		Issuer:         "Let's Encrypt R3",
		Subject:        "CN=vault.kn8.design",
		SANs:           []string{"vault.kn8.design", "bitwarden.kn8.design"},
		ValidFrom:      now.AddDate(0, -1, 0),
		ValidUntil:     now.AddDate(0, 2, 0),
		DaysRemaining:  60,
		SerialNumber:   "03A1B2C3",
		LastVerifiedAt: now,
	}

	if err := db.UpsertTLSCertificate(ctx, cert); err != nil {
		t.Fatalf("UpsertTLSCertificate failed: %v", err)
	}

	certs, err := db.ListTLSCertificates(ctx)
	if err != nil {
		t.Fatalf("ListTLSCertificates failed: %v", err)
	}
	if len(certs) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(certs))
	}
	if certs[0].Domain != "vault.kn8.design" || len(certs[0].SANs) != 2 {
		t.Errorf("cert data mismatch: %v", certs[0])
	}
}

func TestNetworkTopology(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	svc := &models.Service{
		ID:     "srv-dns",
		Name:   "Google DNS",
		Type:   models.ServiceTypeInfra,
		ZoneID: "default",
		Endpoints: []*models.ServiceEndpoint{
			{
				ID:         "ep-dns-1",
				Name:       "Primary 8.8.8.8",
				ScopeID:    models.ScopePublic,
				TargetType: "dns",
				Config:     json.RawMessage(`{"host":"8.8.8.8"}`),
			},
		},
	}
	_ = db.CreateService(ctx, svc)

	topo, err := db.GetNetworkTopology(ctx, "local-node", "My Primary Hub")
	if err != nil {
		t.Fatalf("GetNetworkTopology failed: %v", err)
	}
	if len(topo.Nodes) != 1 || topo.Nodes[0].Name != "My Primary Hub" {
		t.Errorf("nodes mismatch: %v", topo.Nodes)
	}
	if len(topo.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(topo.Services))
	}
	if len(topo.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(topo.Edges))
	}
}
