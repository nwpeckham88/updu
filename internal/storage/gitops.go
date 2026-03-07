package storage

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/updu/updu/internal/models"
)

// SyncMonitors synchronizes monitors from a list (e.g., from YAML config) into the database.
// If a monitor with the same ID already exists, it updates it.
// If no ID is provided, it generates a deterministic ID based on the Name and Type.
func (db *DB) SyncMonitors(ctx context.Context, monitors []*models.Monitor) error {
	for _, m := range monitors {
		// Generate deterministic ID if missing
		if m.ID == "" {
			m.ID = generateDeterministicID(m.Name, m.Type)
		}

		existing, err := db.GetMonitor(ctx, m.ID)
		if err != nil {
			return fmt.Errorf("checking existing monitor %s: %w", m.Name, err)
		}

		now := time.Now()
		if existing == nil {
			slog.Info("creating monitor from gitops", "name", m.Name, "id", m.ID)
			m.CreatedAt = now
			m.UpdatedAt = now
			m.CreatedBy = "system"
			if err := db.CreateMonitor(ctx, m); err != nil {
				return fmt.Errorf("creating monitor %s: %w", m.Name, err)
			}
		} else {
			// Update existing monitor if it has changed (or just always update for now)
			slog.Debug("updating monitor from gitops", "name", m.Name, "id", m.ID)
			m.CreatedAt = existing.CreatedAt
			m.UpdatedAt = now
			m.CreatedBy = existing.CreatedBy
			if err := db.UpdateMonitor(ctx, m); err != nil {
				return fmt.Errorf("updating monitor %s: %w", m.Name, err)
			}
		}
	}
	return nil
}

func generateDeterministicID(name, typ string) string {
	h := sha1.New()
	h.Write([]byte(name + "|" + typ))
	return hex.EncodeToString(h.Sum(nil))[:12] // Use first 12 chars
}
