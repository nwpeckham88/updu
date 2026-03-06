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

// SlackChannel implements notification via Slack webhooks.
type SlackChannel struct {
	client *http.Client
}

// NewSlackChannel creates a new SlackChannel.
func NewSlackChannel() *SlackChannel {
	return &SlackChannel{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *SlackChannel) Type() string {
	return "slack"
}

func (c *SlackChannel) Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("missing or invalid Slack webhook URL")
	}

	color := "#36a64f" // Green
	if event.Status != models.StatusUp {
		color = "#ff0000" // Red
	}

	payload := map[string]any{
		"attachments": []map[string]any{
			{
				"title": fmt.Sprintf("Monitor %s is %s", monitor.Name, event.Status),
				"text":  event.Message,
				"color": color,
				"fields": []map[string]any{
					{"title": "Monitor", "value": monitor.Name, "short": true},
					{"title": "Status", "value": string(event.Status), "short": true},
				},
				"ts": time.Now().Unix(),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}
