package storage

import (
	"context"
	"log/slog"
	"time"
)

// Aggregator handles periodic aggregation of check results.
type Aggregator struct {
	db       *DB
	interval time.Duration
	stop     chan struct{}
}

// NewAggregator creates a new background aggregator.
func NewAggregator(db *DB, interval time.Duration) *Aggregator {
	return &Aggregator{
		db:       db,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

// Start runs the aggregator in the background.
func (a *Aggregator) Start(ctx context.Context) {
	ticker := time.NewTicker(a.interval)
	slog.Info("starting background aggregator", "interval", a.interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := a.AggregateAll(ctx); err != nil {
					slog.Error("aggregation failed", "error", err)
				}
			case <-a.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the aggregator.
func (a *Aggregator) Stop() {
	select {
	case <-a.stop:
		// already closed
	default:
		close(a.stop)
	}
}

// AggregateAll performs aggregation for all monitors.
func (a *Aggregator) AggregateAll(ctx context.Context) error {
	monitors, err := a.db.ListMonitors(ctx)
	if err != nil {
		return err
	}

	// Round to previous bucket start to avoid partial aggregation
	now := time.Now().Add(-a.interval).Truncate(a.interval)
	for _, m := range monitors {
		if err := a.AggregateMonitor(ctx, m.ID, now); err != nil {
			slog.Error("failed to aggregate monitor", "monitor_id", m.ID, "error", err)
		}
	}
	return nil
}

// AggregateMonitor aggregates results for a specific monitor and time period.
func (a *Aggregator) AggregateMonitor(ctx context.Context, monitorID string, periodStart time.Time) error {
	periodEnd := periodStart.Add(a.interval)

	agg, err := a.db.GetCheckAggregateStats(ctx, monitorID, periodStart, periodEnd)
	if err != nil {
		return err
	}

	agg.Resolution = a.interval.String()

	return a.db.UpsertCheckAggregate(ctx, agg)
}
