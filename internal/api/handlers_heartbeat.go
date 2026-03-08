package api

import (
	"net/http"
	"time"

	"github.com/updu/updu/internal/models"
)

func (s *Server) handleHeartbeatPing(w http.ResponseWriter, r *http.Request) {
	// Try token from path first (new format: /heartbeat/{token})
	token := r.PathValue("token")
	var h *models.Heartbeat
	var err error

	if token != "" {
		h, err = s.db.GetHeartbeatByToken(r.Context(), token)
	} else {
		// Fallback to slug (old format: /api/v1/heartbeat/{slug})
		slug := r.PathValue("slug")
		if slug != "" {
			h, err = s.db.GetHeartbeat(r.Context(), slug)
			// For slug-based, we still need token verification (query or header)
			if h != nil {
				reqToken := r.URL.Query().Get("token")
				if reqToken == "" {
					authHeader := r.Header.Get("Authorization")
					if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
						reqToken = authHeader[7:]
					}
				}
				if reqToken != h.Token {
					jsonError(w, "unauthorized: invalid or missing token", http.StatusUnauthorized)
					return
				}
			}
		}
	}

	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if h == nil {
		jsonError(w, "heartbeat not found", http.StatusNotFound)
		return
	}

	// Check for "down" status in query params
	// e.g. /heartbeat/[token]?status=down or ?down=true
	isDown := r.URL.Query().Get("status") == "down" || r.URL.Query().Get("down") == "true"

	// Update last ping
	now := time.Now()
	h.LastPing = &now
	if err := s.db.UpsertHeartbeat(r.Context(), h); err != nil {
		jsonError(w, "failed to update heartbeat", http.StatusInternalServerError)
		return
	}

	// Record check result
	if h.MonitorID != "" {
		status := models.StatusUp
		message := "Heartbeat ping received"
		if isDown {
			status = models.StatusDown
			message = "Heartbeat reported failure (down flag set)"
		}

		result := &models.CheckResult{
			MonitorID: h.MonitorID,
			Status:    status,
			Message:   message,
			CheckedAt: now,
		}
		_ = s.db.InsertCheckResult(r.Context(), result)
	}

	jsonOK(w, map[string]any{"message": "pong", "last_ping": now, "status": h.LastPing})
}
