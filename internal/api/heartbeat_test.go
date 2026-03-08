package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestAPI_Heartbeat(t *testing.T) {
	srv, _, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	ctx := context.Background()

	// Need a user
	u := &models.User{ID: "hb-user", Username: "hb", Role: models.RoleAdmin, CreatedAt: time.Now()}
	srv.db.CreateUser(ctx, u)

	// Create a heartbeat monitor
	m := &models.Monitor{
		ID:        "hb-monitor-1",
		Name:      "HB Mon",
		Type:      "heartbeat",
		Config:    []byte(`{}`), // Using mock config if needed
		Enabled:   true,
		CreatedBy: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := srv.db.CreateMonitor(ctx, m); err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	// Create Heartbeat mapping
	hb := &models.Heartbeat{
		Slug:      "test-slug",
		MonitorID: m.ID,
		ExpectedS: 60,
		GraceS:    10,
		Token:     "test-token",
	}
	if err := srv.db.UpsertHeartbeat(ctx, hb); err != nil {
		t.Fatalf("failed to upsert heartbeat: %v", err)
	}

	// 1. Valid Heartbeat Ping
	req := httptest.NewRequest("POST", "/api/v1/heartbeat/test-slug?token=test-token", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for heartbeat ping, got %d", rr.Code)
	}

	val := rr.Body.String()
	if !strings.Contains(val, "pong") {
		t.Errorf("expected pong in json, got %q", val)
	}

	// Verify it was recorded
	gotHB, _ := srv.db.GetHeartbeat(ctx, "test-slug")
	if gotHB == nil || gotHB.LastPing == nil {
		t.Fatal("expected heartbeat last_ping to be updated")
	}

	// 2. Invalid Heartbeat Ping
	reqInv := httptest.NewRequest("POST", "/api/v1/heartbeat/unknown-slug?token=any", nil)
	rrInv := httptest.NewRecorder()
	router.ServeHTTP(rrInv, reqInv)

	if rrInv.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown slug, got %d", rrInv.Code)
	}
}
