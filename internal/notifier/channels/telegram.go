package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/updu/updu/internal/models"
)

// TelegramChannel implements notification delivery via Telegram Bot API.
type TelegramChannel struct {
	client *http.Client
}

// NewTelegramChannel creates a new TelegramChannel.
func NewTelegramChannel() *TelegramChannel {
	return &TelegramChannel{
		client: newSafeHTTPClient(10 * time.Second),
	}
}

func (c *TelegramChannel) Type() string {
	return "telegram"
}

func (c *TelegramChannel) Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error {
	botToken, ok := config["bot_token"].(string)
	if !ok || botToken == "" {
		return fmt.Errorf("missing or invalid Telegram bot_token")
	}

	chatID, ok := config["chat_id"].(string)
	if !ok || chatID == "" {
		return fmt.Errorf("missing or invalid Telegram chat_id")
	}

	emoji := "🟢"
	if event.Status == models.StatusDown {
		emoji = "🔴"
	} else if event.Status == models.StatusDegraded {
		emoji = "🟡"
	}

	messageText := fmt.Sprintf(
		"<b>%s updu Alert</b>\n\n<b>Monitor:</b> %s\n<b>Status:</b> %s\n<b>Time:</b> %s",
		emoji,
		html.EscapeString(monitor.Name),
		html.EscapeString(string(event.Status)),
		html.EscapeString(event.CreatedAt.Format(time.RFC3339)),
	)
	if event.Message != "" {
		messageText += fmt.Sprintf("\n<b>Details:</b> %s", html.EscapeString(event.Message))
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       messageText,
		"parse_mode": "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling Telegram payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("creating Telegram request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending Telegram alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}

	return nil
}
