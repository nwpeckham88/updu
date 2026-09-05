package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// MatrixChannel implements notifications via Matrix Client-Server API.
type MatrixChannel struct {
	client *http.Client
}

// NewMatrixChannel creates a new MatrixChannel.
func NewMatrixChannel() *MatrixChannel {
	return &MatrixChannel{
		client: newSafeHTTPClient(10 * time.Second),
	}
}

func (c *MatrixChannel) Type() string {
	return "matrix"
}

func (c *MatrixChannel) Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error {
	homeserver, ok := config["homeserver_url"].(string)
	if !ok || homeserver == "" {
		return fmt.Errorf("missing or invalid Matrix homeserver_url")
	}
	homeserver = strings.TrimRight(homeserver, "/")

	roomID, ok := config["room_id"].(string)
	if !ok || roomID == "" {
		return fmt.Errorf("missing or invalid Matrix room_id")
	}

	accessToken, ok := config["access_token"].(string)
	if !ok || accessToken == "" {
		return fmt.Errorf("missing or invalid Matrix access_token")
	}

	emoji := "🟢"
	if event.Status == models.StatusDown {
		emoji = "🔴"
	} else if event.Status == models.StatusDegraded {
		emoji = "🟡"
	}

	plainText := fmt.Sprintf("[%s] %s is %s: %s", emoji, monitor.Name, event.Status, event.Message)
	htmlText := fmt.Sprintf(
		"<b>%s updu Alert</b><br/><b>Monitor:</b> %s<br/><b>Status:</b> %s<br/><b>Time:</b> %s",
		emoji,
		html.EscapeString(monitor.Name),
		html.EscapeString(string(event.Status)),
		html.EscapeString(event.CreatedAt.Format(time.RFC3339)),
	)
	if event.Message != "" {
		htmlText += fmt.Sprintf("<br/><b>Details:</b> %s", html.EscapeString(event.Message))
	}

	txnID := fmt.Sprintf("updu-%d", time.Now().UnixNano())
	endpointURL := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s", homeserver, url.PathEscape(roomID), txnID)

	payload := map[string]any{
		"msgtype":        "m.text",
		"body":           plainText,
		"format":         "org.matrix.custom.html",
		"formatted_body": htmlText,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling Matrix payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpointURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("creating Matrix request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending Matrix alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("matrix API returned status: %d", resp.StatusCode)
	}

	return nil
}
