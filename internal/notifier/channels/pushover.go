package channels

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// PushoverChannel implements notifications via Pushover API.
type PushoverChannel struct {
	client *http.Client
}

// NewPushoverChannel creates a new PushoverChannel.
func NewPushoverChannel() *PushoverChannel {
	return &PushoverChannel{
		client: newSafeHTTPClient(10 * time.Second),
	}
}

func (c *PushoverChannel) Type() string {
	return "pushover"
}

func (c *PushoverChannel) Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error {
	userKey, ok := config["user_key"].(string)
	if !ok || userKey == "" {
		return fmt.Errorf("missing or invalid Pushover user_key")
	}

	token, ok := config["token"].(string)
	if !ok || token == "" {
		token, _ = config["api_token"].(string)
	}
	if token == "" {
		return fmt.Errorf("missing or invalid Pushover api token")
	}

	priority := "0"
	if event.Status == models.StatusDown {
		priority = "1" // High priority
	} else if event.Status == models.StatusUp {
		priority = "0" // Normal priority
	}

	title := fmt.Sprintf("[%s] %s is %s", strings.ToUpper(string(event.Status)), monitor.Name, event.Status)
	message := fmt.Sprintf("Monitor: %s\nStatus: %s\nTime: %s", monitor.Name, event.Status, event.CreatedAt.Format(time.RFC3339))
	if event.Message != "" {
		message += fmt.Sprintf("\nDetails: %s", event.Message)
	}

	form := url.Values{}
	form.Set("token", token)
	form.Set("user", userKey)
	form.Set("title", title)
	form.Set("message", message)
	form.Set("priority", priority)

	sound, _ := config["sound"].(string)
	if sound != "" {
		form.Set("sound", sound)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.pushover.net/1/messages.json", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("creating Pushover request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending Pushover notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pushover API returned status: %d", resp.StatusCode)
	}

	return nil
}
