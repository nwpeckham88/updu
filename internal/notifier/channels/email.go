package channels

import (
	"context"
	"crypto/tls"
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

// Send delivers an email notification.
//
// Supported config keys:
//
//	host     string  – SMTP server hostname (required)
//	port     number  – SMTP port (required)
//	user     string  – SMTP username (optional)
//	pass     string  – SMTP password (optional)
//	from     string  – Sender address (required)
//	to       string  – Recipient address(es), comma-separated (required)
//	tls_mode string  – One of:
//	                   "opportunistic"    (default) use STARTTLS if offered
//	                   "starttls_required" fail if STARTTLS is not available
//	                   "tls"              implicit TLS from the start (port 465)
func (c *EmailChannel) Send(ctx context.Context, monitor *models.Monitor, event *models.Event, config map[string]any) error {
	host, _ := config["host"].(string)
	portVal, _ := config["port"]
	user, _ := config["user"].(string)
	pass, _ := config["pass"].(string)
	from, _ := config["from"].(string)
	to, _ := config["to"].(string)
	tlsMode, _ := config["tls_mode"].(string)
	if tlsMode == "" {
		tlsMode = "opportunistic"
	}

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

	// SSRF protection: reject private / loopback hosts before connecting.
	if err := checkHostSSRF(ctx, host); err != nil {
		return fmt.Errorf("smtp host rejected: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", host, int(port))

	subject := fmt.Sprintf("Subject: [updu] Monitor %s is %s\n", monitor.Name, event.Status)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf("<html><body><h2>Monitor alert</h2><p>Monitor: <b>%s</b></p><p>Status: <b>%s</b></p><p>Message: %s</p></body></html>",
		monitor.Name, event.Status, event.Message)
	msg := []byte(subject + mime + body)

	var smtpAuth smtp.Auth
	if user != "" {
		smtpAuth = smtp.PlainAuth("", user, pass, host)
	}

	toAddrs := strings.Split(to, ",")

	switch tlsMode {
	case "tls":
		return sendWithImplicitTLS(addr, host, smtpAuth, from, toAddrs, msg)
	case "starttls_required":
		return sendWithRequiredSTARTTLS(addr, host, smtpAuth, from, toAddrs, msg)
	default: // "opportunistic"
		if err := smtp.SendMail(addr, smtpAuth, from, toAddrs, msg); err != nil {
			return fmt.Errorf("sending email: %w", err)
		}
		return nil
	}
}

// sendWithImplicitTLS connects with TLS from the start (port 465 / SMTPS).
func sendWithImplicitTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	return smtpSend(client, from, to, msg)
}

// sendWithRequiredSTARTTLS dials plaintext, then upgrades via STARTTLS,
// returning an error if the server does not advertise STARTTLS support.
func sendWithRequiredSTARTTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer client.Close()

	ok, _ := client.Extension("STARTTLS")
	if !ok {
		return fmt.Errorf("smtp server %q does not support STARTTLS (required by tls_mode=starttls_required)", host)
	}

	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if err := client.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("smtp STARTTLS: %w", err)
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	return smtpSend(client, from, to, msg)
}

func smtpSend(client *smtp.Client, from string, to []string, msg []byte) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	for _, addr := range to {
		if err := client.Rcpt(strings.TrimSpace(addr)); err != nil {
			return fmt.Errorf("smtp RCPT TO %q: %w", addr, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}
