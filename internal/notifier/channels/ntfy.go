package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

func (c *NtfyChannel) Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error {
	urlStr, _ := config["url"].(string)
	if urlStr == "" {
		return fmt.Errorf("ntfy url is required")
	}

	statusEmoji := "✅"
	priority := "default"
	if event.Status == models.StatusDown {
		statusEmoji = "🔴"
		priority = "high"
	} else if event.Status == models.StatusDegraded {
		statusEmoji = "🟡"
		priority = "default"
	}

	title := fmt.Sprintf("%s Monitor %s is %s", statusEmoji, monitor.Name, event.Status)
	message := fmt.Sprintf("Status: %s", event.Status)
	if event.Message != "" {
		message += fmt.Sprintf("\nMessage: %s", event.Message)
	}

	body, _ := json.Marshal(map[string]any{
		"topic":    "",
		"title":    title,
		"message":  message,
		"priority": priorityToInt(priority),
	})

	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating ntfy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Parse basic auth credentials from URL if present
	if parsed, err := url.Parse(urlStr); err == nil && parsed.User != nil {
		password, _ := parsed.User.Password()
		req.SetBasicAuth(parsed.User.Username(), password)
	}

	client := newSafeHTTPClient(10 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending ntfy notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readResponseError("ntfy", resp)
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
