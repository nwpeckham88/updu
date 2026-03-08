package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
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
			// Update existing monitor
			slog.Debug("updating monitor from gitops", "name", m.Name, "id", m.ID)
			m.CreatedAt = existing.CreatedAt
			m.UpdatedAt = now
			m.CreatedBy = existing.CreatedBy
			if err := db.UpdateMonitor(ctx, m); err != nil {
				return fmt.Errorf("updating monitor %s: %w", m.Name, err)
			}
		}

		// For "push" monitors, automatically create/update heartbeat record
		if m.Type == "push" {
			var config models.PushMonitorConfig
			_ = json.Unmarshal(m.Config, &config)
			if config.Token == "" {
				token, _ := generateID()
				config.Token = token
				m.Config, _ = json.Marshal(config)
				// Update the monitor with the generated token
				_ = db.UpdateMonitor(ctx, m)
			}

			h := &models.Heartbeat{
				Slug:      m.ID, // Use monitor ID as the default slug
				MonitorID: m.ID,
				Token:     config.Token,
				ExpectedS: m.IntervalS,
				GraceS:    300, // Default grace period
			}
			_ = db.UpsertHeartbeat(ctx, h)
		}
	}
	return nil
}

func generateDeterministicID(name, typ string) string {
	h := sha1.New()
	h.Write([]byte(name + "|" + typ))
	return hex.EncodeToString(h.Sum(nil))[:12] // Use first 12 chars
}

// generateID generates a secure random 16-byte hex ID.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
