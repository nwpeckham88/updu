package channels

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/updu/updu/internal/models"
)

// EmailChannel implements notification via SMTP.
type EmailChannel struct{}

// NewEmailChannel creates a new EmailChannel.
func NewEmailChannel() *EmailChannel {
	return &EmailChannel{}
}

func (c *EmailChannel) Type() string {
	return "email"
}

func (c *EmailChannel) Send(ctx context.Context, monitor *models.Monitor, result *models.CheckResult, config map[string]any) error {
	host, _ := config["host"].(string)
	portVal, _ := config["port"]
	user, _ := config["user"].(string)
	pass, _ := config["pass"].(string)
	from, _ := config["from"].(string)
	to, _ := config["to"].(string)

	var port float64
	switch v := portVal.(type) {
	case float64:
		port = v
	case int:
		port = float64(v)
	}

	if host == "" || port == 0 || from == "" || to == "" {
		return fmt.Errorf("missing email configuration (host, port, from, to)")
	}

	latency := 0
	if result.LatencyMs != nil {
		latency = *result.LatencyMs
	}

	addr := fmt.Sprintf("%s:%d", host, int(port))
	subject := fmt.Sprintf("Subject: [updu] Monitor %s is %s\n", monitor.Name, result.Status)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf("<html><body><h2>Monitor alert</h2><p>Monitor: <b>%s</b></p><p>Status: <b>%s</b></p><p>Message: %s</p><p>Latency: %dms</p></body></html>",
		monitor.Name, result.Status, result.Message, latency)

	msg := []byte(subject + mime + body)

	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	toAddr := strings.Split(to, ",")

	err := smtp.SendMail(addr, auth, from, toAddr, msg)
	if err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}
