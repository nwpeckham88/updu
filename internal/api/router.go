package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/config"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/notifier"
	"github.com/updu/updu/internal/realtime"
	"github.com/updu/updu/internal/scheduler"
	"github.com/updu/updu/internal/storage"
	"github.com/updu/updu/internal/version"
)

// CSS sanitization patterns (compiled once)
var (
	importRe     = regexp.MustCompile(`(?i)@import\b`)
	jsURLRe      = regexp.MustCompile(`(?i)url\s*\(\s*["']?javascript:`)
	expressionRe = regexp.MustCompile(`(?i)expression\s*\(`)
	mozBindingRe = regexp.MustCompile(`(?i)-moz-binding\s*:`)
	behaviorRe   = regexp.MustCompile(`(?i)behavior\s*:`)
	dataURLRe    = regexp.MustCompile(`(?i)url\s*\(\s*["']?data:`)
	styleTagRe   = regexp.MustCompile(`(?i)</\s*style`)
)

// Server holds all API dependencies.
type Server struct {
	db        *storage.DB
	auth      *auth.Auth
	registry  *checker.Registry
	scheduler *scheduler.Scheduler
	notifier  *notifier.Notifier
	sse       *realtime.Hub
	config    *config.Config

	investigationMu sync.RWMutex
	investigations  map[string]*models.MonitorInvestigation

	// Login rate limiting
	loginAttempts map[string]*loginEntry
	loginMu       sync.Mutex
}

type loginEntry struct {
	count       int
	windowStart time.Time
}

// NewServer creates a new API server.
func NewServer(db *storage.DB, a *auth.Auth, reg *checker.Registry, sched *scheduler.Scheduler, n *notifier.Notifier, sse *realtime.Hub, cfg *config.Config) *Server {
	s := &Server{
		db:             db,
		auth:           a,
		registry:       reg,
		scheduler:      sched,
		notifier:       n,
		sse:            sse,
		config:         cfg,
		investigations: make(map[string]*models.MonitorInvestigation),
		loginAttempts:  make(map[string]*loginEntry),
	}

	// Periodically clean rate limiter entries
	go func() {
		for range time.Tick(5 * time.Minute) {
			s.loginMu.Lock()
			now := time.Now()
			for key, entry := range s.loginAttempts {
				if now.Sub(entry.windowStart) > time.Minute {
					delete(s.loginAttempts, key)
				}
			}
			s.loginMu.Unlock()
		}
	}()

	return s
}

