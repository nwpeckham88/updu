package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestAPI_PushMonitorGracePeriodSync(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	sessionCookie := registerAndLoginAdmin(t, router)

	createBody, _ := json.Marshal(map[string]any{
		"name":       "Push Grace Monitor",
		"type":       "push",
		"interval_s": 60,
		"config": map[string]any{
			"token":          "push-grace-token",
			"grace_period_s": 90,
		},
	})
	createReq := httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(createBody))
	createReq.AddCookie(sessionCookie)
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create monitor to succeed, got %d: %s", createRR.Code, createRR.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	heartbeat, err := db.GetHeartbeat(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("failed to load heartbeat after create: %v", err)
	}
	if heartbeat == nil {
		t.Fatal("expected heartbeat record to be created")
	}
	if heartbeat.GraceS != 90 {
		t.Fatalf("expected heartbeat grace to be 90 seconds after create, got %d", heartbeat.GraceS)
	}
	if heartbeat.ExpectedS != 60 {
		t.Fatalf("expected heartbeat cadence to be 60 seconds after create, got %d", heartbeat.ExpectedS)
	}

	updateBody, _ := json.Marshal(map[string]any{
		"name":       "Push Grace Monitor",
		"type":       "push",
		"interval_s": 120,
		"enabled":    true,
		"config": map[string]any{
			"token":          "push-grace-token",
			"grace_period_s": 180,
		},
	})
	updateReq := httptest.NewRequest("PUT", "/api/v1/monitors/"+created.ID, bytes.NewBuffer(updateBody))
	updateReq.AddCookie(sessionCookie)
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)
	if updateRR.Code != http.StatusOK {
		t.Fatalf("expected update monitor to succeed, got %d: %s", updateRR.Code, updateRR.Body.String())
	}

	heartbeat, err = db.GetHeartbeat(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("failed to load heartbeat after update: %v", err)
	}
	if heartbeat == nil {
		t.Fatal("expected heartbeat record to still exist after update")
	}
	if heartbeat.GraceS != 180 {
		t.Fatalf("expected heartbeat grace to be 180 seconds after update, got %d", heartbeat.GraceS)
	}
	if heartbeat.ExpectedS != 120 {
		t.Fatalf("expected heartbeat cadence to be 120 seconds after update, got %d", heartbeat.ExpectedS)
	}
}

func TestAPI_UpdatePushMonitorRejectsInvalidGracePeriod(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	sessionCookie := registerAndLoginAdmin(t, router)

	createBody, _ := json.Marshal(map[string]any{
		"name":       "Push Invalid Update Monitor",
		"type":       "push",
		"interval_s": 60,
		"config": map[string]any{
			"token":          "push-invalid-update-token",
			"grace_period_s": 60,
		},
	})
	createReq := httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(createBody))
	createReq.AddCookie(sessionCookie)
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create monitor to succeed, got %d: %s", createRR.Code, createRR.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	updateBody, _ := json.Marshal(map[string]any{
		"name":       "Push Invalid Update Monitor",
		"type":       "push",
		"interval_s": 60,
		"enabled":    true,
		"config": map[string]any{
			"token":          "push-invalid-update-token",
			"grace_period_s": models.MaxPushGraceSeconds + 1,
		},
	})
	updateReq := httptest.NewRequest("PUT", "/api/v1/monitors/"+created.ID, bytes.NewBuffer(updateBody))
	updateReq.AddCookie(sessionCookie)
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)
	if updateRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid update to return 400, got %d: %s", updateRR.Code, updateRR.Body.String())
	}

	heartbeat, err := db.GetHeartbeat(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("failed to load heartbeat after invalid update: %v", err)
	}
	if heartbeat == nil {
		t.Fatal("expected heartbeat record to exist after invalid update")
	}
	if heartbeat.GraceS != 60 {
		t.Fatalf("expected heartbeat grace to remain 60 seconds after rejected update, got %d", heartbeat.GraceS)
	}
}

func TestAPI_PushMonitorZeroGracePeriodSync(t *testing.T) {
	srv, db, cleanup := setupAPITest(t)
	defer cleanup()

	router := srv.Router()
	sessionCookie := registerAndLoginAdmin(t, router)

	createBody, _ := json.Marshal(map[string]any{
		"name":       "Push Zero Grace Monitor",
		"type":       "push",
		"interval_s": 60,
		"config": map[string]any{
			"token":          "push-zero-grace-token",
			"grace_period_s": 0,
		},
	})
	createReq := httptest.NewRequest("POST", "/api/v1/monitors", bytes.NewBuffer(createBody))
	createReq.AddCookie(sessionCookie)
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create monitor to succeed, got %d: %s", createRR.Code, createRR.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	heartbeat, err := db.GetHeartbeat(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("failed to load heartbeat after create: %v", err)
	}
	if heartbeat == nil {
		t.Fatal("expected heartbeat record to be created")
	}
	if heartbeat.GraceS != 0 {
		t.Fatalf("expected heartbeat grace to be 0 seconds, got %d", heartbeat.GraceS)
	}
}
