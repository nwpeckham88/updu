package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/updu/updu/internal/config"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "updu_session"
	bcryptCost        = 10 // moderate for Pi Zero W
)

type contextKey string

const userContextKey contextKey = "user"

// Auth handles authentication and session management.
type Auth struct {
	db  *storage.DB
	cfg *config.Config
}

// New creates a new Auth instance.
func New(db *storage.DB, cfg *config.Config) *Auth {
	return &Auth{db: db, cfg: cfg}
}

// HashPassword hashes a plaintext password with bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(hash), err
}

// CheckPassword compares a plaintext password against a bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Login validates credentials and creates a session.
func (a *Auth) Login(ctx context.Context, username, password, userAgent, ip string) (*models.Session, error) {
	user, err := a.db.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}
	if !CheckPassword(user.Password, password) {
		return nil, errors.New("invalid credentials")
	}

	return a.createSession(ctx, user.ID, userAgent, ip)
}

// Logout deletes a session.
func (a *Auth) Logout(ctx context.Context, sessionID string) error {
	return a.db.DeleteSession(ctx, sessionID)
}

// EnsureFirstUser creates the initial admin user if no users exist and env vars are set.
func (a *Auth) EnsureFirstUser(ctx context.Context) error {
	count, err := a.db.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// If admin credentials are set via env vars, auto-create the admin user
	if a.cfg.AdminUser != "" && a.cfg.AdminPassword != "" {
		if len(a.cfg.AdminPassword) < 8 {
			slog.Warn("UPDU_ADMIN_PASSWORD is too short (min 8 chars), skipping auto-create")
			return nil
		}
		slog.Info("creating admin user from environment variables", "username", a.cfg.AdminUser)
		_, err := a.Register(ctx, a.cfg.AdminUser, a.cfg.AdminPassword)
		return err
	}

	slog.Info("no users found, first registered user will be admin")
	return nil
}

// Register creates a new user. First user is auto-admin.
func (a *Auth) Register(ctx context.Context, username, password string) (*models.User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	count, err := a.db.CountUsers(ctx)
	if err != nil {
		return nil, err
	}

	role := models.RoleViewer
	if count == 0 {
		role = models.RoleAdmin
		slog.Info("creating first admin user", "username", username)
	}

	id, err := GenerateID()
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:        id,
		Username:  username,
		Password:  hash,
		Role:      role,
		CreatedAt: time.Now(),
	}

	if err := a.db.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Middleware returns an HTTP middleware that validates sessions.
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		session, err := a.db.GetSession(r.Context(), cookie.Value)
		if err != nil || session == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := a.db.GetUserByID(r.Context(), session.UserID)
		if err != nil || user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminMiddleware ensures the authenticated user has admin role.
func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil || user.Role != models.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// UserFromContext extracts the authenticated user from context.
func UserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(userContextKey).(*models.User)
	return u
}

// SetSessionCookie sets the session cookie on the response.
func (a *Auth) SetSessionCookie(w http.ResponseWriter, sessionID string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.cfg.IsSecure(),
	})
}

// ClearSessionCookie removes the session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func (a *Auth) createSession(ctx context.Context, userID, userAgent, ip string) (*models.Session, error) {
	id, err := GenerateID()
	if err != nil {
		return nil, err
	}

	session := &models.Session{
		ID:        id,
		UserID:    userID,
		UserAgent: userAgent,
		IPAddr:    ip,
		ExpiresAt: time.Now().AddDate(0, 0, a.cfg.SessionTTLDays),
		CreatedAt: time.Now(),
	}

	if err := a.db.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// GenerateID generates a secure random 16-byte hex ID.
func GenerateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
