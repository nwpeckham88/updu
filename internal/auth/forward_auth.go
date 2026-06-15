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
	if !cfg.ForwardAuthEnabled {
		return nil
	}

	// Only trust headers from trusted proxies
	if !cfg.IsTrustedProxy(r.RemoteAddr) {
		return nil
	}

	username := strings.TrimSpace(r.Header.Get(cfg.ForwardAuthUserHeader))
	if username == "" {
		return nil
	}

	email := strings.TrimSpace(r.Header.Get(cfg.ForwardAuthEmailHeader))

	var groups []string
	groupsHeader := strings.TrimSpace(r.Header.Get(cfg.ForwardAuthGroupHeader))
	if groupsHeader != "" {
		for _, g := range strings.Split(groupsHeader, ",") {
			g = strings.TrimSpace(g)
			if g != "" {
				groups = append(groups, g)
			}
		}
	}

	isAdmin := false
	adminGroup := strings.ToLower(cfg.ForwardAuthAdminGroup)
	for _, g := range groups {
		if strings.ToLower(g) == adminGroup {
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
