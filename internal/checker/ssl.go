package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/updu/updu/internal/models"
)

// SSLChecker monitors SSL certificate expiration.
type SSLChecker struct{}

func (c *SSLChecker) Type() string { return "ssl" }

func (c *SSLChecker) Validate(config json.RawMessage) error {
	var cfg models.SSLMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid SSL config: %w", err)
	}
	if cfg.Host == "" {
		return fmt.Errorf("host is required")
	}
	return nil
}

func (c *SSLChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.SSLMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	// SSRF protection: check host before connecting
	if err := CheckHostSSRF(ctx, cfg.Host); err != nil {
		return failResult(monitor.ID, err.Error()), nil
	}

	port := cfg.Port
	if port == 0 {
		port = 443
	}

	daysBeforeExpiry := cfg.DaysBeforeExpiry
	if daysBeforeExpiry == 0 {
		daysBeforeExpiry = 7
	}

	// Dial with timeout
	timeout := time.Duration(monitor.TimeoutS) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	start := time.Now()

	dialer := &net.Dialer{Timeout: timeout}
	// We mainly want to check the certificate expiration, so we skip verify
	// to ensure we can read the dates even if it's self-signed or invalid CA.
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(port)), &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: true, // #nosec G402
	})

	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		return &models.CheckResult{
			MonitorID: monitor.ID,
			Status:    models.StatusDown,
			LatencyMs: &latency,
			Message:   err.Error(),
			CheckedAt: time.Now(),
		}, nil
	}
	defer conn.Close()

	// Connection successful, check certificate expiration
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return &models.CheckResult{
			MonitorID: monitor.ID,
			Status:    models.StatusDown,
			LatencyMs: &latency,
			Message:   "no peer certificates found",
			CheckedAt: time.Now(),
		}, nil
	}

	cert := state.PeerCertificates[0]
	timeRemaining := time.Until(cert.NotAfter)

	result := &models.CheckResult{
		MonitorID: monitor.ID,
		LatencyMs: &latency,
		Metadata: buildCertificateMetadata(cert, daysBeforeExpiry, certificateMetadataOptions{
			PeerCertificates: state.PeerCertificates,
			VerificationMode: "skipped",
			Verified:         false,
		}),
		CheckedAt: time.Now(),
	}

	if timeRemaining <= 0 {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("certificate expired on %s", cert.NotAfter.Format(time.RFC3339))
	} else if timeRemaining < time.Duration(daysBeforeExpiry)*24*time.Hour {
		result.Status = models.StatusDegraded
		result.Message = fmt.Sprintf("certificate expires soon (%d days)", int(timeRemaining.Hours()/24))
	} else {
		result.Status = models.StatusUp
		result.Message = fmt.Sprintf("valid for %d more days", int(timeRemaining.Hours()/24))
	}

	return result, nil
}