// Router returns the configured HTTP mux.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Helper to wrap handlers with auth
	authed := func(handler http.HandlerFunc) http.Handler {
		return s.auth.Middleware(handler)
	}

	// Helper to wrap handlers with auth + admin role
	adminAuthed := func(handler http.HandlerFunc) http.Handler {
		return s.auth.Middleware(auth.AdminMiddleware(handler))
	}

	// Helper to require a browser/session-backed admin rather than a bearer token.
	adminSessionAuthed := func(handler http.HandlerFunc) http.Handler {
		return s.auth.Middleware(auth.AdminSessionMiddleware(handler))
	}

	// --- Public routes ---
	mux.HandleFunc("POST /api/v1/auth/login", maxBody(1<<20, s.handleLogin))
	mux.HandleFunc("POST /api/v1/auth/register", maxBody(1<<20, s.handleRegister))
	mux.HandleFunc("GET /api/v1/auth/setup", s.handleSetupCheck)
	mux.HandleFunc("GET /api/v1/auth/providers", s.handleAuthProviders)

	// Register OIDC routes (conditionally compiled via build tags)
	registerOIDCRoutes(mux, s)

	mux.HandleFunc("POST /api/v1/status-pages/{slug}/unlock", maxBody(1<<20, s.handleUnlockStatusPage))
	mux.HandleFunc("GET /api/v1/status-pages/{slug}", s.handleGetStatusPage)
	mux.HandleFunc("POST /api/v1/heartbeat/{slug}", maxBody(1<<20, s.handleHeartbeatPing))

	// New simplified heartbeat routes
	mux.HandleFunc("GET /heartbeat/{token}", s.handleHeartbeatPing)
	mux.HandleFunc("POST /heartbeat/{token}", maxBody(1<<20, s.handleHeartbeatPing))
	mux.HandleFunc("PUT /heartbeat/{token}", maxBody(1<<20, s.handleHeartbeatPing))

	mux.HandleFunc("GET /api/v1/system/health", s.handleHealth)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/openapi.json", s.handleOpenAPI)
	mux.HandleFunc("GET /api/v1/metrics", s.handlePrometheusMetrics)
	mux.HandleFunc("GET /api/v1/custom.css", s.handleCustomCSS)

	// --- SSE (authenticated) ---
	mux.Handle("GET /api/v1/events", authed(http.HandlerFunc(s.handleRealtimeEvents)))

	// --- Authenticated routes ---
	mux.Handle("POST /api/v1/auth/logout", authed(s.handleLogout))
	mux.Handle("GET /api/v1/auth/session", authed(s.handleSession))
	mux.Handle("PUT /api/v1/auth/password", authed(maxBody(1<<20, s.handleChangePassword)))

	mux.Handle("GET /api/v1/monitors", authed(s.handleListMonitors))
	mux.Handle("POST /api/v1/monitors", adminAuthed(maxBody(1<<20, s.handleCreateMonitor)))
	mux.Handle("POST /api/v1/monitors/test", adminAuthed(maxBody(1<<20, s.handleTestMonitor)))
	mux.Handle("GET /api/v1/monitors/{id}", authed(s.handleGetMonitor))
	mux.Handle("POST /api/v1/monitors/{id}/investigate", adminAuthed(maxBody(1<<20, s.handleSetMonitorInvestigation)))
	mux.Handle("PUT /api/v1/monitors/{id}", adminAuthed(maxBody(1<<20, s.handleUpdateMonitor)))
	mux.Handle("DELETE /api/v1/monitors/{id}", adminAuthed(s.handleDeleteMonitor))
	mux.Handle("GET /api/v1/monitors/{id}/checks", authed(s.handleGetMonitorChecks))
	mux.Handle("GET /api/v1/monitors/{id}/events", authed(s.handleListMonitorEvents))
	mux.Handle("GET /api/v1/monitors/{id}/uptime", authed(s.handleGetMonitorUptime))

	mux.Handle("GET /api/v1/dashboard", authed(s.handleDashboard))
	mux.Handle("GET /api/v1/stats", authed(s.handleGetStats))
	mux.Handle("GET /api/v1/events/history", authed(s.handleListEvents))

	// Status Pages (Admin for mutations, auth for reads)
	mux.Handle("GET /api/v1/status-pages", authed(s.handleListStatusPages))
	mux.Handle("POST /api/v1/status-pages", adminAuthed(maxBody(1<<20, s.handleCreateStatusPage)))
	mux.Handle("GET /api/v1/status-pages/{id}/detail", authed(s.handleGetStatusPageByID))
	mux.Handle("PUT /api/v1/status-pages/{id}", adminAuthed(maxBody(1<<20, s.handleUpdateStatusPage)))
	mux.Handle("DELETE /api/v1/status-pages/{id}", adminAuthed(s.handleDeleteStatusPage))

	// Notifications (Admin)
	mux.Handle("GET /api/v1/notifications", adminSessionAuthed(s.handleListNotificationChannels))
	mux.Handle("POST /api/v1/notifications", adminAuthed(maxBody(1<<20, s.handleCreateNotificationChannel)))
	mux.Handle("GET /api/v1/notifications/{id}", adminSessionAuthed(s.handleGetNotificationChannel))
	mux.Handle("PUT /api/v1/notifications/{id}", adminAuthed(maxBody(1<<20, s.handleUpdateNotificationChannel)))
	mux.Handle("DELETE /api/v1/notifications/{id}", adminAuthed(s.handleDeleteNotificationChannel))
	mux.Handle("POST /api/v1/notifications/{id}/test", adminAuthed(maxBody(1<<20, s.handleTestNotificationChannel)))

	// Incidents & Maintenance (Admin for mutations)
	mux.Handle("GET /api/v1/incidents", authed(s.handleListIncidents))
	mux.Handle("POST /api/v1/incidents", adminAuthed(maxBody(1<<20, s.handleCreateIncident)))
	mux.Handle("GET /api/v1/incidents/{id}", authed(s.handleGetIncident))
	mux.Handle("PUT /api/v1/incidents/{id}", adminAuthed(maxBody(1<<20, s.handleUpdateIncident)))
	mux.Handle("DELETE /api/v1/incidents/{id}", adminAuthed(s.handleDeleteIncident))

	mux.Handle("GET /api/v1/maintenance", authed(s.handleListMaintenanceWindows))
	mux.Handle("POST /api/v1/maintenance", adminAuthed(maxBody(1<<20, s.handleCreateMaintenanceWindow)))
	mux.Handle("GET /api/v1/maintenance/{id}", authed(s.handleGetMaintenanceWindow))
	mux.Handle("PUT /api/v1/maintenance/{id}", adminAuthed(maxBody(1<<20, s.handleUpdateMaintenanceWindow)))
	mux.Handle("DELETE /api/v1/maintenance/{id}", adminAuthed(s.handleDeleteMaintenanceWindow))

	// Groups (Admin for mutations)
	mux.Handle("GET /api/v1/groups", authed(s.handleListGroups))
	mux.Handle("PUT /api/v1/groups/{name}", adminAuthed(s.handleUpdateGroup))
	mux.Handle("DELETE /api/v1/groups/{name}", adminAuthed(s.handleDeleteGroup))

	// User Management (Admin)
	mux.Handle("GET /api/v1/admin/users", adminSessionAuthed(s.handleListUsers))
	mux.Handle("PUT /api/v1/admin/users/{id}/role", adminSessionAuthed(s.handleUpdateUserRole))
	mux.Handle("DELETE /api/v1/admin/users/{id}", adminSessionAuthed(s.handleDeleteUser))
	mux.Handle("GET /api/v1/admin/api-tokens", adminSessionAuthed(s.handleListAPITokens))
	mux.Handle("POST /api/v1/admin/api-tokens", adminSessionAuthed(maxBody(1<<20, s.handleCreateAPIToken)))
	mux.Handle("DELETE /api/v1/admin/api-tokens/{id}", adminSessionAuthed(s.handleDeleteAPIToken))
	mux.Handle("GET /api/v1/audit-logs", adminSessionAuthed(s.handleListAuditLogs))

	// Settings (Admin)
	mux.Handle("GET /api/v1/settings", adminSessionAuthed(s.handleGetSettings))
	mux.Handle("POST /api/v1/settings", adminSessionAuthed(maxBody(1<<20, s.handleUpdateSettings)))

	// System (Admin)
	mux.Handle("GET /api/v1/system/metrics", adminSessionAuthed(s.handleGetMetrics))
	mux.Handle("GET /api/v1/system/backup", adminSessionAuthed(s.handleExportConfig))
	mux.Handle("GET /api/v1/system/export/yaml", adminSessionAuthed(s.handleExportYAML))
	mux.Handle("POST /api/v1/system/backup", adminSessionAuthed(maxBody(10<<20, s.handleImportConfig)))
	mux.Handle("GET /api/v1/system/version", adminSessionAuthed(s.handleCheckUpdate))
	mux.Handle("POST /api/v1/system/update", adminSessionAuthed(s.handleApplyUpdate))

	// Wrap with CORS and logging
	return withMiddleware(mux)
}

