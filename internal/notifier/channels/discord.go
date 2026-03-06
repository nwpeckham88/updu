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

// DiscordChannel implements notification via Discord webhooks.
type DiscordChannel struct {
	client *http.Client
}

// NewDiscordChannel creates a new DiscordChannel.
func NewDiscordChannel() *DiscordChannel {
	return &DiscordChannel{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *DiscordChannel) Type() string {
	return "discord"
}

func (c *DiscordChannel) Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("missing or invalid Discord webhook URL")
	}

	color := 0x00ff00 // Green
	if event.Status != models.StatusUp {
		color = 0xff0000 // Red
	}

	payload := map[string]any{
		"embeds": []map[string]any{
			{
				"title":       fmt.Sprintf("Monitor %s is %s", monitor.Name, event.Status),
				"description": event.Message,
				"color":       color,
				"fields": []map[string]any{
					{
						"name":   "Monitor",
						"value":  monitor.Name,
						"inline": true,
					},
					{
						"name":   "Status",
						"value":  string(event.Status),
						"inline": true,
					},
				},
				"timestamp": event.CreatedAt.Format(time.RFC3339),
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
		return fmt.Errorf("sending discord notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}
