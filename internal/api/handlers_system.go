package api

import (
	"log/slog"
	"net/http"

	"github.com/updu/updu/internal/updater"
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
	info, err := updater.CheckForUpdate()
	if err != nil {
		slog.Warn("update check failed", "error", err)
		jsonError(w, "update check failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	jsonOK(w, info)
}

// handleApplyUpdate downloads and applies the latest release. The process
// should be restarted (e.g. by systemd) to run the new version.
func (s *Server) handleApplyUpdate(w http.ResponseWriter, r *http.Request) {
	info, err := updater.CheckForUpdate()
	if err != nil {
		jsonError(w, "update check failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	if !info.UpdateAvailable {
		jsonOK(w, map[string]any{
			"message": "already up to date",
			"version": info.CurrentVersion,
		})
		return
	}

	if err := updater.DownloadAndApply(info); err != nil {
		slog.Error("update apply failed", "error", err)
		jsonError(w, "update failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{
		"message":     "update applied",
		"new_version": info.LatestVersion,
		"restart":     "restart the process or service to use the new version",
	})
}