// realClientIP returns the best-guess client IP.
// Forwarded headers are only honored when the direct peer is an explicitly
// trusted proxy; otherwise RemoteAddr is used.
func realClientIP(cfg *config.Config, r *http.Request) string {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || remoteIP == "" {
		remoteIP = r.RemoteAddr
	}

	if cfg == nil || !cfg.IsTrustedProxy(r.RemoteAddr) {
		return remoteIP
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return remoteIP
	}

	hops := strings.Split(xff, ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		parsed := net.ParseIP(hop)
		if parsed == nil {
			continue
		}
		if cfg.IsTrustedProxyIP(parsed) {
			continue
		}
		return hop
	}

	return remoteIP
}

// checkLoginRateLimit increments the counter for key and returns the current
// count. Returns true (rate-limited) when count exceeds limit.
func (s *Server) checkLoginRateLimit(key string) (count int, limited bool) {
	s.loginMu.Lock()
	entry, ok := s.loginAttempts[key]
	now := time.Now()
	if !ok || now.Sub(entry.windowStart) > time.Minute {
		entry = &loginEntry{windowStart: now}
		s.loginAttempts[key] = entry
	}
	entry.count++
	count = entry.count
	s.loginMu.Unlock()
	return count, count > 5
}

// --- Auth Handlers ---

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Rate limit by client IP, honoring forwarded headers only for trusted proxies.
	ip := realClientIP(s.config, r)
	if _, limited := s.checkLoginRateLimit("ip:" + ip); limited {
		slog.Warn("login rate limited", "ip", ip)
		jsonError(w, "too many login attempts, try again in a minute", http.StatusTooManyRequests)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Also rate-limit by username to prevent distributed password-spraying.
	if req.Username != "" {
		if _, limited := s.checkLoginRateLimit("u:" + strings.ToLower(req.Username)); limited {
			slog.Warn("login rate limited", "username", req.Username)
			jsonError(w, "too many login attempts, try again in a minute", http.StatusTooManyRequests)
			return
		}
	}

	session, err := s.auth.Login(r.Context(), req.Username, req.Password, r.UserAgent(), ip)
	if err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	s.auth.SetSessionCookie(w, session.ID, session.ExpiresAt)
	jsonOK(w, map[string]any{"message": "logged in"})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	// Rate limit registration using the same IP + username limiter.
	ip := realClientIP(s.config, r)
	if _, limited := s.checkLoginRateLimit("ip:" + ip); limited {
		jsonError(w, "too many attempts, try again in a minute", http.StatusTooManyRequests)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Username) < 3 {
		jsonError(w, "username must be at least 3 characters", http.StatusBadRequest)
		return
	}
	if err := auth.ValidatePassword(req.Password, s.config.PasswordPolicy); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Only allow registration if no users exist (first user setup)
	// or if current user is admin
	count, err := s.db.CountUsers(r.Context())
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		var user *models.User
		cookie, err := r.Cookie("updu_session")
		if err == nil {
			session, _ := s.db.GetSession(r.Context(), cookie.Value)
			if session != nil {
				user, _ = s.db.GetUserByID(r.Context(), session.UserID)
			}
		}

		if user == nil || user.Role != models.RoleAdmin {
			jsonError(w, "registration disabled", http.StatusForbidden)
			return
		}
	}

	newUser, err := s.auth.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		jsonError(w, "registration failed", http.StatusBadRequest)
		return
	}

	jsonOK(w, map[string]any{
		"id":       newUser.ID,
		"username": newUser.Username,
		"role":     newUser.Role,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("updu_session")
	if err == nil {
		_ = s.auth.Logout(r.Context(), cookie.Value)
	}
	auth.ClearSessionCookie(w)
	jsonOK(w, map[string]any{"message": "logged out"})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	jsonOK(w, map[string]any{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

// --- Monitor Handlers ---

func (s *Server) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.db.ListMonitors(r.Context())
	if err != nil {
		jsonError(w, "failed to list monitors", http.StatusInternalServerError)
		return
	}

	// Redact sensitive config for non-admin users
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		redacted := make([]*models.Monitor, len(monitors))
		for i, m := range monitors {
			r := models.RedactMonitor(m)
			redacted[i] = &r
		}
		jsonOK(w, redacted)
		return
	}

	for _, m := range monitors {
		s.attachInvestigationToMonitor(m)
	}
	jsonOK(w, monitors)
}

func (s *Server) handleCreateMonitor(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var m models.Monitor
	if err := json.Unmarshal(body, &m); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	var requestDefaults struct {
		Enabled *bool `json:"enabled"`
	}
	_ = json.Unmarshal(body, &requestDefaults)

	// Validate checker type
	c := s.registry.Get(m.Type)
	if c == nil {
		jsonError(w, "unknown monitor type: "+m.Type, http.StatusBadRequest)
		return
	}
	if err := c.Validate(m.Config); err != nil {
		jsonError(w, "invalid config: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Set defaults
	id, err := auth.GenerateID()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	m.ID = id
	m.CreatedBy = user.ID
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	if m.IntervalS == 0 {
		m.IntervalS = 60
	}
	if m.IntervalS < 10 {
		jsonError(w, "interval must be at least 10 seconds", http.StatusBadRequest)
		return
	}
	if m.TimeoutS == 0 {
		m.TimeoutS = 10
	}
	if m.Retries == 0 {
		m.Retries = 3
	}
	m.Enabled = false
	if requestDefaults.Enabled != nil {
		m.Enabled = *requestDefaults.Enabled
	}

	if err := s.db.CreateMonitor(r.Context(), &m); err != nil {
		jsonError(w, "failed to create monitor", http.StatusInternalServerError)
		return
	}

	// For "push" monitors, automatically create a heartbeat record
	if m.Type == "push" {
		var config models.PushMonitorConfig
		_ = json.Unmarshal(m.Config, &config)
		if config.Token == "" {
			token, err := auth.GenerateID()
			if err != nil {
				jsonError(w, "internal error", http.StatusInternalServerError)
				return
			}
			config.Token = token
			m.Config, _ = json.Marshal(config)
			_ = s.db.UpdateMonitor(r.Context(), &m)
		}

		h := &models.Heartbeat{
			Slug:      m.ID, // Use monitor ID as the default slug
			MonitorID: m.ID,
			Token:     config.Token,
			ExpectedS: m.IntervalS,
			GraceS:    config.EffectiveGraceSeconds(m.IntervalS),
		}
		if err := s.db.UpsertHeartbeat(r.Context(), h); err != nil {
			slog.Warn("failed to sync push heartbeat after create", "monitor_id", m.ID, "error", err)
		}
	}

	// Add to scheduler (use background context — request context dies after response)
	s.scheduler.AddMonitor(context.Background(), &m)
	s.recordAudit(r, "monitor.create", "monitor", m.ID, "created monitor "+m.Name)

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, m)
}

func (s *Server) handleTestMonitor(w http.ResponseWriter, r *http.Request) {
	var m models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	c := s.registry.Get(m.Type)
	if c == nil {
		jsonError(w, "unknown monitor type: "+m.Type, http.StatusBadRequest)
		return
	}
	if err := c.Validate(m.Config); err != nil {
		jsonError(w, "invalid config: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Set defaults for the test
	if m.TimeoutS == 0 {
		m.TimeoutS = 10
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	result, err := c.Check(ctx, &m)
	if err != nil {
		jsonOK(w, map[string]any{
			"status":  "down",
			"message": err.Error(),
		})
		return
	}

	jsonOK(w, map[string]any{
		"status":      result.Status,
		"latency_ms":  result.LatencyMs,
		"status_code": result.StatusCode,
		"message":     result.Message,
	})
}

func (s *Server) handleGetMonitor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m, err := s.db.GetMonitor(r.Context(), id)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if m == nil {
		jsonError(w, "monitor not found", http.StatusNotFound)
		return
	}

	// Get latest check
	latest, _ := s.db.GetLatestCheck(r.Context(), id)
	if latest != nil {
		m.Status = latest.Status
		m.LastCheck = &latest.CheckedAt
		m.LastLatency = latest.LatencyMs
	} else {
		m.Status = models.StatusPending
	}
	// Redact sensitive config for non-admin users
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		redacted := models.RedactMonitor(m)
		jsonOK(w, &redacted)
		return
	}

	s.attachInvestigationToMonitor(m)
	jsonOK(w, m)
}

func (s *Server) handleUpdateMonitor(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	existing, err := s.db.GetMonitor(r.Context(), id)
	if err != nil || existing == nil {
		jsonError(w, "monitor not found", http.StatusNotFound)
		return
	}

	var update models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Merge fields
	existing.Name = update.Name
	existing.Type = update.Type
	existing.Config = update.Config
	existing.Groups = update.Groups
	existing.Tags = update.Tags
	existing.IntervalS = update.IntervalS
	existing.TimeoutS = update.TimeoutS
	existing.Retries = update.Retries
	existing.Enabled = update.Enabled
	existing.ParentID = update.ParentID

	c := s.registry.Get(existing.Type)
	if c == nil {
		jsonError(w, "unknown monitor type: "+existing.Type, http.StatusBadRequest)
		return
	}
	if err := c.Validate(existing.Config); err != nil {
		jsonError(w, "invalid config: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateMonitor(r.Context(), existing); err != nil {
		jsonError(w, "failed to update monitor", http.StatusInternalServerError)
		return
	}

	// For "push" monitors, update the heartbeat record
	if existing.Type == "push" {
		var config models.PushMonitorConfig
		_ = json.Unmarshal(existing.Config, &config)
		h := &models.Heartbeat{
			Slug:      existing.ID,
			MonitorID: existing.ID,
			Token:     config.Token,
			ExpectedS: existing.IntervalS,
			GraceS:    config.EffectiveGraceSeconds(existing.IntervalS),
		}
		if err := s.db.UpsertHeartbeat(r.Context(), h); err != nil {
			slog.Warn("failed to sync push heartbeat after update", "monitor_id", existing.ID, "error", err)
		}
	}

	s.scheduler.ReloadMonitor(context.Background(), existing)
	s.recordAudit(r, "monitor.update", "monitor", existing.ID, "updated monitor "+existing.Name)
	jsonOK(w, existing)
}

func (s *Server) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	s.scheduler.RemoveMonitor(id)
	if err := s.db.DeleteMonitor(r.Context(), id); err != nil {
		jsonError(w, "failed to delete monitor", http.StatusInternalServerError)
		return
	}
	s.recordAudit(r, "monitor.delete", "monitor", id, "deleted monitor")
	jsonOK(w, map[string]any{"message": "deleted"})
}

func (s *Server) handleGetMonitorChecks(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	checks, err := s.db.GetRecentChecks(r.Context(), id, 100)
	if err != nil {
		jsonError(w, "failed to get checks", http.StatusInternalServerError)
		return
	}
	jsonOK(w, checks)
}

func (s *Server) handleGetMonitorUptime(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	uptime24h, _ := s.db.GetUptimePercent(r.Context(), id, time.Now().Add(-24*time.Hour))
	uptime7d, _ := s.db.GetUptimePercent(r.Context(), id, time.Now().Add(-7*24*time.Hour))
	uptime30d, _ := s.db.GetUptimePercent(r.Context(), id, time.Now().Add(-30*24*time.Hour))

	jsonOK(w, map[string]any{
		"24h": uptime24h,
		"7d":  uptime7d,
		"30d": uptime30d,
	})
}

// --- Dashboard ---

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	summaries, err := s.db.GetMonitorsSummary(ctx)
	if err != nil {
		jsonError(w, "failed to get dashboard", http.StatusInternalServerError)
		return
	}

	// Enrich each monitor with recent checks and 24h uptime
	user := auth.UserFromContext(r.Context())
	includeInvestigation := user != nil && user.Role == models.RoleAdmin
	since24h := time.Now().Add(-24 * time.Hour)
	for _, sm := range summaries {
		monID, _ := sm["id"].(string)
		if monID == "" {
			continue
		}
		if includeInvestigation {
			s.attachInvestigationToSummary(sm)
		}

		// Last 40 checks for heartbeat bar
		rows, err := s.db.QueryContext(ctx,
			`SELECT status, latency_ms, checked_at FROM check_results
			 WHERE monitor_id = ? ORDER BY checked_at DESC LIMIT 40`, monID)
		if err == nil {
			var checks []map[string]any
			for rows.Next() {
				var st string
				var lat sql.NullInt64
				var at time.Time
				if rows.Scan(&st, &lat, &at) == nil {
					c := map[string]any{"status": st, "checked_at": at}
					if lat.Valid {
						c["latency_ms"] = lat.Int64
					}
					checks = append(checks, c)
				}
			}
			rows.Close()
			sm["recent_checks"] = checks
		}

		// 24h uptime
		var total, up int
		if err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END), 0)
			 FROM check_results WHERE monitor_id = ? AND checked_at >= ?`,
			monID, since24h).Scan(&total, &up); err == nil && total > 0 {
			sm["uptime_24h"] = math.Round(float64(up)/float64(total)*10000) / 100
		} else {
			sm["uptime_24h"] = nil
		}
	}

	jsonOK(w, map[string]any{
		"monitors":    summaries,
		"sse_clients": s.sse.ClientCount(),
	})
}

// --- Status Pages ---

type statusPageUpsertRequest struct {
	Name          string                   `json:"name"`
	Slug          string                   `json:"slug"`
	Description   string                   `json:"description"`
	Groups        []models.StatusPageGroup `json:"groups"`
	IsPublic      bool                     `json:"is_public"`
	Password      string                   `json:"password"`
	ClearPassword bool                     `json:"clear_password"`
}

type statusPageUnlockRequest struct {
	Password string `json:"password"`
}

func (s *Server) handleGetStatusPage(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	sp, err := s.db.GetStatusPageBySlug(r.Context(), slug)
	if err != nil || sp == nil {
		jsonError(w, "status page not found", http.StatusNotFound)
		return
	}
	if sp.Password != "" || !sp.IsPublic {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Add("Vary", "Cookie")
	}

	if sp.Password != "" {
		if !s.hasAdminSession(r) && !s.hasStatusPageAccess(r, sp) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]any{
				"error":             "password required",
				"password_required": true,
			})
			return
		}
	} else if !sp.IsPublic && !s.hasAuthenticatedSession(r) {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	// Get monitor summaries for this status page
	summaries, _ := s.db.GetMonitorsSummary(r.Context())

	// Filter monitors to only include those in the groups or monitor_ids assigned to this status page
	var filtered []map[string]any
	for _, sm := range summaries {
		monitorGroups, _ := sm["groups"].([]string)
		idStr, _ := sm["id"].(string)

		matched := false
		for _, g := range sp.Groups {
			// Check if monitor is in the group
			for _, mg := range monitorGroups {
				if g.Name != "" && g.Name == mg {
					matched = true
					break
				}
			}
			if matched {
				break
			}

			// Check if monitor is specifically listed
			for _, mid := range g.MonitorIDs {
				if mid == idStr {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}

		if matched {
			filtered = append(filtered, sm)
		}
	}

	jsonOK(w, map[string]any{
		"page":     sp,
		"monitors": filtered,
	})
}

func (s *Server) handleUnlockStatusPage(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	sp, err := s.db.GetStatusPageBySlug(r.Context(), slug)
	if err != nil || sp == nil {
		jsonError(w, "status page not found", http.StatusNotFound)
		return
	}
	if sp.Password == "" {
		jsonError(w, "status page is not password protected", http.StatusBadRequest)
		return
	}

	var req statusPageUnlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		jsonError(w, "password is required", http.StatusBadRequest)
		return
	}
	clientIP := realClientIP(s.config, r)
	if _, limited := s.checkLoginRateLimit("status-page:" + slug + ":" + clientIP); limited {
		jsonError(w, "too many unlock attempts, try again in a minute", http.StatusTooManyRequests)
		return
	}
	if !auth.CheckPassword(sp.Password, password) {
		jsonError(w, "invalid password", http.StatusForbidden)
		return
	}

	s.setStatusPageAccessCookie(w, sp)
	jsonOK(w, map[string]any{"message": "unlocked"})
}

func (s *Server) handleListStatusPages(w http.ResponseWriter, r *http.Request) {
	pages, err := s.db.ListStatusPages(r.Context())
	if err != nil {
		jsonError(w, "failed to list status pages", http.StatusInternalServerError)
		return
	}
	jsonOK(w, pages)
}

func (s *Server) handleGetStatusPageByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sp, err := s.db.GetStatusPageByID(r.Context(), id)
	if err != nil || sp == nil {
		jsonError(w, "status page not found", http.StatusNotFound)
		return
	}
	jsonOK(w, sp)
}

func (s *Server) handleCreateStatusPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	var req statusPageUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Password != "" && req.ClearPassword {
		jsonError(w, "password and clear_password cannot be combined", http.StatusBadRequest)
		return
	}
	password := strings.TrimSpace(req.Password)
	if req.Password != "" && password == "" {
		jsonError(w, "password cannot be blank", http.StatusBadRequest)
		return
	}
	if password != "" {
		if err := auth.ValidatePassword(password, s.config.PasswordPolicy); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if password != "" && !req.IsPublic {
		jsonError(w, "password-protected status pages must remain public", http.StatusBadRequest)
		return
	}

	id, err := auth.GenerateID()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if req.Slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}

	sp := &models.StatusPage{
		ID:          id,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Groups:      req.Groups,
		IsPublic:    req.IsPublic,
	}

	// Hash status page password if provided
	if password != "" {
		hash, err := auth.HashPassword(password)
		if err != nil {
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		sp.Password = hash
	}
	sp.PasswordProtected = sp.Password != ""

	if err := s.db.CreateStatusPage(r.Context(), sp); err != nil {
		jsonError(w, "failed to create status page", http.StatusInternalServerError)
		return
	}
	s.recordAudit(r, "status_page.create", "status_page", sp.ID, "created status page "+sp.Name)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, sp)
}

func (s *Server) handleUpdateStatusPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	existing, err := s.db.GetStatusPageByID(r.Context(), id)
	if err != nil || existing == nil {
		jsonError(w, "status page not found", http.StatusNotFound)
		return
	}

	var update statusPageUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if update.Password != "" && update.ClearPassword {
		jsonError(w, "password and clear_password cannot be combined", http.StatusBadRequest)
		return
	}
	password := strings.TrimSpace(update.Password)
	if update.Password != "" && password == "" {
		jsonError(w, "password cannot be blank", http.StatusBadRequest)
		return
	}
	if password != "" {
		if err := auth.ValidatePassword(password, s.config.PasswordPolicy); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if update.Name != "" {
		existing.Name = update.Name
	}
	if update.Slug != "" {
		existing.Slug = update.Slug
	}
	existing.Description = update.Description
	existing.Groups = update.Groups
	existing.IsPublic = update.IsPublic
	if existing.Slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}
	if password != "" {
		hash, err := auth.HashPassword(password)
		if err != nil {
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		existing.Password = hash
	} else if update.ClearPassword {
		existing.Password = ""
	}
	if existing.Password != "" && !existing.IsPublic {
		jsonError(w, "password-protected status pages must remain public", http.StatusBadRequest)
		return
	}
	existing.PasswordProtected = existing.Password != ""

	if err := s.db.UpdateStatusPage(r.Context(), existing); err != nil {
		jsonError(w, "failed to update status page", http.StatusInternalServerError)
		return
	}
	s.recordAudit(r, "status_page.update", "status_page", existing.ID, "updated status page "+existing.Name)
	jsonOK(w, existing)
}

func (s *Server) handleDeleteStatusPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if err := s.db.DeleteStatusPage(r.Context(), id); err != nil {
		jsonError(w, "failed to delete status page", http.StatusInternalServerError)
		return
	}
	s.recordAudit(r, "status_page.delete", "status_page", id, "deleted status page")
	jsonOK(w, map[string]any{"message": "deleted"})
}

// --- System ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Check database connectivity
	dbOK := true
	var dbErr string
	if err := s.db.PingContext(ctx); err != nil {
		dbOK = false
		dbErr = err.Error()
		slog.Error("health check: database ping failed", "error", err)
	}

	schedulerMonitors := s.scheduler.MonitorCount()
	sseClients := s.sse.ClientCount()

	status := "ok"
	httpStatus := http.StatusOK
	if !dbOK {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	resp := map[string]any{
		"status":  status,
		"version": version.Version,
		"components": map[string]any{
			"database":  map[string]any{"ok": dbOK, "error": dbErr},
			"scheduler": map[string]any{"monitors": schedulerMonitors},
			"sse":       map[string]any{"clients": sseClients},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(resp)
}

// handlePrometheusMetrics exposes system and monitor metrics in Prometheus
// text exposition format (no external dependency).
// If UPDU_METRICS_TOKEN (or metrics_token in config) is set, requests must
// supply a matching "Authorization: Bearer <token>" header — compatible with
// Prometheus scrape_config bearer_token.
func (s *Server) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	if token := s.config.MetricsToken; token != "" {
		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix || auth[len(prefix):] != token {
			w.Header().Set("WWW-Authenticate", `Bearer realm="updu metrics"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	ctx := r.Context()

	metrics, err := s.db.GetSystemMetrics(ctx)
	if err != nil {
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Application info
	fmt.Fprintf(w, "# HELP updu_info Application version info.\n")
	fmt.Fprintf(w, "# TYPE updu_info gauge\n")
	fmt.Fprintf(w, "updu_info{version=%q} 1\n", version.Version)

	// Monitor gauges
	fmt.Fprintf(w, "# HELP updu_monitors_total Total number of configured monitors.\n")
	fmt.Fprintf(w, "# TYPE updu_monitors_total gauge\n")
	fmt.Fprintf(w, "updu_monitors_total %d\n", metrics.TotalMonitors)

	fmt.Fprintf(w, "# HELP updu_monitors_up Number of monitors currently up.\n")
	fmt.Fprintf(w, "# TYPE updu_monitors_up gauge\n")
	fmt.Fprintf(w, "updu_monitors_up %d\n", metrics.MonitorsUp)

	fmt.Fprintf(w, "# HELP updu_monitors_down Number of monitors currently down.\n")
	fmt.Fprintf(w, "# TYPE updu_monitors_down gauge\n")
	fmt.Fprintf(w, "updu_monitors_down %d\n", metrics.MonitorsDown)

	fmt.Fprintf(w, "# HELP updu_monitors_degraded Number of monitors in degraded state.\n")
	fmt.Fprintf(w, "# TYPE updu_monitors_degraded gauge\n")
	fmt.Fprintf(w, "updu_monitors_degraded %d\n", metrics.MonitorsDegraded)

	fmt.Fprintf(w, "# HELP updu_monitors_paused Number of paused monitors.\n")
	fmt.Fprintf(w, "# TYPE updu_monitors_paused gauge\n")
	fmt.Fprintf(w, "updu_monitors_paused %d\n", metrics.MonitorsPaused)

	// Incidents
	fmt.Fprintf(w, "# HELP updu_incidents_active Number of active (unresolved) incidents.\n")
	fmt.Fprintf(w, "# TYPE updu_incidents_active gauge\n")
	fmt.Fprintf(w, "updu_incidents_active %d\n", metrics.ActiveIncidents)

	// SSE clients
	fmt.Fprintf(w, "# HELP updu_sse_clients Number of connected SSE clients.\n")
	fmt.Fprintf(w, "# TYPE updu_sse_clients gauge\n")
	fmt.Fprintf(w, "updu_sse_clients %d\n", s.sse.ClientCount())

	// Scheduler
	fmt.Fprintf(w, "# HELP updu_scheduler_monitors Number of monitors scheduled.\n")
	fmt.Fprintf(w, "# TYPE updu_scheduler_monitors gauge\n")
	fmt.Fprintf(w, "updu_scheduler_monitors %d\n", s.scheduler.MonitorCount())

	// Go runtime
	fmt.Fprintf(w, "# HELP updu_go_goroutines Number of goroutines.\n")
	fmt.Fprintf(w, "# TYPE updu_go_goroutines gauge\n")
	fmt.Fprintf(w, "updu_go_goroutines %d\n", runtime.NumGoroutine())

	fmt.Fprintf(w, "# HELP updu_go_memory_alloc_bytes Current heap allocation in bytes.\n")
	fmt.Fprintf(w, "# TYPE updu_go_memory_alloc_bytes gauge\n")
	fmt.Fprintf(w, "updu_go_memory_alloc_bytes %d\n", memStats.Alloc)

	fmt.Fprintf(w, "# HELP updu_go_memory_sys_bytes Total memory obtained from the OS.\n")
	fmt.Fprintf(w, "# TYPE updu_go_memory_sys_bytes gauge\n")
	fmt.Fprintf(w, "updu_go_memory_sys_bytes %d\n", memStats.Sys)

	fmt.Fprintf(w, "# HELP updu_go_gc_completed_total Total number of completed GC cycles.\n")
	fmt.Fprintf(w, "# TYPE updu_go_gc_completed_total counter\n")
	fmt.Fprintf(w, "updu_go_gc_completed_total %d\n", memStats.NumGC)
}

func (s *Server) handleCustomCSS(w http.ResponseWriter, r *http.Request) {
	if !s.config.EnableCustomCSS {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.Write([]byte("/* custom CSS is disabled */"))
		return
	}

	css, err := s.db.GetSetting(r.Context(), "custom_css")
	if err != nil {
		css = ""
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Write([]byte(sanitizeCSS(css)))
}

// sanitizeCSS strips dangerous CSS constructs that could be used for data exfiltration or XSS.
func sanitizeCSS(css string) string {
	// Remove @import rules (data exfiltration via external URLs)
	css = importRe.ReplaceAllString(css, "/* blocked @import */")
	// Remove url(javascript:...) (XSS in legacy browsers)
	css = jsURLRe.ReplaceAllString(css, "/* blocked */")
	// Remove url(data:...) (data exfiltration via CSS)
	css = dataURLRe.ReplaceAllString(css, "/* blocked */")
	// Remove expression() (IE code execution)
	css = expressionRe.ReplaceAllString(css, "/* blocked */")
	// Remove -moz-binding (Firefox XBL)
	css = mozBindingRe.ReplaceAllString(css, "/* blocked */")
	// Remove behavior: (IE HTC)
	css = behaviorRe.ReplaceAllString(css, "/* blocked */")
	// Remove </style> tags to prevent breaking out of the style element (XSS vector)
	css = styleTagRe.ReplaceAllString(css, "/* blocked */")
	return css
}

func (s *Server) handleSetupCheck(w http.ResponseWriter, r *http.Request) {
	count, err := s.db.CountUsers(r.Context())
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	policy := config.NormalizePasswordPolicy(s.config.PasswordPolicy)
	jsonOK(w, map[string]any{
		"setup_required":       count == 0,
		"password_policy":      policy,
		"password_policy_hint": auth.PasswordPolicyHint(policy),
	})
}

func (s *Server) handleAuthProviders(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]bool{
		"oidc": s.auth.IsOIDCConfigured(),
	})
}

// --- Helpers ---

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'")

		next.ServeHTTP(w, r)

		slog.Debug("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}

// maxBody wraps a handler to limit request body size.
func maxBody(limit int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next(w, r)
	}
}
