package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// AppriseChannel implements notifications via an Apprise CLI or HTTP microservice.
type AppriseChannel struct {
	client *http.Client
}

// NewAppriseChannel creates a new AppriseChannel.
func NewAppriseChannel() *AppriseChannel {
	return &AppriseChannel{
		client: newSafeHTTPClient(10 * time.Second),
	}
}

func (c *AppriseChannel) Type() string {
	return "apprise"
}

func (c *AppriseChannel) Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error {
	appriseURL, ok := config["url"].(string)
	if !ok || appriseURL == "" {
		return fmt.Errorf("missing or invalid Apprise url")
	}

	appriseType := "info"
	if event.Status == models.StatusDown {
		appriseType = "failure"
	} else if event.Status == models.StatusDegraded {
		appriseType = "warning"
	} else if event.Status == models.StatusUp {
		appriseType = "success"
	}

	title := fmt.Sprintf("[%s] %s is %s", strings.ToUpper(string(event.Status)), monitor.Name, event.Status)
	bodyText := fmt.Sprintf("Monitor: %s\nStatus: %s\nTimestamp: %s", monitor.Name, event.Status, event.CreatedAt.Format(time.RFC3339))
	if event.Message != "" {
		bodyText += fmt.Sprintf("\nDetails: %s", event.Message)
	}

	payload := map[string]any{
		"title": title,
		"body":  bodyText,
		"type":  appriseType,
	}

	if tag, ok := config["tag"].(string); ok && tag != "" {
		payload["tag"] = tag
	}

	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling Apprise payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, appriseURL, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return fmt.Errorf("creating Apprise request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending Apprise alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("apprise returned status: %d", resp.StatusCode)
	}

	return nil
}
