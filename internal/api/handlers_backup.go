package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/updu/updu/internal/config"
	"github.com/updu/updu/internal/models"
	"gopkg.in/yaml.v3"
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

func (s *Server) handleExportYAML(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	monitors, err := s.db.ListMonitors(ctx)
	if err != nil {
		jsonError(w, "failed to get monitors", http.StatusInternalServerError)
		return
	}

	settings, err := s.db.ListSettings(ctx)
	if err != nil {
		jsonError(w, "failed to get settings", http.StatusInternalServerError)
		return
	}

	// Add runtime config settings to the map if not in DB
	if _, ok := settings["host"]; !ok {
		settings["host"] = s.config.Host
	}
	if _, ok := settings["port"]; !ok {
		settings["port"] = strconv.Itoa(s.config.Port)
	}
	if _, ok := settings["base_url"]; !ok {
		settings["base_url"] = s.config.BaseURL
	}
	if _, ok := settings["db_path"]; !ok {
		settings["db_path"] = s.config.DBPath
	}
	if _, ok := settings["log_level"]; !ok {
		settings["log_level"] = s.config.LogLevel
	}
	if _, ok := settings["trusted_proxy_cidrs"]; !ok && len(s.config.TrustedProxyCIDRs) > 0 {
		settings["trusted_proxy_cidrs"] = strings.Join(s.config.TrustedProxyCIDRs, ",")
	}

	yCfg := config.FromModels(monitors, settings)

	data, err := yaml.Marshal(yCfg)
	if err != nil {
		jsonError(w, "failed to marshal YAML", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=exported.updu.conf")
	w.Write(data)
}

func (s *Server) handleImportConfig(w http.ResponseWriter, r *http.Request) {
	var backup BackupConfig
	if err := json.NewDecoder(r.Body).Decode(&backup); err != nil {
		jsonError(w, "invalid backup format", http.StatusBadRequest)
		return
	}

	// Validate imported data
	for _, m := range backup.Monitors {
		if m.Name == "" {
			jsonError(w, "monitor missing name", http.StatusBadRequest)
			return
		}
		if m.Type == "" {
			jsonError(w, "monitor missing type: "+m.Name, http.StatusBadRequest)
			return
		}
		c := s.registry.Get(m.Type)
		if c == nil {
			jsonError(w, "unknown monitor type: "+m.Type, http.StatusBadRequest)
			return
		}
		if err := c.Validate(m.Config); err != nil {
			jsonError(w, "invalid config for "+m.Name+": "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	for k := range backup.Settings {
		if !allowedSettingsKeys[k] {
			jsonError(w, "unknown setting key in backup: "+k, http.StatusBadRequest)
			return
		}
	}
	// Sanitize custom CSS if present
	if css, ok := backup.Settings["custom_css"]; ok {
		backup.Settings["custom_css"] = sanitizeCSS(css)
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
	s.recordAudit(r, "system.import", "system_backup", "import", "imported configuration backup")

	jsonOK(w, map[string]any{
		"message": "configuration imported",
		"errors":  errCount,
	})
}
