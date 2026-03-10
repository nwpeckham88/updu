package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// HTTPSChecker monitors HTTPS endpoints, checking both HTTP response and TLS certificate health.
type HTTPSChecker struct{}

func (c *HTTPSChecker) Type() string { return "https" }

func (c *HTTPSChecker) Validate(config json.RawMessage) error {
	var cfg models.HTTPSMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid HTTPS config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

func (c *HTTPSChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.HTTPSMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	method := cfg.Method
	if method == "" {
		method = "GET"
	}
	warnDays := cfg.WarnDays
	if warnDays == 0 {
		warnDays = 14
	}

	timeout := time.Duration(monitor.TimeoutS) * time.Second
	dialer := &net.Dialer{
		Timeout: timeout,
		Control: SafeDialer(ctx),
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipTLSVerify, // #nosec G402
			},
		},
	}

	var bodyReader io.Reader
	if cfg.Body != "" {
		bodyReader = strings.NewReader(cfg.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, bodyReader)
	if err != nil {
		return failResult(monitor.ID, "creating request: "+err.Error()), nil
	}
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(req)
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
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	result := &models.CheckResult{
		MonitorID:  monitor.ID,
		LatencyMs:  &latency,
		StatusCode: &statusCode,
		CheckedAt:  time.Now(),
	}

	// Evaluate HTTP status
	if statusCode < 200 || statusCode >= 400 {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("HTTP %d", statusCode)
		return result, nil
	}
	if cfg.ExpectedStatus > 0 && statusCode != cfg.ExpectedStatus {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("expected %d, got %d", cfg.ExpectedStatus, statusCode)
		return result, nil
	}

	// Check expected body
	if cfg.ExpectedBody != "" {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			result.Status = models.StatusDown
			result.Message = "reading body: " + err.Error()
			return result, nil
		}
		if !strings.Contains(string(body), cfg.ExpectedBody) {
			result.Status = models.StatusDown
			result.Message = fmt.Sprintf("body missing keyword: %q", cfg.ExpectedBody)
			return result, nil
		}
	}

	// Inspect TLS certificate
	// resp.TLS is populated by Go's TLS stack even with InsecureSkipVerify.
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		remaining := time.Until(cert.NotAfter)
		if remaining <= 0 {
			result.Status = models.StatusDown
			result.Message = fmt.Sprintf("TLS certificate expired on %s", cert.NotAfter.Format("2006-01-02"))
			return result, nil
		}
		daysLeft := int(remaining.Hours() / 24)
		if daysLeft < warnDays {
			result.Status = models.StatusDegraded
			result.Message = fmt.Sprintf("TLS certificate expires in %d day(s)", daysLeft)
			return result, nil
		}
	}

	result.Status = models.StatusUp
	return result, nil
}
