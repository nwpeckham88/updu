package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestHTTPChecker_Complex(t *testing.T) {
	checker := &HTTPChecker{}
	ctx := context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("X-Test") != "Value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "ping" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("pong"))
	}))
	defer ts.Close()

	m := &models.Monitor{
		ID: "http-test",
		Config: json.RawMessage(`{
			"url": "` + ts.URL + `",
			"method": "POST",
			"headers": {"X-Test": "Value"},
			"body": "ping",
			"expected_status": 201,
			"expected_body": "pong"
		}`),
		TimeoutS: 5,
	}

	res, err := checker.Check(ctx, m)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if res.Status != models.StatusUp {
		t.Errorf("expected Up, got %s: %s", res.Status, res.Message)
	}
}

type mockResolver struct {
	lookupHostFunc  func(ctx context.Context, host string) ([]string, error)
	lookupIPFunc    func(ctx context.Context, network, host string) ([]net.IP, error)
	lookupCNAMEFunc func(ctx context.Context, host string) (string, error)
	lookupMXFunc    func(ctx context.Context, host string) ([]*net.MX, error)
	lookupTXTFunc   func(ctx context.Context, host string) ([]string, error)
	lookupNSFunc    func(ctx context.Context, host string) ([]*net.NS, error)
}

func (m *mockResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return m.lookupHostFunc(ctx, host)
}
func (m *mockResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	return m.lookupIPFunc(ctx, network, host)
}
func (m *mockResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	return m.lookupCNAMEFunc(ctx, host)
}
func (m *mockResolver) LookupMX(ctx context.Context, host string) ([]*net.MX, error) {
	return m.lookupMXFunc(ctx, host)
}
func (m *mockResolver) LookupTXT(ctx context.Context, host string) ([]string, error) {
	return m.lookupTXTFunc(ctx, host)
}
func (m *mockResolver) LookupNS(ctx context.Context, host string) ([]*net.NS, error) {
	return m.lookupNSFunc(ctx, host)
}

func TestDNSChecker_Types(t *testing.T) {
	mock := &mockResolver{}
	c := &DNSChecker{resolver: mock}
	ctx := context.Background()

	// Test A record
	mock.lookupHostFunc = func(ctx context.Context, host string) ([]string, error) {
		return []string{"1.2.3.4"}, nil
	}
	m := &models.Monitor{ID: "dns-1", Config: json.RawMessage(`{"host":"example.com", "record_type":"A"}`), TimeoutS: 5}
	res, _ := c.Check(ctx, m)
	if res.Status != models.StatusUp || !strings.Contains(res.Message, "1.2.3.4") {
		t.Errorf("A record failed: %v", res)
	}

	// Test MX record
	mock.lookupMXFunc = func(ctx context.Context, host string) ([]*net.MX, error) {
		return []*net.MX{{Host: "mail.example.com"}}, nil
	}
	m.Config = json.RawMessage(`{"host":"example.com", "record_type":"MX"}`)
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusUp || !strings.Contains(res.Message, "mail.example.com") {
		t.Errorf("MX record failed: %v", res)
	}

	// Test AAAA record
	mock.lookupIPFunc = func(ctx context.Context, network, host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("2001:db8::1")}, nil
	}
	m.Config = json.RawMessage(`{"host":"example.com", "record_type":"AAAA"}`)
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusUp || !strings.Contains(res.Message, "2001:db8::1") {
		t.Errorf("AAAA record failed: %v", res)
	}

	// Test CNAME record
	mock.lookupCNAMEFunc = func(ctx context.Context, host string) (string, error) {
		return "hello.example.com", nil
	}
	m.Config = json.RawMessage(`{"host":"example.com", "record_type":"CNAME"}`)
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusUp || !strings.Contains(res.Message, "hello.example.com") {
		t.Errorf("CNAME record failed: %v", res)
	}

	// Test NS record
	mock.lookupNSFunc = func(ctx context.Context, host string) ([]*net.NS, error) {
		return []*net.NS{{Host: "ns1.example.com"}}, nil
	}
	m.Config = json.RawMessage(`{"host":"example.com", "record_type":"NS"}`)
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusUp || !strings.Contains(res.Message, "ns1.example.com") {
		t.Errorf("NS record failed: %v", res)
	}

	// Test invalid record type
	m.Config = json.RawMessage(`{"host":"example.com", "record_type":"INVALID"}`)
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusDown || !strings.Contains(res.Message, "unsupported") {
		t.Errorf("invalid record type failed: %v", res)
	}

	// Test lookup error
	mock.lookupHostFunc = func(ctx context.Context, host string) ([]string, error) {
		return nil, fmt.Errorf("lookup failed")
	}
	m.Config = json.RawMessage(`{"host":"example.com", "record_type":"A"}`)
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusDown || !strings.Contains(res.Message, "lookup failed") {
		t.Errorf("lookup error failed: %v", res)
	}
}

func TestHTTPChecker_Errors(t *testing.T) {
	checker := &HTTPChecker{}
	ctx := context.Background()

	// 1. Invalid JSON config
	m := &models.Monitor{ID: "h-err-1", Config: json.RawMessage(`{invalid`)}
	res, _ := checker.Check(ctx, m)
	if res.Status != models.StatusDown || !strings.Contains(res.Message, "invalid config") {
		t.Errorf("expected down for invalid config, got %v", res)
	}

	// Invalid URL
	m.Config = json.RawMessage(`{"url":":\/\/invalid"}`)
	res, _ = checker.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("expected down for invalid url, got %v", res)
	}

	// 2. HTTP 404
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	m.Config = json.RawMessage(`{"url":"` + ts.URL + `"}`)
	res, _ = checker.Check(ctx, m)
	if res.Status != models.StatusDown || !strings.Contains(res.Message, "404") {
		t.Errorf("expected down for 404, got %v", res)
	}

	// 3. Status mismatch
	m.Config = json.RawMessage(`{"url":"` + ts.URL + `", "expected_status":200}`)
	res, _ = checker.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("expected down for status mismatch, got %v", res)
	}
}

