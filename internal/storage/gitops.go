package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
				GraceS:    config.EffectiveGraceSeconds(m.IntervalS),
			}
			if err := db.UpsertHeartbeat(ctx, h); err != nil {
				slog.Warn("failed to sync push heartbeat from gitops", "monitor_id", m.ID, "error", err)
			}
		}
	}
	return nil
}

// SyncServices synchronizes services and their endpoints from a list (e.g., from YAML config) into the database.
func (db *DB) SyncServices(ctx context.Context, services []*models.Service) error {
	for _, s := range services {
		if s.ID == "" {
			s.ID = generateDeterministicID(s.Name, s.Type)
		}

		// Ensure zone exists or fallback to default
		if s.ZoneID == "" {
			s.ZoneID = "default"
		}
		_, _ = db.ExecContext(ctx, "INSERT OR IGNORE INTO zones (id, name, description) VALUES (?, ?, ?)", s.ZoneID, s.ZoneID, "Configured zone")

		now := time.Now()
		existing, err := db.GetService(ctx, s.ID)
		if err != nil {
			return fmt.Errorf("checking existing service %s: %w", s.Name, err)
		}

		if existing == nil {
			slog.Info("creating service from gitops", "name", s.Name, "id", s.ID)
			s.CreatedAt = now
			s.UpdatedAt = now
			s.CreatedBy = "system"
			if err := db.CreateService(ctx, s); err != nil {
				return fmt.Errorf("creating service %s: %w", s.Name, err)
			}
		} else {
			slog.Debug("updating service from gitops", "name", s.Name, "id", s.ID)
			s.CreatedAt = existing.CreatedAt
			s.UpdatedAt = now
			s.CreatedBy = existing.CreatedBy
			if err := db.UpdateService(ctx, s); err != nil {
				return fmt.Errorf("updating service %s: %w", s.Name, err)
			}

			// Upsert endpoints
			for i, ep := range s.Endpoints {
				ep.ServiceID = s.ID
				if ep.ID == "" {
					ep.ID = fmt.Sprintf("%s-ep-%d", s.ID, i)
				}
				if ep.CreatedAt.IsZero() {
					ep.CreatedAt = now
				}
				if err := db.CreateServiceEndpoint(ctx, ep); err != nil {
					return fmt.Errorf("syncing endpoint %s for service %s: %w", ep.Name, s.Name, err)
				}
			}
		}

		// Ensure a corresponding monitor exists for the service's primary endpoint so scheduler monitors it
		var primaryEp *models.ServiceEndpoint
		for _, ep := range s.Endpoints {
			if ep.IsPrimary {
				primaryEp = ep
				break
			}
		}
		if primaryEp == nil && len(s.Endpoints) > 0 {
			primaryEp = s.Endpoints[0]
		}

		if primaryEp != nil {
			mon, err := db.GetMonitor(ctx, s.ID)
			if err != nil {
				return fmt.Errorf("checking monitor for service %s: %w", s.Name, err)
			}
			if mon == nil {
				newMon := &models.Monitor{
					ID:        s.ID,
					Name:      s.Name,
					Type:      primaryEp.TargetType,
					Config:    primaryEp.Config,
					IntervalS: primaryEp.IntervalS,
					TimeoutS:  primaryEp.TimeoutS,
					Retries:   primaryEp.Retries,
					Enabled:   s.Enabled,
					Groups:    s.Groups,
					Tags:      s.Tags,
					CreatedBy: "system",
					CreatedAt: now,
					UpdatedAt: now,
				}
				_ = db.CreateMonitor(ctx, newMon)
			} else {
				mon.Name = s.Name
				mon.Type = primaryEp.TargetType
				mon.Config = primaryEp.Config
				if primaryEp.IntervalS > 0 {
					mon.IntervalS = primaryEp.IntervalS
				}
				if primaryEp.TimeoutS > 0 {
					mon.TimeoutS = primaryEp.TimeoutS
				}
				mon.Retries = primaryEp.Retries
				mon.Enabled = s.Enabled
				mon.Groups = s.Groups
				mon.Tags = s.Tags
				mon.UpdatedAt = now
				_ = db.UpdateMonitor(ctx, mon)
			}
		}
	}
	return nil
}

func generateDeterministicID(name, typ string) string {
	h := sha256.New()
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
