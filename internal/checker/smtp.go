package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/updu/updu/internal/models"
)

// SMTPChecker implements checking for SMTP servers.
type SMTPChecker struct{}

// Type returns the monitor type
func (c *SMTPChecker) Type() string {
	return "smtp"
}

// Check evaluates whether the SMTP server is accessible and responding.
func (c *SMTPChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: monitor.ID,
		CheckedAt: start,
		Status:    models.StatusDown,
	}

	var conf models.SMTPMonitorConfig
	if err := json.Unmarshal(monitor.Config, &conf); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	// SSRF protection: check host before connecting
	if err := CheckHostSSRF(ctx, conf.Host); err != nil {
		result.Message = err.Error()
		return result, nil
	}

	address := fmt.Sprintf("%s:%d", conf.Host, conf.Port)

	// Since net/smtp dial does not easily take a context, we construct a custom dialer
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to connect: %v", err)
		return result, nil
	}
	defer conn.Close()

	// Upgrade connection if TLS is required immediately (like SMTPS on 465)
	if conf.RequireTLS && conf.Port == 465 {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: conf.Host})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			result.Message = fmt.Sprintf("TLS handshake failed: %v", err)
			return result, nil
		}
		conn = tlsConn
	}

	client, err := smtp.NewClient(conn, conf.Host)
	if err != nil {
		result.Message = fmt.Sprintf("SMTP protocol error: %v", err)
		return result, nil
	}
	defer client.Quit()

	// If STARTTLS is required (usually on port 587 or 25)
	if conf.RequireTLS && conf.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: conf.Host}
			if err = client.StartTLS(config); err != nil {
				result.Message = fmt.Sprintf("STARTTLS failed: %v", err)
				return result, nil
			}
		} else {
			result.Message = "SMTP server does not support STARTTLS"
			return result, nil
		}
	}

	// Identify ourselves
	if err := client.Hello("localhost"); err != nil {
		result.Message = fmt.Sprintf("HELO failed: %v", err)
		return result, nil
	}

	latency := int(time.Since(start).Milliseconds())
	result.LatencyMs = &latency
	result.Status = models.StatusUp
	result.Message = "SMTP greeting successful"
	return result, nil
}

// Validate ensures the monitor config is valid for an SMTP Checker
func (c *SMTPChecker) Validate(config json.RawMessage) error {
	var conf models.SMTPMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid smtp config: %w", err)
	}

	if conf.Host == "" {
		return fmt.Errorf("host is required for smtp monitors")
	}

	if conf.Port == 0 {
		return fmt.Errorf("port is required for smtp monitors")
	}

	return nil
}
