package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/models"
)

// SingleStatusResponse represents lightweight status for external services / dashboards.
type SingleStatusResponse struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Status     string   `json:"status"`
	Enabled    bool     `json:"enabled"`
	LatencyMs  *int     `json:"latency_ms,omitempty"`
	Uptime24h  *float64 `json:"uptime_24h,omitempty"`
	LastCheck  *string  `json:"last_check,omitempty"`
	Message    string   `json:"message,omitempty"`
	BadgeColor string   `json:"badge_color"`
	Groups     []string `json:"groups,omitempty"`
}

// BatchStatusResponse represents a collection of service statuses.
type BatchStatusResponse struct {
	Services []SingleStatusResponse `json:"services"`
}

// resolveEffectiveUser returns the authenticated user, or an anonymous viewer if called
// from a trusted proxy or localhost.
func (s *Server) resolveEffectiveUser(r *http.Request) *models.User {
	user := auth.UserFromContext(r.Context())
	if user != nil {
		return user
	}
	// Fallback for trusted proxies or localhost if unauthenticated
	if s.config != nil && (s.config.IsTrustedProxy(r.RemoteAddr) || strings.HasPrefix(r.RemoteAddr, "127.0.0.1") || strings.HasPrefix(r.RemoteAddr, "[::1]")) {
		return &models.User{
			ID:       "trusted-proxy-anonymous",
			Username: "anonymous",
			Role:     models.RoleViewer,
		}
	}
	return nil
}

// findMonitorByIdentifier finds a monitor by ID, slug, or case-insensitive name.
func (s *Server) findMonitorByIdentifier(r *http.Request, identifier string) (*models.Monitor, error) {
	ctx := r.Context()
	decoded, err := url.PathUnescape(identifier)
	if err == nil && decoded != "" {
		identifier = decoded
	}

	// 1. Direct ID lookup
	m, err := s.db.GetMonitor(ctx, identifier)
	if err == nil && m != nil {
		return m, nil
	}

	// 2. Lookup across all monitors (match Name or ID)
	monitors, err := s.db.ListMonitors(ctx)
	if err != nil {
		return nil, err
	}

	for _, mon := range monitors {
		if strings.EqualFold(mon.ID, identifier) || strings.EqualFold(mon.Name, identifier) {
			return mon, nil
		}
	}

	return nil, nil
}

// GET /api/v1/status/{target}
func (s *Server) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("target")
	if target == "" {
		target = r.URL.Query().Get("id")
		if target == "" {
			target = r.URL.Query().Get("name")
		}
	}
	if target == "" {
		jsonError(w, "missing service or monitor identifier", http.StatusBadRequest)
		return
	}

	user := s.resolveEffectiveUser(r)
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	m, err := s.findMonitorByIdentifier(r, target)
	if err != nil {
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if m == nil || !canAccessMonitor(user, m) {
		jsonError(w, "service or monitor not found", http.StatusNotFound)
		return
	}

	resp := s.buildStatusResponse(r, m)
	jsonOK(w, resp)
}

// GET /api/v1/status
func (s *Server) handleBatchStatus(w http.ResponseWriter, r *http.Request) {
	user := s.resolveEffectiveUser(r)
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	monitors, err := s.db.ListMonitors(r.Context())
	if err != nil {
		jsonError(w, "failed to list monitors", http.StatusInternalServerError)
		return
	}

	// Filter by query parameters if supplied
	filterGroup := strings.TrimSpace(r.URL.Query().Get("group"))
	filterIDs := make(map[string]bool)
	if rawIDs := r.URL.Query().Get("ids"); rawIDs != "" {
		for _, id := range strings.Split(rawIDs, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				filterIDs[strings.ToLower(id)] = true
			}
		}
	}

	var results []SingleStatusResponse
	for _, m := range monitors {
		if !canAccessMonitor(user, m) {
			continue
		}

		if len(filterIDs) > 0 {
			if !filterIDs[strings.ToLower(m.ID)] && !filterIDs[strings.ToLower(m.Name)] {
				continue
			}
		}

		if filterGroup != "" {
			matched := false
			for _, g := range m.Groups {
				if strings.EqualFold(strings.TrimSpace(g), filterGroup) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		results = append(results, s.buildStatusResponse(r, m))
	}

	jsonOK(w, BatchStatusResponse{Services: results})
}

// GET /api/v1/status/{target}/badge
func (s *Server) handleStatusBadge(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("target")
	user := s.resolveEffectiveUser(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	m, err := s.findMonitorByIdentifier(r, target)
	if err != nil || m == nil || !canAccessMonitor(user, m) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	status := string(m.Status)
	if !m.Enabled {
		status = "paused"
	} else {
		latest, _ := s.db.GetLatestCheck(r.Context(), m.ID)
		if latest != nil {
			status = string(latest.Status)
		} else {
			status = "pending"
		}
	}

	label := r.URL.Query().Get("label")
	if label == "" {
		label = "status"
	}

	svg := generateStatusBadgeSVG(label, status)
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache, max-age=15")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(svg))
}

func (s *Server) buildStatusResponse(r *http.Request, m *models.Monitor) SingleStatusResponse {
	ctx := r.Context()
	st := string(m.Status)
	var lat *int
	var lastCheckStr *string
	var msg string

	if !m.Enabled {
		st = "paused"
	} else {
		latest, _ := s.db.GetLatestCheck(ctx, m.ID)
		if latest != nil {
			st = string(latest.Status)
			lat = latest.LatencyMs
			t := latest.CheckedAt.UTC().Format(time.RFC3339)
			lastCheckStr = &t
			msg = latest.Message
		} else {
			st = "pending"
		}
	}

	var uptimePtr *float64
	if uptime, err := s.db.GetUptimePercent(ctx, m.ID, time.Now().Add(-24*time.Hour)); err == nil {
		uptimePtr = &uptime
	}
	badgeColor := "brightgreen"
	switch st {
	case "down":
		badgeColor = "red"
	case "degraded":
		badgeColor = "yellow"
	case "paused":
		badgeColor = "lightgrey"
	case "pending":
		badgeColor = "blue"
	}

	return SingleStatusResponse{
		ID:         m.ID,
		Name:       m.Name,
		Type:       m.Type,
		Status:     st,
		Enabled:    m.Enabled,
		LatencyMs:  lat,
		Uptime24h:  uptimePtr,
		LastCheck:  lastCheckStr,
		Message:    msg,
		BadgeColor: badgeColor,
		Groups:     m.Groups,
	}
}

func generateStatusBadgeSVG(label, status string) string {
	var color string
	switch status {
	case "up":
		color = "#22c55e" // green
	case "down":
		color = "#ef4444" // red
	case "degraded":
		color = "#f59e0b" // yellow
	case "paused":
		color = "#6b7280" // gray
	default:
		color = "#3b82f6" // blue
	}

	labelWidth := len(label)*6 + 12
	statusWidth := len(status)*6 + 12
	totalWidth := labelWidth + statusWidth

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
  <linearGradient id="b" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="%d" height="20" rx="3" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <path fill="#555" d="M0 0h%dv20H0z"/>
    <path fill="%s" d="M%d 0h%dv20H%dz"/>
    <path fill="url(#b)" d="M0 0h%dv20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="14">%s</text>
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="14">%s</text>
  </g>
</svg>`,
		totalWidth, totalWidth,
		labelWidth,
		color, labelWidth, statusWidth, labelWidth,
		totalWidth,
		labelWidth/2, label,
		labelWidth/2, label,
		labelWidth+statusWidth/2, status,
		labelWidth+statusWidth/2, status,
	)
}
