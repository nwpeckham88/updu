package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/updu/updu/internal/models"
)

// TCPChecker monitors TCP port availability.
type TCPChecker struct{}

func (c *TCPChecker) Type() string { return "tcp" }

func (c *TCPChecker) Validate(config json.RawMessage) error {
	var cfg models.TCPMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid TCP config: %w", err)
	}
	if cfg.Host == "" {
		return fmt.Errorf("host is required")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func (c *TCPChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.TCPMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	timeout := time.Duration(monitor.TimeoutS) * time.Second

	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	latency := int(time.Since(start).Milliseconds())

	result := &models.CheckResult{
		MonitorID: monitor.ID,
		LatencyMs: &latency,
		CheckedAt: time.Now(),
	}

	if err != nil {
		result.Status = models.StatusDown
		result.Message = err.Error()
	} else {
		conn.Close()
		result.Status = models.StatusUp
		result.Message = fmt.Sprintf("connected to %s", addr)
	}

	return result, nil
}
