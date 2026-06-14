package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

var (
	// Commonly used WHOIS expiry date prefixes
	expiryPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)Registry Expiry Date:\s*(.+)`),
		regexp.MustCompile(`(?i)Registrar Registration Expiration Date:\s*(.+)`),
		regexp.MustCompile(`(?i)Expiration Date:\s*(.+)`),
		regexp.MustCompile(`(?i)Expiry Date:\s*(.+)`),
		regexp.MustCompile(`(?i)expire date:\s*(.+)`),
		regexp.MustCompile(`(?i)valid-date:\s*(.+)`), // .ru
		regexp.MustCompile(`(?i)paid-till:\s*(.+)`),  // .ru, .su
		regexp.MustCompile(`(?i)Expiration Time:\s*(.+)`),
		regexp.MustCompile(`(?i)Expires on:\s*(.+)`),
		regexp.MustCompile(`(?i)Expiration:\s*(.+)`),
	}

	// whoisServerPattern extracts "whois:" field from IANA response
	whoisServerPattern = regexp.MustCompile(`(?i)whois:\s*([^\s]+)`)

	// Common date formats found in WHOIS
	dateFormats = []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02-Jan-2006",
		"02.01.2006",
		"02/01/2006",
		"2006.01.02",
		"2006/01/02",
		"Jan 02 2006",
		"02 Jan 2006",
		"2006-01-02 15:04:05 UTC",
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05-07:00",
	}
)

type WhoisChecker struct{}

func (c *WhoisChecker) Type() string { return "whois" }

func (c *WhoisChecker) Validate(config json.RawMessage) error {
	var cfg models.WhoisMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid whois config: %w", err)
	}
	if strings.TrimSpace(cfg.Domain) == "" {
		return fmt.Errorf("domain is required")
	}
	return nil
}

func (c *WhoisChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.WhoisMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	domain := strings.TrimSpace(cfg.Domain)
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.Split(domain, "/")[0] // Remove paths
	domain = strings.Split(domain, ":")[0] // Remove port

	start := time.Now()
	timeout := time.Duration(monitor.TimeoutS) * time.Second

	// 1. Resolve authoritative WHOIS server from IANA
	tld := getTLD(domain)
	if tld == "" {
		return failResult(monitor.ID, "invalid domain, no TLD found"), nil
	}

	ianaCtx, cancelIANA := context.WithTimeout(ctx, timeout/2)
	defer cancelIANA()
	ianaResp, err := queryWhois(ianaCtx, "whois.iana.org", tld)
	if err != nil {
		return failResult(monitor.ID, "iana whois failed: "+err.Error()), nil
	}

	authServerMatch := whoisServerPattern.FindStringSubmatch(ianaResp)
	if len(authServerMatch) < 2 {
		// Fallback to a common guess if IANA fails to provide one
		authServerMatch = []string{"", tld + ".whois-servers.net"}
	}
	authServer := strings.TrimSpace(authServerMatch[1])

	// 2. Query the authoritative server
	authCtx, cancelAuth := context.WithTimeout(ctx, timeout/2)
	defer cancelAuth()
	
	// Some whois servers need special prefixes. Verisign (com/net) works best with "=" prefix to exact match, but let's just send the domain first.
	query := domain
	if authServer == "whois.verisign-grs.com" {
		query = "=" + domain
	} else if authServer == "whois.denic.ru" {
		query = domain
	}

	authResp, err := queryWhois(authCtx, authServer, query)
	if err != nil {
		return failResult(monitor.ID, "authoritative whois failed: "+err.Error()), nil
	}

	latency := int(time.Since(start).Milliseconds())

	// 3. Parse expiration date
	expiryStr := extractExpiryDate(authResp)
	if expiryStr == "" {
		return &models.CheckResult{
			MonitorID: monitor.ID,
			Status:    models.StatusDown,
			LatencyMs: &latency,
			Message:   "could not parse expiration date from whois response",
			CheckedAt: time.Now(),
		}, nil
	}

	expiryDate, err := parseDate(expiryStr)
	if err != nil {
		return &models.CheckResult{
			MonitorID: monitor.ID,
			Status:    models.StatusDown,
			LatencyMs: &latency,
			Message:   fmt.Sprintf("could not parse date format: %q", expiryStr),
			CheckedAt: time.Now(),
		}, nil
	}

	daysBeforeExpiry := cfg.DaysBeforeExpiry
	if daysBeforeExpiry <= 0 {
		daysBeforeExpiry = 14
	}

	daysRemaining := int(time.Until(expiryDate).Hours() / 24)

	meta, _ := json.Marshal(map[string]any{
		"expiry_date":    expiryDate.Format(time.RFC3339),
		"days_remaining": daysRemaining,
		"whois_server":   authServer,
	})

	if daysRemaining < 0 {
		return &models.CheckResult{
			MonitorID: monitor.ID,
			Status:    models.StatusDown,
			LatencyMs: &latency,
			Message:   fmt.Sprintf("domain expired %d days ago", -daysRemaining),
			Metadata:  meta,
			CheckedAt: time.Now(),
		}, nil
	}

	if daysRemaining <= daysBeforeExpiry {
		return &models.CheckResult{
			MonitorID: monitor.ID,
			Status:    models.StatusDegraded,
			LatencyMs: &latency,
			Message:   fmt.Sprintf("domain expiring in %d days", daysRemaining),
			Metadata:  meta,
			CheckedAt: time.Now(),
		}, nil
	}

	return &models.CheckResult{
		MonitorID: monitor.ID,
		Status:    models.StatusUp,
		LatencyMs: &latency,
		Message:   fmt.Sprintf("domain valid for %d days", daysRemaining),
		Metadata:  meta,
		CheckedAt: time.Now(),
	}, nil
}

func getTLD(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

func queryWhois(ctx context.Context, server, query string) (string, error) {
	dialer := &net.Dialer{Control: SafeDialer(ctx)}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(server, "43"))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(query + "\r\n")); err != nil {
		return "", err
	}

	buf, err := io.ReadAll(io.LimitReader(conn, 1024*1024)) // 1MB limit
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func extractExpiryDate(whoisData string) string {
	for _, pattern := range expiryPatterns {
		matches := pattern.FindStringSubmatch(whoisData)
		if len(matches) > 1 {
			// Some formats might have comments or extra spaces
			val := strings.TrimSpace(matches[1])
			// Example: "2025-01-01T00:00:00Z"
			// Split by space and take first part if it looks like just a date without time?
			// But time is ok. Let's just return the trimmed string.
			return val
		}
	}
	return ""
}

func parseDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	// some whois servers return extra junk after the date, like "2025-01-01 (YYYY-MM-DD)"
	if idx := strings.Index(dateStr, " ("); idx != -1 {
		dateStr = dateStr[:idx]
	}
	
	// Normalize common issues
	dateStr = strings.ReplaceAll(dateStr, "  ", " ") // remove double spaces

	for _, format := range dateFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown date format")
}
