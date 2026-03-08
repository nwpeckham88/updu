package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// PingChecker monitors hosts via ICMP ping.
// Uses the system `ping` command for portability (no raw socket needed without CAP_NET_RAW).
type PingChecker struct {
	commander Commander
}

func (c *PingChecker) Type() string { return "ping" }

func (c *PingChecker) Validate(config json.RawMessage) error {
	var cfg models.PingMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid ping config: %w", err)
	}
	if cfg.Host == "" {
		return fmt.Errorf("host is required")
	}
	return nil
}

var pingTimeRegex = regexp.MustCompile(`time[=<]\s*([\d.]+)\s*ms`)

func (c *PingChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.PingMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	count := cfg.Count
	if count <= 0 {
		count = 3
	}

	// SSRF protection: check host before connecting
	if err := CheckHostSSRF(ctx, cfg.Host); err != nil {
		return failResult(monitor.ID, err.Error()), nil
	}

	// Resolve hostname first to check validity
	if _, err := net.LookupHost(cfg.Host); err != nil {
		return &models.CheckResult{
			MonitorID: monitor.ID,
			Status:    models.StatusDown,
			Message:   "dns resolution failed: " + err.Error(),
			CheckedAt: time.Now(),
		}, nil
	}

	timeout := time.Duration(monitor.TimeoutS) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	commander := c.commander
	if commander == nil {
		commander = &defaultCommander{}
	}

	start := time.Now()
	output, err := commander.CombinedOutput(ctx, "ping", "-c", strconv.Itoa(count), "-W",
		strconv.Itoa(monitor.TimeoutS), cfg.Host)
	totalTime := int(time.Since(start).Milliseconds())

	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: time.Now(),
	}

	if err != nil {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("ping failed: %s", strings.TrimSpace(string(output)))
		result.LatencyMs = &totalTime
		return result, nil
	}

	// Parse average latency from ping output
	matches := pingTimeRegex.FindAllStringSubmatch(string(output), -1)
	if len(matches) > 0 {
		var totalLatency float64
		for _, m := range matches {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil {
				totalLatency += v
			}
		}
		avgLatency := int(totalLatency / float64(len(matches)))
		result.LatencyMs = &avgLatency
	} else {
		result.LatencyMs = &totalTime
	}

	result.Status = models.StatusUp
	result.Message = fmt.Sprintf("%d packets sent, avg %dms", count, *result.LatencyMs)
	return result, nil
}
