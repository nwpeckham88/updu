package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// RedisChecker implements checking for Redis servers via RESP protocol.
type RedisChecker struct{}

// Type returns the monitor type
func (c *RedisChecker) Type() string {
	return "redis"
}

// Check evaluates whether the Redis server responds to a PING.
func (c *RedisChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: start,
		Status:    models.StatusDown,
	}

	var conf models.RedisMonitorConfig
	if err := json.Unmarshal(monitor.Config, &conf); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	address := fmt.Sprintf("%s:%d", conf.Host, conf.Port)

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to connect to Redis: %v", err)
		return result, nil
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	// Simple RESP Authentication if password is provided
	// AUTH <password> => *2\r\n$4\r\nAUTH\r\n$<len>\r\n<password>\r\n or simple command
	if conf.Password != "" {
		authCmd := fmt.Sprintf("AUTH %s\r\n", conf.Password)
		if _, err := conn.Write([]byte(authCmd)); err != nil {
			result.Message = fmt.Sprintf("Failed to write AUTH command: %v", err)
			return result, nil
		}

		respParams := make([]byte, 1024)
		n, err := conn.Read(respParams)
		if err != nil {
			result.Message = fmt.Sprintf("Empty or failed AUTH response: %v", err)
			return result, nil
		}

		authRes := string(respParams[:n])
		if !strings.HasPrefix(authRes, "+OK") {
			result.Message = fmt.Sprintf("Redis AUTH failed: %s", authRes)
			return result, nil
		}
	}

	// Simple SELECT if Database > 0
	if conf.Database > 0 {
		selectCmd := fmt.Sprintf("SELECT %d\r\n", conf.Database)
		if _, err := conn.Write([]byte(selectCmd)); err != nil {
			result.Message = fmt.Sprintf("Failed to write SELECT command: %v", err)
			return result, nil
		}

		respParams := make([]byte, 1024)
		n, err := conn.Read(respParams)
		if err != nil || !strings.HasPrefix(string(respParams[:n]), "+OK") {
			result.Message = fmt.Sprintf("Redis SELECT DB failed")
			return result, nil
		}
	}

	// Ping Command => PING\r\n
	if _, err := conn.Write([]byte("PING\r\n")); err != nil {
		result.Message = fmt.Sprintf("Failed to write PING command: %v", err)
		return result, nil
	}

	respParams := make([]byte, 1024)
	n, err := conn.Read(respParams)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to read PONG response: %v", err)
		return result, nil
	}

	pongRes := string(respParams[:n])
	if !strings.HasPrefix(pongRes, "+PONG") {
		result.Message = fmt.Sprintf("Unexpected PING response: %s", pongRes)
		return result, nil
	}

	latency := int(time.Since(start).Milliseconds())
	result.LatencyMs = &latency
	result.Status = models.StatusUp
	result.Message = "Redis PING successful"
	return result, nil
}

// Validate ensures the monitor config is valid for a Redis Checker
func (c *RedisChecker) Validate(config json.RawMessage) error {
	var conf models.RedisMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid redis config: %w", err)
	}

	if conf.Host == "" {
		return fmt.Errorf("host is required for redis monitors")
	}

	if conf.Port == 0 || conf.Port > 65535 {
		return fmt.Errorf("a valid port (1-65535) is required for redis monitors (default is 6379)")
	}

	return nil
}
