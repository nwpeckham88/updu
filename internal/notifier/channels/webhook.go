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

// WebhookChannel implements notification via generic HTTP webhooks.
type WebhookChannel struct {
	client *http.Client
}

// NewWebhookChannel creates a new WebhookChannel.
func NewWebhookChannel() *WebhookChannel {
	return &WebhookChannel{
		client: newSafeHTTPClient(10 * time.Second),
	}
}

func (c *WebhookChannel) Type() string {
	return "webhook"
}

func (c *WebhookChannel) Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("missing or invalid webhook URL")
	}

	method, _ := config["method"].(string)
	if method == "" {
		method = "POST"
	}

	payload := map[string]any{
		"monitor_id":   monitor.ID,
		"monitor_name": monitor.Name,
		"status":       event.Status,
		"message":      event.Message,
		"timestamp":    event.CreatedAt.Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "updu-webhook/1.0")

	// Add custom headers if provided
	if headers, ok := config["headers"].(map[string]any); ok {
		for k, v := range headers {
			if s, ok := v.(string); ok {
				req.Header.Set(k, s)
			}
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}
