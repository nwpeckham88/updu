package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func TestAggregator_AggregateAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 1. Create a monitor
	m := &models.Monitor{
		ID:        "agg-monitor-1",
		Name:      "Agg Test",
		Type:      "http",
		Config:    []byte(`{"url":"http://test"}`),
		CreatedBy: "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.CreateMonitor(context.Background(), m); err != nil {
		t.Fatalf("failed creating monitor: %v", err)
	}

	// 2. Insert older check results
	// Let's create checks that fall into the previous 5-minute bucket
	now := time.Now()
	// Round down to nearest 5m
	bucketStart := now.Truncate(5 * time.Minute)
	if now.Sub(bucketStart) < 1*time.Minute {
		// Ensure we're not right on the edge
		bucketStart = bucketStart.Add(-5 * time.Minute)
	}

	// Two checks inside the same previous 5m bucket
	checkTime1 := bucketStart.Add(1 * time.Minute)
	checkTime2 := bucketStart.Add(3 * time.Minute)

	lat1, lat2 := 40, 60

	err := db.InsertCheckResult(context.Background(), &models.CheckResult{
		MonitorID: m.ID,
		Status:    models.StatusUp,
		LatencyMs: &lat1,
		CheckedAt: checkTime1,
	})
	if err != nil {
		t.Fatalf("failed inserting check 1: %v", err)
	}

	err = db.InsertCheckResult(context.Background(), &models.CheckResult{
		MonitorID: m.ID,
		Status:    models.StatusDown,
		LatencyMs: &lat2,
		CheckedAt: checkTime2,
	})
	if err != nil {
		t.Fatalf("failed inserting check 2: %v", err)
	}

	// 3. Run Aggregator
	agg := NewAggregator(db, 5*time.Minute) // We call AggregateAll manually, interval defines the bucket resolution

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := agg.AggregateAll(ctx); err != nil {
		t.Fatalf("aggregate all failed: %v", err)
	}

	// 4. Verify Aggregate
	var aggCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM check_aggregates WHERE monitor_id = ?", m.ID).Scan(&aggCount)
	if err != nil {
		t.Fatalf("failed counting aggregates: %v", err)
	}

	// Should have at least one 5m aggregate
	if aggCount == 0 {
		t.Fatalf("expected aggregates, got %d", aggCount)
	}

	// Verify specific aggregate data
	var up, down, total int
	var avg sql.NullFloat64
	err = db.QueryRowContext(ctx, "SELECT up_count, down_count, total_checks, avg_latency FROM check_aggregates WHERE monitor_id = ? AND resolution = '5m0s'", m.ID).
		Scan(&up, &down, &total, &avg)
	if err != nil {
		t.Fatalf("failed to read aggregate details: %v", err)
	}

	if total != 2 {
		t.Errorf("expected 2 total checks, got %d", total)
	}
	if up != 1 {
		t.Errorf("expected 1 up check, got %d", up)
	}
	if down != 1 {
		t.Errorf("expected 1 down check, got %d", down)
	}
	if avg.Valid && avg.Float64 != 50.0 {
		t.Errorf("expected average latency 50.0, got %f", avg.Float64)
	}
}

func TestAggregator_StartStop(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	agg := NewAggregator(db, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agg.Start(ctx)

	// Wait momentarily to ensure goroutine started
	time.Sleep(20 * time.Millisecond)

	// Stop the aggregator
	agg.Stop()

	// Stopping again should be safe
	agg.Stop()
}
