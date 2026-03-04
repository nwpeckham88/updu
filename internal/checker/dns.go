package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// DNSChecker monitors DNS resolution.
type DNSChecker struct {
	resolver Resolver
}

func (c *DNSChecker) Type() string { return "dns" }

func (c *DNSChecker) Validate(config json.RawMessage) error {
	var cfg models.DNSMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid DNS config: %w", err)
	}
	if cfg.Host == "" {
		return fmt.Errorf("host is required")
	}
	return nil
}

func (c *DNSChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.DNSMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	resolver := c.resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	if cfg.Resolver != "" {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: time.Duration(monitor.TimeoutS) * time.Second}
				resolverAddr := cfg.Resolver
				if !strings.Contains(resolverAddr, ":") {
					resolverAddr += ":53"
				}
				return d.DialContext(ctx, "udp", resolverAddr)
			},
		}
	}

	timeout := time.Duration(monitor.TimeoutS) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	recordType := cfg.RecordType
	if recordType == "" {
		recordType = "A"
	}

	start := time.Now()
	var answers []string
	var err error

	switch strings.ToUpper(recordType) {
	case "A":
		var addrs []string
		addrs, err = resolver.LookupHost(ctx, cfg.Host)
		answers = addrs
	case "AAAA":
		var ips []net.IP
		ips, err = resolver.LookupIP(ctx, "ip6", cfg.Host)
		for _, ip := range ips {
			answers = append(answers, ip.String())
		}
	case "CNAME":
		var cname string
		cname, err = resolver.LookupCNAME(ctx, cfg.Host)
		if cname != "" {
			answers = []string{cname}
		}
	case "MX":
		var mxs []*net.MX
		mxs, err = resolver.LookupMX(ctx, cfg.Host)
		for _, mx := range mxs {
			answers = append(answers, mx.Host)
		}
	case "TXT":
		answers, err = resolver.LookupTXT(ctx, cfg.Host)
	case "NS":
		var nss []*net.NS
		nss, err = resolver.LookupNS(ctx, cfg.Host)
		for _, ns := range nss {
			answers = append(answers, ns.Host)
		}
	default:
		return failResult(monitor.ID, "unsupported record type: "+recordType), nil
	}

	latency := int(time.Since(start).Milliseconds())

	result := &models.CheckResult{
		MonitorID: monitor.ID,
		LatencyMs: &latency,
		CheckedAt: time.Now(),
	}

	if err != nil {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("DNS lookup failed: %s", err.Error())
		return result, nil
	}

	if len(answers) == 0 {
		result.Status = models.StatusDown
		result.Message = "no records found"
		return result, nil
	}

	// Check expected answer if configured
	if cfg.Expected != "" {
		found := false
		for _, a := range answers {
			if strings.Contains(a, cfg.Expected) {
				found = true
				break
			}
		}
		if !found {
			result.Status = models.StatusDown
			result.Message = fmt.Sprintf("expected %q not in: %s", cfg.Expected, strings.Join(answers, ", "))
			return result, nil
		}
	}

	result.Status = models.StatusUp
	result.Message = strings.Join(answers, ", ")

	metadata, _ := json.Marshal(map[string]any{
		"record_type": recordType,
		"answers":     answers,
	})
	result.Metadata = metadata

	return result, nil
}