type mockCommander struct {
	output []byte
	err    error
}

func (m *mockCommander) CombinedOutput(ctx context.Context, name string, arg ...string) ([]byte, error) {
	return m.output, m.err
}

func TestPingChecker_Mock(t *testing.T) {
	mock := &mockCommander{
		output: []byte("64 bytes from 1.2.3.4: icmp_seq=1 ttl=64 time=10.5 ms\n"),
	}
	c := &PingChecker{commander: mock}
	ctx := context.Background()

	m := &models.Monitor{ID: "ping-1", Config: json.RawMessage(`{"host":"1.2.3.4"}`), TimeoutS: 5}
	res, _ := c.Check(ctx, m)

	if res.Status != models.StatusUp {
		t.Errorf("expected Up, got %s: %s", res.Status, res.Message)
	}
	if res.LatencyMs == nil || *res.LatencyMs != 10 {
		t.Errorf("expected 10ms latency, got %v", res.LatencyMs)
	}
}

func TestPingChecker_Errors(t *testing.T) {
	mock := &mockCommander{err: fmt.Errorf("ping: unknown host")}
	c := &PingChecker{commander: mock}
	ctx := context.Background()

	m := &models.Monitor{ID: "p-err-1", Config: json.RawMessage(`{"host":"127.0.0.1"}`), TimeoutS: 1}
	res, _ := c.Check(ctx, m)

	if res.Status != models.StatusDown {
		t.Errorf("expected Down for failed ping, got %s", res.Status)
	}
}

func TestTCPChecker_Real(t *testing.T) {
	c := &TCPChecker{}
	ctx := context.Background()

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
		ln.Close()
	}()

	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)

	m := &models.Monitor{
		ID:       "tcp-1",
		Config:   json.RawMessage(fmt.Sprintf(`{"host":"%s", "port":%d}`, host, port)),
		TimeoutS: 1,
	}

	res, _ := c.Check(ctx, m)
	if res.Status != models.StatusUp {
		t.Errorf("expected Up for local listener, got %s: %s", res.Status, res.Message)
	}

	// 2. Closed port (Connection Refused / Timeout)
	m.Config = json.RawMessage(`{"host":"127.0.0.1", "port":2}`) // Port 2 is rarely used
	res, _ = c.Check(ctx, m)
	if res.Status != models.StatusDown {
		t.Errorf("expected Down for closed port, got %s", res.Status)
	}
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()
	types := reg.Types()
	if len(types) < 4 {
		t.Errorf("expected at least 4 checkers, got %d", len(types))
	}

	foundHttp := false
	for _, typ := range types {
		if typ == "http" {
			foundHttp = true
		}
	}
	if !foundHttp {
		t.Error("http checker not found in registry")
	}

	c := reg.Get("http")
	if c == nil || c.Type() != "http" {
		t.Error("failed to get http checker from registry")
	}

	if reg.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent checker")
	}
}

func TestChecker_Validate(t *testing.T) {
	// HTTP
	hc := &HTTPChecker{}
	if err := hc.Validate(json.RawMessage(`{"url":"http://test"}`)); err != nil {
		t.Errorf("expected valid pure http config, got %v", err)
	}
	if err := hc.Validate(json.RawMessage(`{"url":""}`)); err == nil {
		t.Errorf("expected error for empty url http config")
	}

	// Ping
	pc := &PingChecker{}
	if err := pc.Validate(json.RawMessage(`{"host":"1.2.3.4"}`)); err != nil {
		t.Errorf("expected valid pure ping config, got %v", err)
	}
	if err := pc.Validate(json.RawMessage(`{"host":""}`)); err == nil {
		t.Errorf("expected error for empty host ping config")
	}

	// DNS
	dc := &DNSChecker{}
	if err := dc.Validate(json.RawMessage(`{"host":"example.com", "record_type":"A"}`)); err != nil {
		t.Errorf("expected valid pure dns config, got %v", err)
	}
	if err := dc.Validate(json.RawMessage(`{"host":""}`)); err == nil {
		t.Errorf("expected error for empty host dns config")
	}

	// TCP
	tc := &TCPChecker{}
	if err := tc.Validate(json.RawMessage(`{"host":"127.0.0.1", "port":80}`)); err != nil {
		t.Errorf("expected valid pure tcp config, got %v", err)
	}
	if err := tc.Validate(json.RawMessage(`{"host":"", "port":0}`)); err == nil {
		t.Errorf("expected error for empty host/port tcp config")
	}

	// SSL Validate
	sc := &SSLChecker{}
	if err := sc.Validate(json.RawMessage(`{"host":"example.com", "days_before_expiry": 7}`)); err != nil {
		t.Errorf("expected valid ssl config, got %v", err)
	}
	if err := sc.Validate(json.RawMessage(`{"days_before_expiry": 7}`)); err == nil {
		t.Errorf("expected error for empty host ssl config")
	}

	// Bad JSON coverage for all existing types
	checkersToTestError := []Checker{hc, pc, dc, tc, sc}
	for _, c := range checkersToTestError {
		if err := c.Validate(json.RawMessage(`{bad`)); err == nil {
			t.Errorf("expected err for bad json in '%s' checker", c.Type())
		}
	}
}
