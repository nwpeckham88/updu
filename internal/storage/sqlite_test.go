package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "updu-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if db.DB == nil {
		t.Error("expected sql.DB to be initialized")
	}

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected database file to exist")
	}
}

func TestMigrate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "updu-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Verify tables exist
	tables := []string{"users", "sessions", "monitors", "check_results", "notification_channels", "status_pages"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}

	// Run migration again to ensure idempotency
	if err := db.Migrate(ctx); err != nil {
		t.Errorf("second migration failed: %v", err)
	}
}

func TestForeignKeysCascade(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "updu-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Verify PRAGMA foreign_keys is 1
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("querying foreign_keys pragma: %v", err)
	}
	if fk != 1 {
		t.Fatalf("expected foreign_keys=1, got %d", fk)
	}

	// Insert monitor
	_, err = db.ExecContext(ctx, "INSERT INTO monitors (id, name, type, config, created_by) VALUES (?, ?, ?, ?, ?)", "m-casc", "Cascade", "http", "{}", "admin")
	if err != nil {
		t.Fatalf("inserting monitor: %v", err)
	}

	// Insert check_result referencing the monitor
	_, err = db.ExecContext(ctx, "INSERT INTO check_results (monitor_id, status, checked_at) VALUES (?, ?, CURRENT_TIMESTAMP)", "m-casc", "up")
	if err != nil {
		t.Fatalf("inserting check result: %v", err)
	}

	// Delete monitor
	_, err = db.ExecContext(ctx, "DELETE FROM monitors WHERE id = ?", "m-casc")
	if err != nil {
		t.Fatalf("deleting monitor: %v", err)
	}

	// Verify check_results were cascade deleted
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM check_results WHERE monitor_id = ?", "m-casc").Scan(&count); err != nil {
		t.Fatalf("counting checks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 checks after cascade delete, got %d", count)
	}
}

func TestMigrateBackfillsMonitorsToServices(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "updu-test-backfill-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("initial migration failed: %v", err)
	}

	// Insert raw monitor bypassing Go layer
	_, err = db.ExecContext(ctx, "INSERT INTO monitors (id, name, type, config, created_by) VALUES (?, ?, ?, ?, ?)", "m-pre9", "Legacy Pre-v9 Monitor", "https", `{"url":"https://legacy.example.com"}`, "admin")
	if err != nil {
		t.Fatalf("inserting legacy monitor: %v", err)
	}

	// Re-run Migrate to trigger backfill
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("subsequent migration failed: %v", err)
	}

	// Verify service was backfilled
	var svcName string
	if err := db.QueryRow("SELECT name FROM services WHERE id = ?", "m-pre9").Scan(&svcName); err != nil {
		t.Fatalf("service was not backfilled: %v", err)
	}
	if svcName != "Legacy Pre-v9 Monitor" {
		t.Errorf("expected backfilled service name 'Legacy Pre-v9 Monitor', got %s", svcName)
	}

	// Verify endpoint was backfilled
	var epName string
	if err := db.QueryRow("SELECT name FROM service_endpoints WHERE id = ?", "m-pre9-default").Scan(&epName); err != nil {
		t.Fatalf("service endpoint was not backfilled: %v", err)
	}
	if epName != "Legacy Pre-v9 Monitor" {
		t.Errorf("expected backfilled endpoint name 'Legacy Pre-v9 Monitor', got %s", epName)
	}
}

