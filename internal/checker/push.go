package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/updu/updu/internal/models"
)

// PushChecker implements checking logically for passive Heartbeat/Push monitors.
// For Push monitors, the actual check is passive; the "check loop" just evaluates
// how much time has passed since `LastCheck`. If `time.Since(LastCheck) > IntervalS + grace`,
// then the monitor is considered down.
type PushChecker struct{}

// Type returns the monitor type
func (c *PushChecker) Type() string {
	return "push"
}

// Check evaluates whether the push monitor is still alive based on LastCheck time.
func (c *PushChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var config models.PushMonitorConfig
	if len(monitor.Config) > 0 {
		if err := json.Unmarshal(monitor.Config, &config); err != nil {
			return nil, fmt.Errorf("invalid push config: %w", err)
		}
	}

	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: time.Now(),
		Status:    models.StatusDown,
		Message:   "No check-in arrived within the expected window",
	}

	if monitor.LastCheck == nil || monitor.LastCheck.IsZero() {
		result.Status = models.StatusPending
		result.Message = "Waiting for the first check-in"
		return result, nil
	}

	expectedInterval := time.Duration(monitor.IntervalS) * time.Second
	gracePeriod := config.EffectiveGraceDuration(monitor.IntervalS)
	maxAllowedAge := expectedInterval + gracePeriod

	age := time.Since(*monitor.LastCheck)

	if age <= maxAllowedAge {
		result.Status = models.StatusUp
		result.Message = fmt.Sprintf("Last check-in received %v ago", age.Round(time.Second))
	} else {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("Check-in overdue. Last received %v ago (expected within %s)", age.Round(time.Second), maxAllowedAge)
	}

	return result, nil
}

// Validate ensures the monitor config is valid for a Push Checker
func (c *PushChecker) Validate(config json.RawMessage) error {
	var conf models.PushMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid push config: %w", err)
	}

	if conf.Token == "" {
		return fmt.Errorf("token is required for push monitors")
	}
	if conf.GracePeriodS != nil && *conf.GracePeriodS < 0 {
		return fmt.Errorf("grace_period_s must be zero or greater")
	}
	if conf.GracePeriodS != nil && *conf.GracePeriodS > models.MaxPushGraceSeconds {
		return fmt.Errorf(
			"grace_period_s must not exceed %d seconds",
			models.MaxPushGraceSeconds,
		)
	}

	return nil
}
