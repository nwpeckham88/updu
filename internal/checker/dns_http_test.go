package checker

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

type stubDNSResolver struct {
	resolvedIPs   []string
	resolvedCNAME string
	cnameErr      error
}

func (s stubDNSResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return append([]string(nil), s.resolvedIPs...), nil
}

func (s stubDNSResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	if s.cnameErr != nil {
		return "", s.cnameErr
	}
	return s.resolvedCNAME, nil
}

func TestDNSHTTPChecker(t *testing.T) {
	c := &DNSHTTPChecker{}
	if c.Type() != "dns_http" {
		t.Fatalf("Type() = %q, want %q", c.Type(), "dns_http")
	}

	if err := c.Validate([]byte(`{"url":"https://example.com/healthz"}`)); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
	if err := c.Validate([]byte(`{}`)); err == nil {
		t.Fatal("expected error for missing url")
	}
	if err := c.Validate([]byte(`{bad`)); err == nil {
		t.Fatal("expected error for bad json")
	}

	server := createHTTPSCheckerServer(t, 30*24*time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	dnsURL := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
	matchingPrefix := "127.0.0.1"
	mismatchingPrefix := "not-a-real-prefix"
	matchingCNAME := "edge-origin.example.net"
	mismatchingCNAME := "not-a-real-cname.example"

	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)
	oldResolver := defaultDNSResolver
	defaultDNSResolver = stubDNSResolver{
		resolvedIPs:   []string{matchingPrefix},
		resolvedCNAME: matchingCNAME + ".",
	}
	t.Cleanup(func() {
		defaultDNSResolver = oldResolver
	})

	t.Run("reports up when dns and http expectations match", func(t *testing.T) {
		monitor := &models.Monitor{
			ID: "dns-http-up",
			Config: json.RawMessage(`{
				"url": "` + dnsURL + `",
				"expected_ip_prefix": "` + matchingPrefix + `",
				"expected_status": 200,
				"expected_body": "ok",
				"skip_tls_verify": true
			}`),
			TimeoutS: 5,
		}

		res, err := c.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusUp {
			t.Fatalf("expected up, got %s (%s)", res.Status, res.Message)
		}
	})

	t.Run("reports degraded when dns resolves to an unexpected prefix", func(t *testing.T) {
		monitor := &models.Monitor{
			ID: "dns-http-degraded",
			Config: json.RawMessage(`{
				"url": "` + dnsURL + `",
				"expected_ip_prefix": "` + mismatchingPrefix + `",
				"expected_status": 200,
				"expected_body": "ok",
				"skip_tls_verify": true
			}`),
			TimeoutS: 5,
		}

		res, err := c.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusDegraded {
			t.Fatalf("expected degraded, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "do not match expected prefix") {
			t.Fatalf("expected ip mismatch message, got %q", res.Message)
		}
	})

	t.Run("reports up when cname matches expected", func(t *testing.T) {
		monitor := &models.Monitor{
			ID: "dns-http-cname-up",
			Config: json.RawMessage(`{
				"url": "` + dnsURL + `",
				"expected_ip_prefix": "` + matchingPrefix + `",
				"expected_cname": "` + matchingCNAME + `",
				"expected_status": 200,
				"expected_body": "ok",
				"skip_tls_verify": true
			}`),
			TimeoutS: 5,
		}

		res, err := c.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusUp {
			t.Fatalf("expected up, got %s (%s)", res.Status, res.Message)
		}
	})

	t.Run("reports degraded when cname does not match expected", func(t *testing.T) {
		monitor := &models.Monitor{
			ID: "dns-http-cname-degraded",
			Config: json.RawMessage(`{
				"url": "` + dnsURL + `",
				"expected_ip_prefix": "` + matchingPrefix + `",
				"expected_cname": "` + mismatchingCNAME + `",
				"expected_status": 200,
				"expected_body": "ok",
				"skip_tls_verify": true
			}`),
			TimeoutS: 5,
		}

		res, err := c.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusDegraded {
			t.Fatalf("expected degraded, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "CNAME") {
			t.Fatalf("expected cname mismatch message, got %q", res.Message)
		}
	})

	t.Run("reports degraded when ip prefix and cname both mismatch", func(t *testing.T) {
		monitor := &models.Monitor{
			ID: "dns-http-both-degraded",
			Config: json.RawMessage(`{
				"url": "` + dnsURL + `",
				"expected_ip_prefix": "` + mismatchingPrefix + `",
				"expected_cname": "` + mismatchingCNAME + `",
				"expected_status": 200,
				"expected_body": "ok",
				"skip_tls_verify": true
			}`),
			TimeoutS: 5,
		}

		res, err := c.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusDegraded {
			t.Fatalf("expected degraded, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "expected prefix") || !strings.Contains(res.Message, "CNAME") {
			t.Fatalf("expected combined mismatch message, got %q", res.Message)
		}
	})

	t.Run("reports down when expected body is missing", func(t *testing.T) {
		monitor := &models.Monitor{
			ID: "dns-http-down",
			Config: json.RawMessage(`{
				"url": "` + dnsURL + `",
				"expected_ip_prefix": "` + matchingPrefix + `",
				"expected_status": 200,
				"expected_body": "missing",
				"skip_tls_verify": true
			}`),
			TimeoutS: 5,
		}

		res, err := c.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusDown {
			t.Fatalf("expected down, got %s (%s)", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "body missing keyword") {
			t.Fatalf("expected body mismatch message, got %q", res.Message)
		}
	})
}
