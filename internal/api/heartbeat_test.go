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

	// 3. Token route supports GET without query token.
	reqTokenGet := httptest.NewRequest("GET", "/heartbeat/test-token", nil)
	rrTokenGet := httptest.NewRecorder()
	router.ServeHTTP(rrTokenGet, reqTokenGet)
	if rrTokenGet.Code != http.StatusOK {
		t.Fatalf("expected 200 for token get heartbeat, got %d", rrTokenGet.Code)
	}

	// 4. Token route supports POST without query token.
	reqTokenPost := httptest.NewRequest("POST", "/heartbeat/test-token?status=down", nil)
	rrTokenPost := httptest.NewRecorder()
	router.ServeHTTP(rrTokenPost, reqTokenPost)
	if rrTokenPost.Code != http.StatusOK {
		t.Fatalf("expected 200 for token post heartbeat, got %d", rrTokenPost.Code)
	}

	// 5. Legacy slug route stays POST-only.
	reqLegacyGet := httptest.NewRequest("GET", "/api/v1/heartbeat/test-slug?token=test-token", nil)
	rrLegacyGet := httptest.NewRecorder()
	router.ServeHTTP(rrLegacyGet, reqLegacyGet)
	if rrLegacyGet.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for legacy slug get heartbeat, got %d", rrLegacyGet.Code)
	}
}
