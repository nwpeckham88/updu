package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/updu/updu/internal/updater"
)

var (
	checkForUpdateForChannel = updater.CheckForUpdateForChannel
	downloadAndApplyUpdate   = updater.DownloadAndApply
	scheduleRestart          = updater.ScheduleRestart
)

// restartDelay is how long to wait after a successful self-update before
// sending SIGTERM to ourselves so the supervisor can relaunch the new
// binary. The delay exists so the HTTP response that triggered the update
// has time to reach the client before shutdown begins.
const restartDelay = 2 * time.Second

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
		jsonError(w, fmt.Sprintf("update check failed: %v", err), http.StatusBadGateway)
		return
	}
	jsonOK(w, info)
}

// handleApplyUpdate downloads and applies the latest release, then schedules
// a self-restart so the supervising process (systemd, docker, etc.) launches
// the new binary. Errors from the check, download, checksum, or atomic
// rename steps are surfaced verbatim so an operator can act on them.
func (s *Server) handleApplyUpdate(w http.ResponseWriter, r *http.Request) {
	info, err := checkForUpdateForChannel(s.updateChannel(r.Context()))
	if err != nil {
		slog.Warn("update check failed", "error", err)
		jsonError(w, fmt.Sprintf("update check failed: %v", err), http.StatusBadGateway)
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
		slog.Error("update apply failed", "error", err, "version", info.LatestVersion)
		jsonError(w, fmt.Sprintf("update failed: %v", err), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{
		"message":     "update applied; restarting",
		"new_version": info.LatestVersion,
		"restart":     "process will exit shortly so the supervisor can launch the new version",
	})

	// Flush the response before scheduling SIGTERM so the client actually
	// receives the success payload.
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	scheduleRestart(restartDelay, fmt.Sprintf("self-update to %s", info.LatestVersion))
}

func (s *Server) updateChannel(ctx context.Context) string {
	channel, err := s.db.GetSetting(ctx, "update_channel")
	if err != nil {
		slog.Warn("failed to load update channel", "error", err)
		return ""
	}
	return updater.NormalizeReleaseChannel(channel)
}
