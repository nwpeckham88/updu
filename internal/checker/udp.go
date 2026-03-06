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

// UDPChecker implements checking for UDP endpoints.
type UDPChecker struct{}

// Type returns the monitor type
func (c *UDPChecker) Type() string {
	return "udp"
}

// Check evaluates whether the UDP server responds within the timeout.
func (c *UDPChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: start,
		Status:    models.StatusDown,
	}

	var conf models.UDPMonitorConfig
	if err := json.Unmarshal(monitor.Config, &conf); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	address := fmt.Sprintf("%s:%d", conf.Host, conf.Port)

	// In Go, UDP Dial doesn't actively send packets over the network until Write
	// It just sets up a local socket
	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", address)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to dial UDP: %v", err)
		return result, nil
	}
	defer conn.Close()

	// If no payload is specified, we just assume "up" if we could resolve the address
	// UDP is connectionless, so without a payload and a response, we can't truly verify
	if conf.SendPayload == "" {
		latency := int(time.Since(start).Milliseconds())
		result.LatencyMs = &latency
		result.Status = models.StatusUp
		result.Message = "UDP address resolved (No payload sent)"
		return result, nil
	}

	// Make sure context dictates deadlines
	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}

	_, err = conn.Write([]byte(conf.SendPayload))
	if err != nil {
		result.Message = fmt.Sprintf("Failed to write UDP payload: %v", err)
		return result, nil
	}

	// Wait for response
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.Message = "UDP read timeout (No response received)"
		} else {
			result.Message = fmt.Sprintf("Failed to read UDP response: %v", err)
		}
		return result, nil
	}

	responseStr := string(buffer[:n])

	if conf.ExpectedResponse != "" && !strings.Contains(responseStr, conf.ExpectedResponse) {
		result.Message = "Received response did not match expected string"
		return result, nil
	}

	latency := int(time.Since(start).Milliseconds())
	result.LatencyMs = &latency
	result.Status = models.StatusUp
	result.Message = "UDP ping successful"
	return result, nil
}

// Validate ensures the monitor config is valid for a UDP Checker
func (c *UDPChecker) Validate(config json.RawMessage) error {
	var conf models.UDPMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid udp config: %w", err)
	}

	if conf.Host == "" {
		return fmt.Errorf("host is required for udp monitors")
	}

	if conf.Port == 0 || conf.Port > 65535 {
		return fmt.Errorf("a valid port (1-65535) is required for udp monitors")
	}

	if conf.ExpectedResponse != "" && conf.SendPayload == "" {
		return fmt.Errorf("expected response requires a send payload")
	}

	return nil
}
