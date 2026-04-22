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
	if err := c.Validate([]byte(`{"token": "my-secret-token", "grace_period_s": -1}`)); err == nil {
		t.Errorf("Expected error for negative grace period")
	}
	if err := c.Validate([]byte(`{"token": "my-secret-token", "grace_period_s": 999999999}`)); err == nil {
		t.Errorf("Expected error for excessive grace period")
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

func TestPushChecker_UsesConfiguredGracePeriod(t *testing.T) {
	c := &PushChecker{}
	ctx := context.Background()

	withinGrace := time.Now().Add(-110 * time.Second)
	monitor := &models.Monitor{
		ID:        "push-grace",
		IntervalS: 60,
		Config:    json.RawMessage(`{"token":"xyz","grace_period_s":60}`),
		LastCheck: &withinGrace,
	}

	res, err := c.Check(ctx, monitor)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.Status != models.StatusUp {
		t.Fatalf("expected StatusUp within configured grace, got %v (%s)", res.Status, res.Message)
	}

	overdue := time.Now().Add(-121 * time.Second)
	monitor.LastCheck = &overdue
	res, err = c.Check(ctx, monitor)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.Status != models.StatusDown {
		t.Fatalf("expected StatusDown after configured grace, got %v (%s)", res.Status, res.Message)
	}
}

func TestPushChecker_DefaultGracePeriodFallback(t *testing.T) {
	c := &PushChecker{}
	ctx := context.Background()

	// Default grace is min(10% of interval, 10m). For a 60s interval that is 6s,
	// so the monitor is Up while overdue < 66s and Down once it crosses that.
	withinDefaultGrace := time.Now().Add(-63 * time.Second)
	monitor := &models.Monitor{
		ID:        "push-default-grace",
		IntervalS: 60,
		Config:    json.RawMessage(`{"token":"xyz"}`),
		LastCheck: &withinDefaultGrace,
	}

	res, err := c.Check(ctx, monitor)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.Status != models.StatusUp {
		t.Fatalf("expected StatusUp within default grace window, got %v (%s)", res.Status, res.Message)
	}

	overdue := time.Now().Add(-90 * time.Second)
	monitor.LastCheck = &overdue
	res, err = c.Check(ctx, monitor)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.Status != models.StatusDown {
		t.Fatalf("expected StatusDown after default grace window, got %v (%s)", res.Status, res.Message)
	}
}

func TestPushChecker_ZeroGracePeriod(t *testing.T) {
	c := &PushChecker{}
	ctx := context.Background()

	withinInterval := time.Now().Add(-59 * time.Second)
	monitor := &models.Monitor{
		ID:        "push-zero-grace",
		IntervalS: 60,
		Config:    json.RawMessage(`{"token":"xyz","grace_period_s":0}`),
		LastCheck: &withinInterval,
	}

	res, err := c.Check(ctx, monitor)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.Status != models.StatusUp {
		t.Fatalf("expected StatusUp before interval expires, got %v (%s)", res.Status, res.Message)
	}

	overdue := time.Now().Add(-61 * time.Second)
	monitor.LastCheck = &overdue
	res, err = c.Check(ctx, monitor)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.Status != models.StatusDown {
		t.Fatalf("expected StatusDown once interval expires with zero grace, got %v (%s)", res.Status, res.Message)
	}
}
