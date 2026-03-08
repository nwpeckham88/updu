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
	"syscall"
	"time"

	"github.com/updu/updu/internal/models"
)

// HTTPChecker monitors HTTP/HTTPS endpoints.
type HTTPChecker struct{}

func (c *HTTPChecker) Type() string { return "http" }

func (c *HTTPChecker) Validate(config json.RawMessage) error {
	var cfg models.HTTPMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid HTTP config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

// SafeDialer returns a Control function for net.Dialer that blocks private/loopback IPs.
func SafeDialer(ctx context.Context) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}

		ip := net.ParseIP(host)
		if ip == nil {
			return nil
		}

		if isBlocked(ctx, ip) {
			return fmt.Errorf("connection to %s is blocked (SSRF protection)", ip)
		}
		return nil
	}
}

func isBlocked(ctx context.Context, ip net.IP) bool {
	if ip.IsLoopback() {
		// Allow loopback if specifically allowed in context (e.g. for testing)
		if allow, _ := ctx.Value(AllowLocalhostKey).(bool); allow {
			return false
		}
		return true
	}

	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Block RFC1918 private ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
	if ip.IsPrivate() {
		if allow, _ := ctx.Value(AllowLocalhostKey).(bool); allow {
			return false
		}
		return true
	}

	// Block IPv4 169.254.169.254 (AWS/GCP/Azure metadata)
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}

	return false
}

// CheckHostSSRF resolves a hostname and checks all resulting IPs against the SSRF blocklist.
// Returns an error if any resolved IP is blocked.
func CheckHostSSRF(ctx context.Context, host string) error {
	// If it's already an IP, check directly
	if ip := net.ParseIP(host); ip != nil {
		if isBlocked(ctx, ip) {
			return fmt.Errorf("connection to %s is blocked (SSRF protection)", ip)
		}
		return nil
	}

	// Resolve hostname and check all IPs
	ips, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return err
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil && isBlocked(ctx, ip) {
			return fmt.Errorf("connection to %s (%s) is blocked (SSRF protection)", host, ip)
		}
	}
	return nil
}

func (c *HTTPChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.HTTPMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	method := cfg.Method
	if method == "" {
		method = "GET"
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
		// Don't follow redirects automatically for status code checking
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
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

	// Check expected status code
	expectedStatus := cfg.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = 200
	}

	if statusCode < 200 || statusCode >= 400 {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("HTTP %d", statusCode)
	} else if cfg.ExpectedStatus > 0 && statusCode != cfg.ExpectedStatus {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("expected %d, got %d", cfg.ExpectedStatus, statusCode)
	} else {
		result.Status = models.StatusUp
	}

	// Check expected body keyword
	if cfg.ExpectedBody != "" && result.Status == models.StatusUp {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
		if err != nil {
			result.Status = models.StatusDown
			result.Message = "reading body: " + err.Error()
		} else if !strings.Contains(string(body), cfg.ExpectedBody) {
			result.Status = models.StatusDown
			result.Message = fmt.Sprintf("body missing keyword: %q", cfg.ExpectedBody)
		}
	}

	return result, nil
}

func failResult(monitorID, message string) *models.CheckResult {
	return &models.CheckResult{
		MonitorID: monitorID,
		Status:    models.StatusDown,
		Message:   message,
		CheckedAt: time.Now(),
	}
}
