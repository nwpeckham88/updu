package api

import (
	"net/http"
	"time"

	"github.com/updu/updu/internal/models"
)

func (s *Server) handleHeartbeatPing(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}

	h, err := s.db.GetHeartbeat(r.Context(), slug)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if h == nil {
		jsonError(w, "heartbeat not found", http.StatusNotFound)
		return
	}

	// Token verification
	token := r.URL.Query().Get("token")
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" || token != h.Token {
		jsonError(w, "unauthorized: invalid or missing token", http.StatusUnauthorized)
		return
	}

	// Update last ping
	now := time.Now()
	h.LastPing = &now
	if err := s.db.UpsertHeartbeat(r.Context(), h); err != nil {
		jsonError(w, "failed to update heartbeat", http.StatusInternalServerError)
		return
	}

	// If this heartbeat is linked to a monitor, we should probably record a "UP" result
	if h.MonitorID != "" {
		m, err := s.db.GetMonitor(r.Context(), h.MonitorID)
		if err == nil && m != nil {
			result := &models.CheckResult{
				MonitorID: m.ID,
				Status:    models.StatusUp,
				Message:   "Heartbeat ping received",
				CheckedAt: now,
			}
			_ = s.db.InsertCheckResult(r.Context(), result)
		}
	}

	jsonOK(w, map[string]any{"message": "pong", "last_ping": now})
}
