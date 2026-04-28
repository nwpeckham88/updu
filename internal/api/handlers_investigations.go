package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/realtime"
)

type monitorInvestigationRequest struct {
	Investigating *bool `json:"investigating"`
}

func (s *Server) handleSetMonitorInvestigation(w http.ResponseWriter, r *http.Request) {
	monitorID := r.PathValue("id")
	monitor, err := s.db.GetMonitor(r.Context(), monitorID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if monitor == nil {
		jsonError(w, "monitor not found", http.StatusNotFound)
		return
	}

	var req monitorInvestigationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	active := req.Investigating == nil || *req.Investigating
	if req.Investigating == nil {
		active = s.currentInvestigation(monitorID) == nil
	}

	user := auth.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	investigation := &models.MonitorInvestigation{
		MonitorID: monitorID,
		Active:    active,
		UpdatedBy: user.ID,
		UpdatedAt: time.Now(),
	}

	s.investigationMu.Lock()
	if active {
		s.investigations[monitorID] = investigation
	} else {
		delete(s.investigations, monitorID)
	}
	s.investigationMu.Unlock()

	s.sse.Broadcast(realtime.Event{
		Type:      "monitor:investigation",
		Data:      investigation,
		AdminOnly: true,
	})

	jsonOK(w, investigation)
}

func (s *Server) currentInvestigation(monitorID string) *models.MonitorInvestigation {
	s.investigationMu.RLock()
	investigation, ok := s.investigations[monitorID]
	s.investigationMu.RUnlock()
	if !ok {
		return nil
	}

	copy := *investigation
	return &copy
}

func (s *Server) attachInvestigationToMonitor(monitor *models.Monitor) {
	monitor.Investigation = s.currentInvestigation(monitor.ID)
}

func (s *Server) attachInvestigationToSummary(summary map[string]any) {
	monitorID, ok := summary["id"].(string)
	if !ok || monitorID == "" {
		delete(summary, "investigation")
		return
	}

	investigation := s.currentInvestigation(monitorID)
	if investigation == nil {
		delete(summary, "investigation")
		return
	}
	summary["investigation"] = investigation
}
