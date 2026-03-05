package checker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestPushChecker(t *testing.T) {
	c := &PushChecker{}
	if c.Type() != "push" {
		t.Errorf("Type() = %v, want push", c.Type())
	}

	// Test Validate
	if err := c.Validate([]byte(`{"token": "my-secret-token"}`)); err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if err := c.Validate([]byte(`{"token": ""}`)); err == nil {
		t.Errorf("Expected error for empty token")
	}
	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Errorf("Expected error for bad json")
	}

	// Test Check
	ctx := context.Background()
	monitor := &models.Monitor{
		ID:        "123",
		IntervalS: 60,
		Config:    json.RawMessage(`{"token": "xyz"}`),
	}

	// 1. Pending (no last check)
	res, err := c.Check(ctx, monitor)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if res.Status != models.StatusPending {
		t.Errorf("Expected StatusPending, got %v", res.Status)
	}

	// 2. Up (recent last check)
	now := time.Now()
	monitor.LastCheck = &now
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusUp {
		t.Errorf("Expected StatusUp, got %v", res.Status)
	}

	// 3. Down (overdue last check)
	oldTime := time.Now().Add(-2 * time.Hour)
	monitor.LastCheck = &oldTime
	res, _ = c.Check(ctx, monitor)
	if res.Status != models.StatusDown {
		t.Errorf("Expected StatusDown, got %v", res.Status)
	}
}
