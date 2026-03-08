package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// SSHChecker monitors SSH connectivity by checking for the SSH banner.
type SSHChecker struct{}

func (c *SSHChecker) Type() string { return "ssh" }

func (c *SSHChecker) Validate(config json.RawMessage) error {
	var cfg models.SSHMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid SSH config: %w", err)
	}
	if cfg.Host == "" {
		return fmt.Errorf("host is required")
	}
	return nil
}

func (c *SSHChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.SSHMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	// SSRF protection: check host before connecting
	if err := CheckHostSSRF(ctx, cfg.Host); err != nil {
		return failResult(monitor.ID, err.Error()), nil
	}

	port := cfg.Port
	if port <= 0 {
		port = 22
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	timeout := time.Duration(monitor.TimeoutS) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

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
		return result, nil
	}
	defer conn.Close()

	// Read the SSH banner (e.g., "SSH-2.0-OpenSSH_8.9p1")
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		result.Status = models.StatusDown
		result.Message = "connected but no SSH banner: " + err.Error()
		return result, nil
	}

	banner := strings.TrimSpace(string(buf[:n]))
	if !strings.HasPrefix(banner, "SSH-") {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("not an SSH server: %q", banner)
		return result, nil
	}

	result.Status = models.StatusUp
	result.Message = banner
	return result, nil
}
