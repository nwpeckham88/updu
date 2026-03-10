package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/updu/updu/internal/models"
)

// CompositeChecker evaluates K-of-N quorum across a set of other monitors.
// It reads current statuses from storage rather than making network calls.
type CompositeChecker struct {
	sr StatusReader
}

func (c *CompositeChecker) Type() string { return "composite" }

func (c *CompositeChecker) Validate(config json.RawMessage) error {
	var cfg models.CompositeMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid composite config: %w", err)
	}
	if len(cfg.MonitorIDs) == 0 {
		return fmt.Errorf("monitor_ids must not be empty")
	}
	switch cfg.Mode {
	case "all_up", "any_up":
		// valid
	case "quorum":
		if cfg.Quorum <= 0 {
			return fmt.Errorf("quorum must be > 0")
		}
	default:
		return fmt.Errorf("mode must be one of: all_up, any_up, quorum")
	}
	return nil
}

func (c *CompositeChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.CompositeMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	if c.sr == nil {
		return failResult(monitor.ID, "composite checker not initialised (no StatusReader)"), nil
	}

	statuses, err := c.sr.GetMonitorStatuses(ctx, cfg.MonitorIDs)
	if err != nil {
		return failResult(monitor.ID, "reading monitor statuses: "+err.Error()), nil
	}

	upCount := 0
	for _, id := range cfg.MonitorIDs {
		if s, ok := statuses[id]; ok && s == models.StatusUp {
			upCount++
		}
	}
	total := len(cfg.MonitorIDs)

	// Build metadata snapshot
	statusMap := make(map[string]string, total)
	for _, id := range cfg.MonitorIDs {
		if s, ok := statuses[id]; ok {
			statusMap[id] = string(s)
		} else {
			statusMap[id] = string(models.StatusPending)
		}
	}
	metadata, _ := json.Marshal(map[string]any{
		"monitor_statuses": statusMap,
		"up_count":         upCount,
		"total":            total,
	})

	zero := 0
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		LatencyMs: &zero,
		CheckedAt: time.Now(),
		Metadata:  metadata,
	}

	var quorumMet bool
	switch cfg.Mode {
	case "all_up":
		quorumMet = upCount == total
		if !quorumMet {
			result.Message = fmt.Sprintf("%d/%d monitors up (all required)", upCount, total)
		}
	case "any_up":
		quorumMet = upCount >= 1
		if !quorumMet {
			result.Message = fmt.Sprintf("0/%d monitors up", total)
		}
	case "quorum":
		quorumMet = upCount >= cfg.Quorum
		if !quorumMet {
			result.Message = fmt.Sprintf("%d/%d monitors up (%d required)", upCount, total, cfg.Quorum)
		}
	}

	if quorumMet {
		result.Status = models.StatusUp
		result.Message = fmt.Sprintf("%d/%d monitors up", upCount, total)
	} else {
		result.Status = models.StatusDown
	}

	return result, nil
}
