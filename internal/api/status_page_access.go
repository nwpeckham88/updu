package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/updu/updu/internal/models"
)

const statusPageAccessCookieMaxAge = 24 * 60 * 60

func statusPageAccessCookieName(slug string) string {
	sum := sha256.Sum256([]byte(slug))
	return "updu_status_access_" + hex.EncodeToString(sum[:8])
}

func (s *Server) setStatusPageAccessCookie(w http.ResponseWriter, sp *models.StatusPage) {
	http.SetCookie(w, &http.Cookie{
		Name:     statusPageAccessCookieName(sp.Slug),
		Value:    s.statusPageAccessToken(sp),
		Path:     "/",
		MaxAge:   statusPageAccessCookieMaxAge,
		Expires:  time.Now().Add(statusPageAccessCookieMaxAge * time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.config.IsSecure(),
	})
}

func (s *Server) hasStatusPageAccess(r *http.Request, sp *models.StatusPage) bool {
	if sp == nil || sp.Password == "" {
		return false
	}

	cookie, err := r.Cookie(statusPageAccessCookieName(sp.Slug))
	if err != nil || cookie.Value == "" {
		return false
	}

	expected := s.statusPageAccessToken(sp)
	if len(cookie.Value) != len(expected) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(expected)) == 1
}

func (s *Server) statusPageAccessToken(sp *models.StatusPage) string {
	mac := hmac.New(sha256.New, []byte(s.config.AuthSecret))
	mac.Write([]byte(sp.Slug))
	mac.Write([]byte{0})
	mac.Write([]byte(sp.Password))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) hasAuthenticatedSession(r *http.Request) bool {
	user, err := s.sessionUser(r)
	return err == nil && user != nil
}

func (s *Server) hasAdminSession(r *http.Request) bool {
	user, err := s.sessionUser(r)
	return err == nil && user != nil && user.Role == models.RoleAdmin
}

func (s *Server) sessionUser(r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie("updu_session")
	if err != nil || cookie.Value == "" {
		return nil, err
	}

	session, err := s.db.GetSession(r.Context(), cookie.Value)
	if err != nil || session == nil {
		return nil, err
	}

	if session.ExpiresAt.Before(time.Now()) {
		return nil, http.ErrNoCookie
	}

	return s.db.GetUserByID(r.Context(), session.UserID)
}
