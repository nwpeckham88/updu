package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/updu/updu/internal/models"
)

// NtfyChannel implements notification via ntfy.sh or self-hosted ntfy.
type NtfyChannel struct{}

// NewNtfyChannel creates a new NtfyChannel.
func NewNtfyChannel() *NtfyChannel {
	return &NtfyChannel{}
}

func (c *NtfyChannel) Type() string {
	return "ntfy"
}

func (c *NtfyChannel) Send(ctx context.Context, monitor *models.Monitor, result *models.CheckResult, config map[string]any) error {
	url, _ := config["url"].(string)
	if url == "" {
		return fmt.Errorf("ntfy url is required")
	}

	statusEmoji := "✅"
	priority := "default"
	if result.Status == models.StatusDown {
		statusEmoji = "🔴"
		priority = "high"
	} else if result.Status == models.StatusDegraded {
		statusEmoji = "🟡"
		priority = "default"
	}

	latency := 0
	if result.LatencyMs != nil {
		latency = *result.LatencyMs
	}

	title := fmt.Sprintf("%s Monitor %s is %s", statusEmoji, monitor.Name, result.Status)
	message := fmt.Sprintf("Status: %s\nLatency: %dms", result.Status, latency)
	if result.Message != "" {
		message += fmt.Sprintf("\nMessage: %s", result.Message)
	}

	body, _ := json.Marshal(map[string]any{
		"topic":    "",
		"title":    title,
		"message":  message,
		"priority": priorityToInt(priority),
	})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating ntfy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending ntfy notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ntfy returned status %d", resp.StatusCode)
	}

	return nil
}

func priorityToInt(p string) int {
	switch p {
	case "min":
		return 1
	case "low":
		return 2
	case "default":
		return 3
	case "high":
		return 4
	case "max", "urgent":
		return 5
	default:
		return 3
	}
}
