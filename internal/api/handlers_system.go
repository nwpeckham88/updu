package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/updu/updu/internal/updater"
)

var (
	checkForUpdateForChannel = updater.CheckForUpdateForChannel
	downloadAndApplyUpdate   = updater.DownloadAndApply
)

// handleGetMetrics returns system-wide metrics such as monitor counts.
func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := s.db.GetSystemMetrics(r.Context())
	if err != nil {
		jsonError(w, "failed to get system metrics", http.StatusInternalServerError)
		return
	}
	jsonOK(w, metrics)
}

// handleCheckUpdate queries GitHub for the latest release and reports
// whether an update is available.
func (s *Server) handleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	info, err := checkForUpdateForChannel(s.updateChannel(r.Context()))
	if err != nil {
		slog.Warn("update check failed", "error", err)
		jsonError(w, "update check failed", http.StatusBadGateway)
		return
	}
	jsonOK(w, info)
}

// handleApplyUpdate downloads and applies the latest release. The process
// should be restarted (e.g. by systemd) to run the new version.
func (s *Server) handleApplyUpdate(w http.ResponseWriter, r *http.Request) {
	info, err := checkForUpdateForChannel(s.updateChannel(r.Context()))
	if err != nil {
		jsonError(w, "update check failed", http.StatusBadGateway)
		return
	}

	if !info.UpdateAvailable {
		jsonOK(w, map[string]any{
			"message": "already up to date",
			"version": info.CurrentVersion,
		})
		return
	}

	if err := downloadAndApplyUpdate(info); err != nil {
		slog.Error("update apply failed", "error", err)
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{
		"message":     "update applied",
		"new_version": info.LatestVersion,
		"restart":     "restart the process or service to use the new version",
	})
}

func (s *Server) updateChannel(ctx context.Context) string {
	channel, err := s.db.GetSetting(ctx, "update_channel")
	if err != nil {
		slog.Warn("failed to load update channel", "error", err)
		return ""
	}
	return updater.NormalizeReleaseChannel(channel)
}
