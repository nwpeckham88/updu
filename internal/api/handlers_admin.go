package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/models"
)

func (s *Server) handleListIncidents(w http.ResponseWriter, r *http.Request) {
	incidents, err := s.db.ListIncidents(r.Context())
	if err != nil {
		jsonError(w, "failed to list incidents", http.StatusInternalServerError)
		return
	}
	jsonOK(w, incidents)
}

func (s *Server) handleCreateIncident(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	var i models.Incident
	if err := json.NewDecoder(r.Body).Decode(&i); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	id, _ := auth.GenerateID()
	i.ID = id
	i.CreatedBy = user.ID
	i.StartedAt = time.Now()

	if err := s.db.CreateIncident(r.Context(), &i); err != nil {
		jsonError(w, "failed to create incident", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, i)
}

func (s *Server) handleListMaintenanceWindows(w http.ResponseWriter, r *http.Request) {
	windows, err := s.db.ListMaintenanceWindows(r.Context())
	if err != nil {
		jsonError(w, "failed to list maintenance windows", http.StatusInternalServerError)
		return
	}
	jsonOK(w, windows)
}

func (s *Server) handleCreateMaintenanceWindow(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	var mw models.MaintenanceWindow
	if err := json.NewDecoder(r.Body).Decode(&mw); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	id, _ := auth.GenerateID()
	mw.ID = id
	mw.CreatedBy = user.ID

	if err := s.db.CreateMaintenanceWindow(r.Context(), &mw); err != nil {
		jsonError(w, "failed to create maintenance window", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, mw)
}

func (s *Server) handleGetIncident(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	i, err := s.db.GetIncident(r.Context(), id)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if i == nil {
		jsonError(w, "incident not found", http.StatusNotFound)
		return
	}
	jsonOK(w, i)
}

func (s *Server) handleUpdateIncident(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	existing, err := s.db.GetIncident(r.Context(), id)
	if err != nil || existing == nil {
		jsonError(w, "incident not found", http.StatusNotFound)
		return
	}

	var update models.Incident
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	existing.Title = update.Title
	existing.Description = update.Description
	existing.Status = update.Status
	existing.Severity = update.Severity
	existing.MonitorIDs = update.MonitorIDs
	existing.ResolvedAt = update.ResolvedAt

	if err := s.db.UpdateIncident(r.Context(), existing); err != nil {
		jsonError(w, "failed to update incident", http.StatusInternalServerError)
		return
	}
	jsonOK(w, existing)
}

func (s *Server) handleDeleteIncident(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if err := s.db.DeleteIncident(r.Context(), id); err != nil {
		jsonError(w, "failed to delete incident", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"message": "deleted"})
}

func (s *Server) handleGetMaintenanceWindow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	mw, err := s.db.GetMaintenanceWindow(r.Context(), id)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if mw == nil {
		jsonError(w, "maintenance window not found", http.StatusNotFound)
		return
	}
	jsonOK(w, mw)
}

func (s *Server) handleUpdateMaintenanceWindow(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	existing, err := s.db.GetMaintenanceWindow(r.Context(), id)
	if err != nil || existing == nil {
		jsonError(w, "maintenance window not found", http.StatusNotFound)
		return
	}

	var update models.MaintenanceWindow
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	existing.Title = update.Title
	existing.MonitorIDs = update.MonitorIDs
	existing.StartsAt = update.StartsAt
	existing.EndsAt = update.EndsAt
	existing.Recurring = update.Recurring

	if err := s.db.UpdateMaintenanceWindow(r.Context(), existing); err != nil {
		jsonError(w, "failed to update maintenance window", http.StatusInternalServerError)
		return
	}
	jsonOK(w, existing)
}

func (s *Server) handleDeleteMaintenanceWindow(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if err := s.db.DeleteMaintenanceWindow(r.Context(), id); err != nil {
		jsonError(w, "failed to delete maintenance window", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"message": "deleted"})
}

// --- Groups ---

func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.db.ListGroups(r.Context())
	if err != nil {
		jsonError(w, "failed to list groups", http.StatusInternalServerError)
		return
	}
	jsonOK(w, groups)
}

func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	oldName := r.PathValue("name")
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		jsonError(w, "invalid request body or missing name", http.StatusBadRequest)
		return
	}

	if err := s.db.RenameGroup(r.Context(), oldName, req.Name); err != nil {
		jsonError(w, "failed to update group", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{"message": "updated"})
}

func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	name := r.PathValue("name")
	if err := s.db.DeleteGroup(r.Context(), name); err != nil {
		jsonError(w, "failed to delete group", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{"message": "deleted (monitors unassigned)"})
}

// --- User Management ---

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	users, err := s.db.ListUsers(r.Context())
	if err != nil {
		jsonError(w, "failed to list users", http.StatusInternalServerError)
		return
	}
	jsonOK(w, users)
}

func (s *Server) handleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	var req struct {
		Role models.UserRole `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Role != models.RoleAdmin && req.Role != models.RoleViewer) {
		jsonError(w, "invalid role", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateUserRole(r.Context(), id, req.Role); err != nil {
		jsonError(w, "failed to update role", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{"message": "role updated"})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if id == user.ID {
		jsonError(w, "cannot delete yourself", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteUser(r.Context(), id); err != nil {
		jsonError(w, "failed to delete user", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{"message": "user deleted"})
}

// --- Settings ---

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	settings, err := s.db.ListSettings(r.Context())
	if err != nil {
		jsonError(w, "failed to get settings", http.StatusInternalServerError)
		return
	}
	jsonOK(w, settings)
}

// allowedSettingsKeys is the set of settings keys that may be modified via the API.
var allowedSettingsKeys = map[string]bool{
	"site_name":         true,
	"site_description":  true,
	"base_url":          true,
	"custom_css":        true,
	"enable_custom_css": true,
	"logo_url":          true,
	"favicon_url":       true,
	"theme":             true,
	"timezone":          true,
	"date_format":       true,
	"enable_public":     true,
	"maintenance_mode":  true,
	"notify_on_down":    true,
	"notify_on_up":      true,
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	for k, v := range req {
		if !allowedSettingsKeys[k] {
			jsonError(w, "unknown setting key: "+k, http.StatusBadRequest)
			return
		}

		// Sanitize custom CSS server-side
		if k == "custom_css" {
			v = sanitizeCSS(v)
		}

		if err := s.db.SetSetting(r.Context(), k, v); err != nil {
			jsonError(w, "failed to update setting", http.StatusInternalServerError)
			return
		}
	}

	jsonOK(w, map[string]any{"message": "settings updated"})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 8 {
		jsonError(w, "new password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Verify current password
	if !auth.CheckPassword(user.Password, req.CurrentPassword) {
		jsonError(w, "current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Hash new password
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.db.UpdateUserPassword(r.Context(), user.ID, hash); err != nil {
		jsonError(w, "failed to update password", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{"message": "password updated"})
}
