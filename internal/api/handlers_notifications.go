package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/models"
)

func (s *Server) handleListNotificationChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := s.db.ListNotificationChannels(r.Context())
	if err != nil {
		jsonError(w, "failed to list notification channels", http.StatusInternalServerError)
		return
	}
	jsonOK(w, channels)
}

func (s *Server) handleCreateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	var nc models.NotificationChannel
	if err := json.NewDecoder(r.Body).Decode(&nc); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	id, _ := auth.GenerateID()
	nc.ID = id

	if err := s.db.CreateNotificationChannel(r.Context(), &nc); err != nil {
		jsonError(w, "failed to create notification channel", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, nc)
}

func (s *Server) handleGetNotificationChannel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	nc, err := s.db.GetNotificationChannel(r.Context(), id)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if nc == nil {
		jsonError(w, "notification channel not found", http.StatusNotFound)
		return
	}
	jsonOK(w, nc)
}

func (s *Server) handleUpdateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	existing, err := s.db.GetNotificationChannel(r.Context(), id)
	if err != nil || existing == nil {
		jsonError(w, "notification channel not found", http.StatusNotFound)
		return
	}

	var update models.NotificationChannel
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	existing.Name = update.Name
	existing.Type = update.Type
	existing.Config = update.Config
	existing.Enabled = update.Enabled

	if err := s.db.UpdateNotificationChannel(r.Context(), existing); err != nil {
		jsonError(w, "failed to update notification channel", http.StatusInternalServerError)
		return
	}

	jsonOK(w, existing)
}

func (s *Server) handleDeleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if err := s.db.DeleteNotificationChannel(r.Context(), id); err != nil {
		jsonError(w, "failed to delete notification channel", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"message": "deleted"})
}

func (s *Server) handleTestNotificationChannel(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	nc, err := s.db.GetNotificationChannel(r.Context(), id)
	if err != nil || nc == nil {
		jsonError(w, "notification channel not found", http.StatusNotFound)
		return
	}

	// Create a dummy monitor and result for testing
	dummyMonitor := &models.Monitor{
		Name: "Test Monitor",
	}
	event := &models.Event{
		Status:    models.StatusDown,
		Message:   "This is a test notification from updu.",
		CreatedAt: time.Now(),
	}

	// Trigger notification
	s.notifier.Notify(r.Context(), dummyMonitor, event)

	jsonOK(w, map[string]any{"message": "test notification dispatched"})
}
