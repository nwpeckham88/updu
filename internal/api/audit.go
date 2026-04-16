package api

import (
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/models"
)

func (s *Server) recordAudit(r *http.Request, action, resourceType, resourceID, summary string) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		return
	}

	actorType := "user"
	actorID := user.ID
	actorName := user.Username
	if token := auth.APITokenFromContext(r.Context()); token != nil {
		actorType = "api_token"
		actorID = token.ID
		actorName = user.Username + " via " + token.Name
	}

	entry := &models.AuditLog{
		ActorType:    actorType,
		ActorID:      actorID,
		ActorName:    actorName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Summary:      summary,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.db.CreateAuditLog(r.Context(), entry); err != nil {
		slog.Warn("failed to write audit log", "action", action, "resource_type", resourceType, "resource_id", resourceID, "error", err)
	}
}

func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := s.db.ListAuditLogs(r.Context(), models.AuditLogFilters{
		Limit:        limit,
		ActorID:      r.URL.Query().Get("actor_id"),
		Action:       r.URL.Query().Get("action"),
		ResourceType: r.URL.Query().Get("resource_type"),
		ResourceID:   r.URL.Query().Get("resource_id"),
	})
	if err != nil {
		jsonError(w, "failed to list audit logs", http.StatusInternalServerError)
		return
	}
	jsonOK(w, logs)
}

func summarizeStringMapKeys(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return "updated keys: " + strings.Join(keys, ", ")
}
