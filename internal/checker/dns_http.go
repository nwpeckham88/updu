package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

type dnsResolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
	LookupCNAME(ctx context.Context, host string) (string, error)
}

var defaultDNSResolver dnsResolver = net.DefaultResolver

// DNSHTTPChecker validates DNS resolution for a hostname and then checks the HTTP endpoint.
// It can detect CDN misrouting and DNS failover issues by verifying the resolved IPs
// match an expected prefix before (and independently of) the HTTP check.
type DNSHTTPChecker struct{}

func (c *DNSHTTPChecker) Type() string { return "dns_http" }

func (c *DNSHTTPChecker) Validate(config json.RawMessage) error {
	var cfg models.DNSHTTPMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid dns_http config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

func (c *DNSHTTPChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.DNSHTTPMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	parsed, err := url.Parse(cfg.URL)
	if err != nil {
		return failResult(monitor.ID, "invalid url: "+err.Error()), nil
	}
	hostname := parsed.Hostname()

	// Step 1: DNS resolution
	resolvedIPs, err := defaultDNSResolver.LookupHost(ctx, hostname)
	if err != nil {
		zero := 0
		return &models.CheckResult{
			MonitorID: monitor.ID,
			Status:    models.StatusDown,
			LatencyMs: &zero,
			Message:   "DNS lookup failed: " + err.Error(),
			CheckedAt: time.Now(),
		}, nil
	}

	// Step 2: optional IP prefix validation (does not short-circuit the HTTP check)
	ipMismatch := false
	if cfg.ExpectedIPPrefix != "" {
		matched := false
		for _, ip := range resolvedIPs {
			if strings.HasPrefix(ip, cfg.ExpectedIPPrefix) {
				matched = true
				break
			}
		}
		if !matched {
			ipMismatch = true
		}
	}

	// Step 2b: optional CNAME validation (does not short-circuit the HTTP check)
	resolvedCNAME := ""
	cnameLookupErr := ""
	cnameMismatch := false
	if cfg.ExpectedCNAME != "" {
		resolvedCNAME, err = defaultDNSResolver.LookupCNAME(ctx, hostname)
		if err != nil {
			cnameLookupErr = err.Error()
			cnameMismatch = true
		} else if normalizeDNSName(resolvedCNAME) != normalizeDNSName(cfg.ExpectedCNAME) {
			cnameMismatch = true
		}
	}

	// Step 3: HTTP request
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

	req, err := http.NewRequestWithContext(ctx, "GET", cfg.URL, nil)
	if err != nil {
		return failResult(monitor.ID, "creating request: "+err.Error()), nil
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := int(time.Since(start).Milliseconds())

	metadataPayload := map[string]any{
		"hostname":     hostname,
		"resolved_ips": resolvedIPs,
	}
	if resolvedCNAME != "" {
		metadataPayload["resolved_cname"] = resolvedCNAME
	}
	if cnameLookupErr != "" {
		metadataPayload["cname_lookup_error"] = cnameLookupErr
	}
	metadata, _ := json.Marshal(metadataPayload)

	if err != nil {
		return &models.CheckResult{
			MonitorID: monitor.ID,
			Status:    models.StatusDown,
			LatencyMs: &latency,
			Message:   "HTTP request failed: " + err.Error(),
			Metadata:  metadata,
			CheckedAt: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	result := &models.CheckResult{
		MonitorID:  monitor.ID,
		LatencyMs:  &latency,
		StatusCode: &statusCode,
		Metadata:   metadata,
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

	// HTTP passed — degrade if DNS resolved to unexpected IPs or CNAME.
	degradedReasons := make([]string, 0, 2)
	if ipMismatch {
		degradedReasons = append(degradedReasons, fmt.Sprintf(
			"resolved IPs %v do not match expected prefix %q",
			resolvedIPs,
			cfg.ExpectedIPPrefix,
		))
	}
	if cnameMismatch {
		if cnameLookupErr != "" {
			degradedReasons = append(degradedReasons, fmt.Sprintf(
				"CNAME lookup failed for expected %q: %s",
				cfg.ExpectedCNAME,
				cnameLookupErr,
			))
		} else {
			degradedReasons = append(degradedReasons, fmt.Sprintf(
				"resolved CNAME %q does not match expected %q",
				resolvedCNAME,
				cfg.ExpectedCNAME,
			))
		}
	}
	if len(degradedReasons) > 0 {
		result.Status = models.StatusDegraded
		result.Message = "HTTP ok but " + strings.Join(degradedReasons, "; ")
		return result, nil
	}

	result.Status = models.StatusUp
	return result, nil
}

func normalizeDNSName(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}
