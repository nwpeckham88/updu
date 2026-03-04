package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/updu/updu/internal/config"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/storage"
)

func setupAuthTest(t *testing.T) (*Auth, *storage.DB, func()) {
	tempDir, err := os.MkdirTemp("", "updu-auth-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	cfg := &config.Config{
		SessionTTLDays: 7,
	}

	auth := New(db, cfg)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return auth, db, cleanup
}

func TestRegister(t *testing.T) {
	auth, _, cleanup := setupAuthTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test first user (admin)
	user1, err := auth.Register(ctx, "admin", "password123")
	if err != nil {
		t.Fatalf("failed to register admin: %v", err)
	}
	if user1.Role != models.RoleAdmin {
		t.Errorf("expected first user to be admin, got %s", user1.Role)
	}

	// Test second user (viewer)
	user2, err := auth.Register(ctx, "viewer", "password123")
	if err != nil {
		t.Fatalf("failed to register viewer: %v", err)
	}
	if user2.Role != models.RoleViewer {
		t.Errorf("expected second user to be viewer, got %s", user2.Role)
	}
}

func TestLoginLogout(t *testing.T) {
	auth, _, cleanup := setupAuthTest(t)
	defer cleanup()

	ctx := context.Background()
	auth.Register(ctx, "testuser", "password123")

	// Successful login
	session, err := auth.Login(ctx, "testuser", "password123", "agent", "1.2.3.4")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if session == nil {
		t.Fatal("expected session, got nil")
	}

	// Failed login (wrong password)
	_, err = auth.Login(ctx, "testuser", "wrong", "agent", "1.2.3.4")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}

	// Logout
	if err := auth.Logout(ctx, session.ID); err != nil {
		t.Fatalf("logout failed: %v", err)
	}
}

func TestMiddleware(t *testing.T) {
	auth, _, cleanup := setupAuthTest(t)
	defer cleanup()

	ctx := context.Background()
	user, _ := auth.Register(ctx, "testuser", "password123")
	session, _ := auth.Login(ctx, "testuser", "password123", "agent", "1.2.3.4")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil || u.ID != user.ID {
			t.Error("user not found in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Valid session
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session.ID})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", rr.Code)
	}

	// Invalid session
	req = httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "invalid"})
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status Unauthorized, got %d", rr.Code)
	}
}

func TestAdminMiddleware(t *testing.T) {
	auth, _, cleanup := setupAuthTest(t)
	defer cleanup()

	ctx := context.Background()
	userAdmin, _ := auth.Register(ctx, "admin", "password123")   // First is admin
	userViewer, _ := auth.Register(ctx, "viewer", "password123") // Second is viewer

	handler := AdminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 1. Unauthenticated request
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected Forbidden for no session, got %d", rr.Code)
	}

	// 2. Viewer request
	req = httptest.NewRequest("GET", "/", nil)
	// Inject viewer into context
	ctxViewer := context.WithValue(req.Context(), userContextKey, userViewer)
	req = req.WithContext(ctxViewer)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected Forbidden for viewer, got %d", rr.Code)
	}

	// 3. Admin request
	req = httptest.NewRequest("GET", "/", nil)
	// Inject admin into context
	ctxAdmin := context.WithValue(req.Context(), userContextKey, userAdmin)
	req = req.WithContext(ctxAdmin)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected OK for admin, got %d", rr.Code)
	}
}

func TestEnsureFirstUser(t *testing.T) {
	auth, _, cleanup := setupAuthTest(t)
	defer cleanup()

	ctx := context.Background()

	// Initially no users, should just work
	err := auth.EnsureFirstUser(ctx)
	if err != nil {
		t.Fatalf("EnsureFirstUser failed: %v", err)
	}

	// Register a user
	auth.Register(ctx, "admin", "admin")

	// Run again, should still work
	err = auth.EnsureFirstUser(ctx)
	if err != nil {
		t.Fatalf("EnsureFirstUser failed second time: %v", err)
	}
}

func TestSetClearSessionCookie(t *testing.T) {
	auth, _, cleanup := setupAuthTest(t)
	defer cleanup()

	rr := httptest.NewRecorder()

	auth.SetSessionCookie(rr, "test-session", time.Now().Add(1*time.Hour))
	res := rr.Result()
	cookies := res.Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || cookies[0].Value != "test-session" {
		t.Errorf("expected session cookie to be set correctly, got %v", cookies)
	}

	rr2 := httptest.NewRecorder()
	ClearSessionCookie(rr2)
	res2 := rr2.Result()
	cookies2 := res2.Cookies()
	if len(cookies2) != 1 || cookies2[0].Name != sessionCookieName || cookies2[0].Value != "" || cookies2[0].MaxAge > 0 {
		t.Errorf("expected session cookie to be cleared, got %v", cookies2)
	}
}
