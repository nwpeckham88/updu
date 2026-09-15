package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/gitops"
	"github.com/updu/updu/internal/models"
)

var (
	gitopsMu         sync.Mutex
	lastGitOpsSync   *time.Time
	lastGitOpsStatus string
	lastGitOpsError  string
)

// POST /api/v1/gitops/webhook
func (s *Server) handleGitOpsWebhook(w http.ResponseWriter, r *http.Request) {
	if s.config == nil || !s.config.GitOps.Enabled {
		jsonError(w, "gitops is not enabled", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20)) // 2MB limit
	if err != nil {
		jsonError(w, "failed to read webhook body", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("X-Forgejo-Signature")
	if sig == "" {
		sig = r.Header.Get("X-Gitea-Signature")
	}
	tok := r.Header.Get("X-Forgejo-Token")
	if tok == "" {
		tok = r.Header.Get("X-Gitea-Token")
	}

	secret := s.config.GitOps.WebhookSecret
	if secret != "" && !gitops.VerifySignature(body, secret, sig, tok) {
		slog.Warn("gitops: invalid webhook signature from forgejo", "remote", r.RemoteAddr)
		jsonError(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Check branch if payload contains ref
	var payload struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Ref != "" {
		targetBranch := s.config.GitOps.Branch
		if targetBranch == "" {
			targetBranch = "main"
		}
		expectedRef := "refs/heads/" + targetBranch
		if payload.Ref != expectedRef {
			slog.Info("gitops: ignoring push to non-target branch", "ref", payload.Ref, "expected", expectedRef)
			jsonOK(w, map[string]any{"message": "ignored push to non-target branch", "ref": payload.Ref})
			return
		}
	}

	res, err := s.triggerGitOpsSync(r.Context())
	if err != nil {
		slog.Error("gitops: webhook sync failed", "error", err)
		jsonError(w, "sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.recordAudit(r, "gitops.webhook_sync", "gitops", "webhook", "synced configuration from forgejo webhook")
	jsonOK(w, map[string]any{
		"message": "gitops sync successful",
		"result":  res,
	})
}

// POST /api/v1/gitops/sync
func (s *Server) handleGitOpsManualSync(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	if s.config == nil || !s.config.GitOps.Enabled {
		jsonError(w, "gitops is not enabled", http.StatusBadRequest)
		return
	}

	res, err := s.triggerGitOpsSync(r.Context())
	if err != nil {
		jsonError(w, "gitops sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.recordAudit(r, "gitops.manual_sync", "gitops", user.ID, "triggered manual gitops sync")
	jsonOK(w, map[string]any{
		"message": "gitops sync successful",
		"result":  res,
	})
}

// GET /api/v1/gitops/status
func (s *Server) handleGitOpsStatus(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	cfg := s.config.GitOps
	gitopsMu.Lock()
	lastSync := lastGitOpsSync
	lastStat := lastGitOpsStatus
	lastErr := lastGitOpsError
	gitopsMu.Unlock()

	var lastSyncStr *string
	if lastSync != nil {
		str := lastSync.UTC().Format(time.RFC3339)
		lastSyncStr = &str
	}

	jsonOK(w, map[string]any{
		"enabled":     cfg.Enabled,
		"provider":    cfg.Provider,
		"server_url":  cfg.ServerURL,
		"repository":  cfg.Repository,
		"branch":      cfg.Branch,
		"path":        cfg.Path,
		"auto_reload": cfg.AutoReload,
		"last_sync":   lastSyncStr,
		"last_status": lastStat,
		"last_error":  lastErr,
	})
}

func (s *Server) triggerGitOpsSync(ctx httpContext) (*gitops.SyncResult, error) {
	gitopsMu.Lock()
	defer gitopsMu.Unlock()

	rawYAML, err := gitops.FetchRawConfig(ctx, &s.config.GitOps)
	if err != nil {
		now := time.Now()
		lastGitOpsSync = &now
		lastGitOpsStatus = "error"
		lastGitOpsError = err.Error()
		return nil, err
	}

	res, err := gitops.SyncConfig(ctx, rawYAML, s.registry, s.db, s.scheduler)
	now := time.Now()
	lastGitOpsSync = &now
	if err != nil {
		lastGitOpsStatus = "error"
		lastGitOpsError = err.Error()
		return nil, err
	}

	lastGitOpsStatus = "success"
	lastGitOpsError = ""
	return res, nil
}

type httpContext = interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}
