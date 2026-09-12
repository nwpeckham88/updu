package auth

import (
	"net/http"
	"strings"

	"github.com/updu/updu/internal/config"
)

// ForwardAuthIdentity represents a user identity extracted from trusted proxy headers.
type ForwardAuthIdentity struct {
	Username string
	Email    string
	Groups   []string
	IsAdmin  bool
}

// ExtractForwardAuth extracts user identity from trusted proxy headers.
// Returns nil if forward-auth is not active or headers are missing.
func ExtractForwardAuth(cfg *config.Config, r *http.Request) *ForwardAuthIdentity {
	if cfg == nil || !cfg.ForwardAuthEnabled {
		return nil
	}

	// Only trust headers from trusted proxies
	if !cfg.IsTrustedProxy(r.RemoteAddr) {
		return nil
	}

	userHeader := cfg.ForwardAuthUserHeader
	if userHeader == "" {
		userHeader = "Remote-User"
	}
	username := strings.TrimSpace(r.Header.Get(userHeader))
	if username == "" && strings.EqualFold(userHeader, "Remote-User") {
		username = strings.TrimSpace(r.Header.Get("X-Forwarded-User"))
	}
	if username == "" {
		return nil
	}

	emailHeader := cfg.ForwardAuthEmailHeader
	if emailHeader == "" {
		emailHeader = "Remote-Email"
	}
	email := strings.TrimSpace(r.Header.Get(emailHeader))
	if email == "" && strings.EqualFold(emailHeader, "Remote-Email") {
		email = strings.TrimSpace(r.Header.Get("X-Forwarded-Email"))
	}

	groupHeader := cfg.ForwardAuthGroupHeader
	if groupHeader == "" {
		groupHeader = "Remote-Groups"
	}
	groupsHeader := strings.TrimSpace(r.Header.Get(groupHeader))
	if groupsHeader == "" && strings.EqualFold(groupHeader, "Remote-Groups") {
		groupsHeader = strings.TrimSpace(r.Header.Get("X-Forwarded-Groups"))
	}

	var groups []string
	if groupsHeader != "" {
		for _, g := range strings.Split(groupsHeader, ",") {
			g = strings.TrimSpace(g)
			if g != "" {
				groups = append(groups, g)
			}
		}
	}

	isAdmin := false
	adminGroup := strings.ToLower(strings.TrimSpace(cfg.ForwardAuthAdminGroup))
	for _, g := range groups {
		lg := strings.ToLower(g)
		if (adminGroup != "" && lg == adminGroup) || ((adminGroup == "" || adminGroup == "updu-admins") && (lg == "admin" || lg == "admins" || lg == "updu-admins")) {
			isAdmin = true
			break
		}
	}

	return &ForwardAuthIdentity{
		Username: username,
		Email:    email,
		Groups:   groups,
		IsAdmin:  isAdmin,
	}
}
