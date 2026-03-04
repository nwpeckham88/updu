package api

import (
	"encoding/json"
	"net/http"

	"github.com/updu/updu/internal/models"
)

type BackupConfig struct {
	Monitors             []*models.Monitor             `json:"monitors"`
	Incidents            []*models.Incident            `json:"incidents"`
	MaintenanceWindows   []*models.MaintenanceWindow   `json:"maintenance_windows"`
	NotificationChannels []*models.NotificationChannel `json:"notification_channels"`
	Settings             map[string]string             `json:"settings"`
}

func (s *Server) handleExportConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	monitors, err := s.db.ListMonitors(ctx)
	if err != nil {
		jsonError(w, "failed to export monitors", http.StatusInternalServerError)
		return
	}

	incidents, err := s.db.ListIncidents(ctx)
	if err != nil {
		jsonError(w, "failed to export incidents", http.StatusInternalServerError)
		return
	}

	windows, err := s.db.ListMaintenanceWindows(ctx)
	if err != nil {
		jsonError(w, "failed to export maintenance windows", http.StatusInternalServerError)
		return
	}

	channels, err := s.db.ListNotificationChannels(ctx)
	if err != nil {
		jsonError(w, "failed to export notification channels", http.StatusInternalServerError)
		return
	}

	settings, err := s.db.ListSettings(ctx)
	if err != nil {
		jsonError(w, "failed to export settings", http.StatusInternalServerError)
		return
	}

	backup := BackupConfig{
		Monitors:             monitors,
		Incidents:            incidents,
		MaintenanceWindows:   windows,
		NotificationChannels: channels,
		Settings:             settings,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=updu-backup.json")
	json.NewEncoder(w).Encode(backup)
}

func (s *Server) handleImportConfig(w http.ResponseWriter, r *http.Request) {
	var backup BackupConfig
	if err := json.NewDecoder(r.Body).Decode(&backup); err != nil {
		jsonError(w, "invalid backup format", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	var errCount int

	for _, m := range backup.Monitors {
		if err := s.db.CreateMonitor(ctx, m); err != nil {
			errCount++
		}
	}
	for _, i := range backup.Incidents {
		if err := s.db.CreateIncident(ctx, i); err != nil {
			errCount++
		}
	}
	for _, mw := range backup.MaintenanceWindows {
		if err := s.db.CreateMaintenanceWindow(ctx, mw); err != nil {
			errCount++
		}
	}
	for _, nc := range backup.NotificationChannels {
		if err := s.db.CreateNotificationChannel(ctx, nc); err != nil {
			errCount++
		}
	}
	for k, v := range backup.Settings {
		if err := s.db.SetSetting(ctx, k, v); err != nil {
			errCount++
		}
	}

	jsonOK(w, map[string]any{
		"message": "configuration imported",
		"errors":  errCount,
	})
}
