package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"math"
	"net"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/notifier"
	"github.com/updu/updu/internal/realtime"
	"github.com/updu/updu/internal/scheduler"
	"github.com/updu/updu/internal/storage"
)

// CSS sanitization patterns (compiled once)
var (
	importRe     = regexp.MustCompile(`(?i)@import\b`)
	jsURLRe      = regexp.MustCompile(`(?i)url\s*\(\s*["']?javascript:`)
	expressionRe = regexp.MustCompile(`(?i)expression\s*\(`)
	mozBindingRe = regexp.MustCompile(`(?i)-moz-binding\s*:`)
	behaviorRe   = regexp.MustCompile(`(?i)behavior\s*:`)
)

// Server holds all API dependencies.
type Server struct {
	db        *storage.DB
	auth      *auth.Auth
	registry  *checker.Registry
	scheduler *scheduler.Scheduler
	notifier  *notifier.Notifier
	sse       *realtime.Hub

	// Login rate limiting
	loginAttempts map[string]*loginEntry
	loginMu       sync.Mutex
}

type loginEntry struct {
	count       int
	windowStart time.Time
}

// NewServer creates a new API server.
func NewServer(db *storage.DB, a *auth.Auth, reg *checker.Registry, sched *scheduler.Scheduler, n *notifier.Notifier, sse *realtime.Hub) *Server {
	s := &Server{
		db:        db,
		auth:      a,
		registry:  reg,
		scheduler: sched,
		notifier:  n,
		sse:       sse,
	}

	// Periodically clean rate limiter entries
	go func() {
		for range time.Tick(5 * time.Minute) {
			s.loginMu.Lock()
			now := time.Now()
			for ip, entry := range s.loginAttempts {
				if now.Sub(entry.windowStart) > time.Minute {
					delete(s.loginAttempts, ip)
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

	// --- Public routes ---
	mux.HandleFunc("POST /api/v1/auth/login", maxBody(1<<20, s.handleLogin))
	mux.HandleFunc("POST /api/v1/auth/register", maxBody(1<<20, s.handleRegister))
	mux.HandleFunc("GET /api/v1/auth/setup", s.handleSetupCheck)
	mux.HandleFunc("GET /api/v1/auth/providers", s.handleAuthProviders)

	// Register OIDC routes (conditionally compiled via build tags)
	registerOIDCRoutes(mux, s)

	mux.HandleFunc("GET /api/v1/status-pages/{slug}", s.handleGetStatusPage)
	mux.HandleFunc("POST /api/v1/heartbeat/{slug}", maxBody(1<<20, s.handleHeartbeatPing))
	mux.HandleFunc("GET /api/v1/system/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/custom.css", s.handleCustomCSS)

	// --- SSE (authenticated) ---
	mux.Handle("GET /api/v1/events", authed(s.sse.ServeHTTP))

	// --- Authenticated routes ---
	mux.Handle("POST /api/v1/auth/logout", authed(s.handleLogout))
	mux.Handle("GET /api/v1/auth/session", authed(s.handleSession))
	mux.Handle("PUT /api/v1/auth/password", authed(maxBody(1<<20, s.handleChangePassword)))

	mux.Handle("GET /api/v1/monitors", authed(s.handleListMonitors))
	mux.Handle("POST /api/v1/monitors", adminAuthed(maxBody(1<<20, s.handleCreateMonitor)))
	mux.Handle("POST /api/v1/monitors/test", adminAuthed(maxBody(1<<20, s.handleTestMonitor)))
	mux.Handle("GET /api/v1/monitors/{id}", authed(s.handleGetMonitor))
	mux.Handle("PUT /api/v1/monitors/{id}", adminAuthed(maxBody(1<<20, s.handleUpdateMonitor)))
	mux.Handle("DELETE /api/v1/monitors/{id}", adminAuthed(s.handleDeleteMonitor))
	mux.Handle("GET /api/v1/monitors/{id}/checks", authed(s.handleGetMonitorChecks))
	mux.Handle("GET /api/v1/monitors/{id}/uptime", authed(s.handleGetMonitorUptime))

	mux.Handle("GET /api/v1/dashboard", authed(s.handleDashboard))
	mux.Handle("GET /api/v1/stats", authed(s.handleGetStats))

	// Status Pages (Admin for mutations, auth for reads)
	mux.Handle("GET /api/v1/status-pages", authed(s.handleListStatusPages))
	mux.Handle("POST /api/v1/status-pages", adminAuthed(maxBody(1<<20, s.handleCreateStatusPage)))
	mux.Handle("GET /api/v1/status-pages/{id}/detail", authed(s.handleGetStatusPageByID))
	mux.Handle("PUT /api/v1/status-pages/{id}", adminAuthed(maxBody(1<<20, s.handleUpdateStatusPage)))
	mux.Handle("DELETE /api/v1/status-pages/{id}", adminAuthed(s.handleDeleteStatusPage))

	// Notifications (Admin)
	mux.Handle("GET /api/v1/notifications", authed(s.handleListNotificationChannels))
	mux.Handle("POST /api/v1/notifications", adminAuthed(maxBody(1<<20, s.handleCreateNotificationChannel)))
	mux.Handle("GET /api/v1/notifications/{id}", authed(s.handleGetNotificationChannel))
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
	mux.Handle("GET /api/v1/admin/users", adminAuthed(s.handleListUsers))
	mux.Handle("PUT /api/v1/admin/users/{id}/role", adminAuthed(s.handleUpdateUserRole))
	mux.Handle("DELETE /api/v1/admin/users/{id}", adminAuthed(s.handleDeleteUser))

	// Settings (Admin)
	mux.Handle("GET /api/v1/settings", adminAuthed(s.handleGetSettings))
	mux.Handle("POST /api/v1/settings", adminAuthed(maxBody(1<<20, s.handleUpdateSettings)))

	// System (Admin)
	mux.Handle("GET /api/v1/system/metrics", adminAuthed(s.handleGetMetrics))
	mux.Handle("GET /api/v1/system/backup", adminAuthed(s.handleExportConfig))
	mux.Handle("POST /api/v1/system/backup", adminAuthed(maxBody(10<<20, s.handleImportConfig)))

	// Wrap with CORS and logging
	return withMiddleware(mux)
}

// --- Auth Handlers ---

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Rate limit: 5 attempts per IP per minute
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}

	s.loginMu.Lock()
	if s.loginAttempts == nil {
		s.loginAttempts = make(map[string]*loginEntry)
	}
	entry, ok := s.loginAttempts[ip]
	now := time.Now()
	if !ok || now.Sub(entry.windowStart) > time.Minute {
		entry = &loginEntry{windowStart: now}
		s.loginAttempts[ip] = entry
	}
	entry.count++
	count := entry.count
	s.loginMu.Unlock()

	if count > 5 {
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

	session, err := s.auth.Login(r.Context(), req.Username, req.Password, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	s.auth.SetSessionCookie(w, session.ID, session.ExpiresAt)
	jsonOK(w, map[string]any{"message": "logged in"})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
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
	if len(req.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
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
		jsonError(w, "registration failed: "+err.Error(), http.StatusBadRequest)
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
	jsonOK(w, monitors)
}

func (s *Server) handleCreateMonitor(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	var m models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

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
	id, _ := auth.GenerateID()
	m.ID = id
	m.CreatedBy = user.ID
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	if m.IntervalS == 0 {
		m.IntervalS = 60
	}
	if m.TimeoutS == 0 {
		m.TimeoutS = 10
	}
	if m.Retries == 0 {
		m.Retries = 3
	}
	m.Enabled = true

	if err := s.db.CreateMonitor(r.Context(), &m); err != nil {
		jsonError(w, "failed to create monitor", http.StatusInternalServerError)
		return
	}

	// Add to scheduler (use background context — request context dies after response)
	s.scheduler.AddMonitor(context.Background(), &m)

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
	existing.GroupName = update.GroupName
	existing.Tags = update.Tags
	existing.IntervalS = update.IntervalS
	existing.TimeoutS = update.TimeoutS
	existing.Retries = update.Retries
	existing.Enabled = update.Enabled
	existing.ParentID = update.ParentID

	if err := s.db.UpdateMonitor(r.Context(), existing); err != nil {
		jsonError(w, "failed to update monitor", http.StatusInternalServerError)
		return
	}

	s.scheduler.ReloadMonitor(context.Background(), existing)
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
	since24h := time.Now().Add(-24 * time.Hour)
	for _, sm := range summaries {
		monID, _ := sm["id"].(string)
		if monID == "" {
			continue
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

func (s *Server) handleGetStatusPage(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	sp, err := s.db.GetStatusPageBySlug(r.Context(), slug)
	if err != nil || sp == nil {
		jsonError(w, "status page not found", http.StatusNotFound)
		return
	}

	// Enforce visibility
	if !sp.IsPublic {
		cookie, err := r.Cookie("updu_session")
		if err != nil || cookie.Value == "" {
			jsonError(w, "forbidden", http.StatusForbidden)
			return
		}
		session, err := s.db.GetSession(r.Context(), cookie.Value)
		if err != nil || session == nil || session.ExpiresAt.Before(time.Now()) {
			jsonError(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	// Get monitor summaries for this status page
	summaries, _ := s.db.GetMonitorsSummary(r.Context())

	// Filter monitors to only include those in the groups or monitor_ids assigned to this status page
	var filtered []map[string]any
	for _, sm := range summaries {
		groupName, _ := sm["group_name"].(string)
		idStr, _ := sm["id"].(string)

		matched := false
		for _, g := range sp.Groups {
			if g.Name != "" && g.Name == groupName {
				matched = true
				break
			}
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

	var sp models.StatusPage
	if err := json.NewDecoder(r.Body).Decode(&sp); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	id, _ := auth.GenerateID()
	sp.ID = id
	if sp.Slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}

	if err := s.db.CreateStatusPage(r.Context(), &sp); err != nil {
		jsonError(w, "failed to create status page", http.StatusInternalServerError)
		return
	}
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

	var update models.StatusPage
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	existing.Name = update.Name
	existing.Slug = update.Slug
	existing.Description = update.Description
	existing.Groups = update.Groups
	existing.IsPublic = update.IsPublic
	if update.Password != "" {
		existing.Password = update.Password
	}

	if err := s.db.UpdateStatusPage(r.Context(), existing); err != nil {
		jsonError(w, "failed to update status page", http.StatusInternalServerError)
		return
	}
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
	jsonOK(w, map[string]any{"message": "deleted"})
}

// --- System ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]any{"status": "ok"})
}

func (s *Server) handleCustomCSS(w http.ResponseWriter, r *http.Request) {
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
	// Remove expression() (IE code execution)
	css = expressionRe.ReplaceAllString(css, "/* blocked */")
	// Remove -moz-binding (Firefox XBL)
	css = mozBindingRe.ReplaceAllString(css, "/* blocked */")
	// Remove behavior: (IE HTC)
	css = behaviorRe.ReplaceAllString(css, "/* blocked */")
	return css
}

func (s *Server) handleSetupCheck(w http.ResponseWriter, r *http.Request) {
	count, err := s.db.CountUsers(r.Context())
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]bool{"setup_required": count == 0})
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
