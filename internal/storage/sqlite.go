package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_initial.sql
var migration001 string

//go:embed migrations/002_settings.sql
var migration002 string

// DB wraps a sql.DB with updu-specific methods.
type DB struct {
	*sql.DB
}

// Open creates or opens a SQLite database at the given path.
func Open(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache_size=-2000&_temp_store=MEMORY", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// SQLite should only use one connection for writes
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	slog.Info("database opened", "path", dbPath)
	return &DB{db}, nil
}

// Migrate runs all pending migrations.
func (db *DB) Migrate(ctx context.Context) error {
	// Create migrations tracking table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}
	migrations := []struct {
		version int
		sql     string
	}{
		{1, migration001},
		{2, migration002},
	}

	for _, m := range migrations {
		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", m.version).Scan(&count)
		if err != nil {
			return fmt.Errorf("checking migration %d: %w", m.version, err)
		}
		if count > 0 {
			continue
		}

		slog.Info("applying migration", "version", m.version)
		if _, err := db.ExecContext(ctx, m.sql); err != nil {
			return fmt.Errorf("applying migration %d: %w", m.version, err)
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
			return fmt.Errorf("recording migration %d: %w", m.version, err)
		}
	}

	slog.Info("migrations complete")
	return nil
}
