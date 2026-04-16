package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/models"
)

func (s *Server) handleListAPITokens(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	tokens, err := s.db.ListAPITokens(r.Context())
	if err != nil {
		jsonError(w, "failed to list api tokens", http.StatusInternalServerError)
		return
	}
	jsonOK(w, tokens)
}

func (s *Server) handleCreateAPIToken(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		Name  string               `json:"name"`
		Scope models.APITokenScope `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || !req.Scope.IsValid() {
		jsonError(w, "invalid api token request", http.StatusBadRequest)
		return
	}

	id, err := auth.GenerateID()
	if err != nil {
		jsonError(w, "failed to generate token id", http.StatusInternalServerError)
		return
	}
	secretID, err := auth.GenerateID()
	if err != nil {
		jsonError(w, "failed to generate token secret", http.StatusInternalServerError)
		return
	}
	secret := "updu_" + secretID
	hash := sha256.Sum256([]byte(secret))
	createdAt := time.Now().UTC()
	prefix := secret
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	created := &models.APITokenSecret{
		APIToken: models.APIToken{
			ID:        id,
			Name:      req.Name,
			Prefix:    prefix,
			Scope:     req.Scope,
			CreatedBy: user.ID,
			CreatedAt: createdAt,
		},
		Token: secret,
	}

	if err := s.db.CreateAPIToken(r.Context(), &created.APIToken, hex.EncodeToString(hash[:])); err != nil {
		jsonError(w, "failed to create api token", http.StatusInternalServerError)
		return
	}

	s.recordAudit(r, "api_token.create", "api_token", created.ID, "created API token "+req.Name)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, created)
}

func (s *Server) handleDeleteAPIToken(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if err := s.db.RevokeAPIToken(r.Context(), id, time.Now().UTC()); err != nil {
		jsonError(w, "failed to revoke api token", http.StatusInternalServerError)
		return
	}

	s.recordAudit(r, "api_token.revoke", "api_token", id, "revoked API token")
	jsonOK(w, map[string]any{"message": "api token revoked"})
}
