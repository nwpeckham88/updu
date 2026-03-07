package storage

import (
	"context"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestStorage_OIDCAndPass(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Create User
	sub := "sub-123"
	issuer := "issuer-123"
	u := &models.User{
		ID:         "oidc-user",
		Username:   "oidcuser",
		OIDCSub:    &sub,
		OIDCIssuer: &issuer,
		CreatedAt:  time.Now(),
	}
	db.CreateUser(ctx, u)

	// 2. GetByOIDCSub
	got, err := db.GetUserByOIDCSub(ctx, sub, issuer)
	if err != nil || got == nil || got.ID != u.ID {
		t.Errorf("failed to get user by OIDC sub: %v", err)
	}

	// 3. Update Password
	err = db.UpdateUserPassword(ctx, u.ID, "new-secret-hash")
	if err != nil {
		t.Errorf("failed to update password: %v", err)
	}
	u2, _ := db.GetUserByID(ctx, u.ID)
	if u2.Password != "new-secret-hash" {
		t.Errorf("password not updated")
	}
}

func TestStorage_AggregatorEdge(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	agg := NewAggregator(db, 1*time.Hour)

	// AggregateAll on empty DB
	err := agg.AggregateAll(ctx)
	if err != nil {
		t.Errorf("AggregateAll failed on empty DB: %v", err)
	}

	// AggregateMonitor on nonexistent
	err = agg.AggregateMonitor(ctx, "ghost", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Errorf("AggregateMonitor failed on ghost: %v", err)
	}
}

func TestStorage_SQLiteEdge(t *testing.T) {
	// Test double migrate
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.Migrate(context.Background())
	if err != nil {
		t.Errorf("double migrate failed: %v", err)
	}
}
