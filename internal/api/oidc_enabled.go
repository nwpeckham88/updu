//go:build oidc

package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/updu/updu/internal/models"
	"golang.org/x/oauth2"
)

const oidcStateCookie = "oidc_state"

// registerOIDCRoutes configures OIDC routes when the "oidc" build tag is present.
func registerOIDCRoutes(mux *http.ServeMux, s *Server) {
	mux.HandleFunc("GET /api/v1/auth/oidc/login", s.handleOIDCLogin)
	mux.HandleFunc("GET /api/v1/auth/oidc/callback", s.handleOIDCCallback)
}

// getOAuthConfig lazily initializes the OAuth2 config and OIDC provider.
func (s *Server) getOAuthConfig(ctx context.Context) (*oauth2.Config, *oidc.Provider, error) {
	cfg := s.auth.Config()
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, nil, err
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return oauth2Config, provider, nil
}

func (s *Server) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if !s.auth.IsOIDCConfigured() {
		jsonError(w, "OIDC is not configured", http.StatusNotFound)
		return
	}

	oauth2Config, _, err := s.getOAuthConfig(r.Context())
	if err != nil {
		slog.Error("failed to initialize OIDC provider", "error", err)
		jsonError(w, "internal settings error", http.StatusInternalServerError)
		return
	}

	// Generate and store random state string
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.RawURLEncoding.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
		Secure:   s.auth.Config().IsSecure(),
	})

	url := oauth2Config.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if !s.auth.IsOIDCConfigured() {
		http.Error(w, "OIDC is not configured", http.StatusNotFound)
		return
	}

	stateCookie, err := r.Cookie(oidcStateCookie)
	if err != nil {
		http.Error(w, "State cookie missing", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	oauth2Config, provider, err := s.getOAuthConfig(r.Context())
	if err != nil {
		slog.Error("failed to initialize OIDC provider", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found in request", http.StatusBadRequest)
		return
	}

	oauth2Token, err := oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("Failed to exchange token", "error", err)
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Extract the ID Token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token field in oauth2 token", http.StatusInternalServerError)
		return
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: oauth2Config.ClientID})
	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		slog.Error("Failed to verify ID token", "error", err)
		http.Error(w, "Failed to verify ID token", http.StatusInternalServerError)
		return
	}

	var claims struct {
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
		return
	}

	sub := idToken.Subject
	issuer := idToken.Issuer

	// Find or create user
	user, err := s.db.GetUserByOIDCSub(r.Context(), sub, issuer)
	if err != nil {
		slog.Error("Error finding user by OIDC", "error", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if user == nil {
		// Does this user exist by username using PreferredUsername or Email?
		username := claims.PreferredUsername
		if username == "" {
			username = claims.Email
		}
		if username == "" {
			username = claims.Name
		}
		if username == "" {
			username = "oidc-" + sub
		}

		userCount, err := s.db.CountUsers(r.Context())
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		role := models.RoleViewer
		if userCount == 0 {
			role = models.RoleAdmin
			slog.Info("Creating first user via OIDC as admin", "username", username)
		} else {
			// Check if we allow auto-registration? For now, yes, anyone matching OIDC constraints is allowed.
			// Or we could limit based on groups. For UPDU simplicity, SSO is trusted.
		}

		// Ensure username uniqueness
		existingUser, _ := s.db.GetUserByUsername(r.Context(), username)
		if existingUser != nil {
			// Append random string to avoid collision
			username = username + "-" + sub[:5]
		}

		b := make([]byte, 16)
		rand.Read(b)
		id := base64.RawURLEncoding.EncodeToString(b)

		user = &models.User{
			ID:         id,
			Username:   username,
			Password:   "", // Unusable password hash or empty for OIDC only users
			Role:       role,
			OIDCSub:    &sub,
			OIDCIssuer: &issuer,
			CreatedAt:  time.Now(),
		}

		if err := s.db.CreateUser(r.Context(), user); err != nil {
			slog.Error("Failed to create OIDC user", "error", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	}

	// Make sure we pass correct original ip
	ip := r.RemoteAddr

	session, err := s.auth.CreateSession(r.Context(), user.ID, r.UserAgent(), ip)
	if err != nil {
		slog.Error("Failed to create session", "error", err)
		http.Error(w, "Failed to login", http.StatusInternalServerError)
		return
	}

	s.auth.SetSessionCookie(w, session.ID, session.ExpiresAt)

	// Redirect back to frontend
	http.Redirect(w, r, "/", http.StatusFound)
}
