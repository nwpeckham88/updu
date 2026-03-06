package api

import (
	"net/http"
	"strconv"
)

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	events, err := s.db.ListRecentEvents(r.Context(), limit)
	if err != nil {
		jsonError(w, "failed to list events", http.StatusInternalServerError)
		return
	}
	jsonOK(w, events)
}

func (s *Server) handleListMonitorEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	events, err := s.db.ListEventsByMonitor(r.Context(), id, limit)
	if err != nil {
		jsonError(w, "failed to get monitor events", http.StatusInternalServerError)
		return
	}
	jsonOK(w, events)
}
