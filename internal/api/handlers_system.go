package api

import (
	"net/http"
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
