package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/updu/updu/internal/models"
	"nhooyr.io/websocket"
)

// WebSocketChecker implements checking for WebSocket endpoints.
type WebSocketChecker struct{}

// Type returns the monitor type
func (c *WebSocketChecker) Type() string {
	return "websocket"
}

// Check evaluates whether the WebSocket endpoint is accessible.
func (c *WebSocketChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: start,
		Status:    models.StatusDown,
	}

	var conf models.WebSocketMonitorConfig
	if err := json.Unmarshal(monitor.Config, &conf); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	// Prepare dial options
	opts := &websocket.DialOptions{
		Subprotocols: []string{}, // Add if needed in the future
	}

	// SSRF-safe client
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: SafeDialer(ctx),
	}
	opts.HTTPClient = &http.Client{
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: conf.SkipTLSVerify, // #nosec G402
			},
		},
	}

	// Attempt connection
	conn, resp, err := websocket.Dial(ctx, conf.URL, opts)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to dial: %v", err)
		if resp != nil {
			result.Message += fmt.Sprintf(" (Status: %d)", resp.StatusCode)
			code := resp.StatusCode
			result.StatusCode = &code
		}
		return result, nil
	}
	defer conn.Close(websocket.StatusInternalError, "monitor closing connection")

	// If we got here, connection was established
	conn.Close(websocket.StatusNormalClosure, "check successful")

	latency := int(time.Since(start).Milliseconds())
	result.LatencyMs = &latency
	result.Status = models.StatusUp
	result.Message = "WebSocket connection successful"
	code := 101 // Switching Protocols
	if resp != nil {
		code = resp.StatusCode
	}
	result.StatusCode = &code

	return result, nil
}

// Validate ensures the monitor config is valid for a WebSocket Checker
func (c *WebSocketChecker) Validate(config json.RawMessage) error {
	var conf models.WebSocketMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid websocket config: %w", err)
	}

	if conf.URL == "" {
		return fmt.Errorf("url is required for websocket monitors")
	}

	return nil
}
