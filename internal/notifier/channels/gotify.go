package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// GotifyChannel implements notification delivery to a Gotify server.
type GotifyChannel struct {
	client *http.Client
}

// NewGotifyChannel creates a new GotifyChannel.
func NewGotifyChannel() *GotifyChannel {
	return &GotifyChannel{
		client: newSafeHTTPClient(10 * time.Second),
	}
}

func (c *GotifyChannel) Type() string {
	return "gotify"
}

func (c *GotifyChannel) Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error {
	baseURL, _ := config["url"].(string)
	if baseURL == "" {
		return fmt.Errorf("gotify server URL is required")
	}

	token, _ := config["token"].(string)
	if token == "" {
		return fmt.Errorf("gotify app token is required")
	}

	messageURL, err := gotifyMessageURL(baseURL)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"title":    fmt.Sprintf("Monitor %s is %s", monitor.Name, event.Status),
		"message":  gotifyMessageBody(monitor, event),
		"priority": gotifyPriority(event.Status),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, messageURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating gotify request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gotify-Key", token)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending gotify notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gotify returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

func gotifyMessageURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("gotify server URL is invalid")
	}

	trimmedPath := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(trimmedPath, "/message") {
		trimmedPath += "/message"
	}
	if trimmedPath == "" {
		trimmedPath = "/message"
	}
	parsed.Path = trimmedPath

	return parsed.String(), nil
}

func gotifyMessageBody(monitor *models.Monitor, event *models.Event) string {
	lines := []string{
		fmt.Sprintf("Monitor: %s", monitor.Name),
		fmt.Sprintf("Status: %s", event.Status),
	}
	if event.Message != "" {
		lines = append(lines, fmt.Sprintf("Message: %s", event.Message))
	}
	return strings.Join(lines, "\n")
}

func gotifyPriority(status models.MonitorStatus) int {
	switch status {
	case models.StatusDown:
		return 8
	case models.StatusDegraded:
		return 5
	default:
		return 2
	}
}