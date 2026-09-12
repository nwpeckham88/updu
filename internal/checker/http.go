package checker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
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

	parsedURL, _ := url.Parse(cfg.URL)
	var serverName string
	if parsedURL != nil {
		serverName = parsedURL.Hostname()
	}

	var capturedTLS *tls.ConnectionState
	var capturedVerified bool

	tlsConfig := &tls.Config{
		ServerName: serverName,
	}

	if parsedURL != nil && strings.EqualFold(parsedURL.Scheme, "https") {
		// Use InsecureSkipVerify so TLS handshake completes to capture peer certificates even if expired/untrusted
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyConnection = func(cs tls.ConnectionState) error {
			copyState := cs
			capturedTLS = &copyState
			if cfg.SkipTLSVerify {
				capturedVerified = false
				return nil
			}
			opts := x509.VerifyOptions{
				Intermediates: x509.NewCertPool(),
			}
			if net.ParseIP(serverName) == nil && serverName != "" {
				opts.DNSName = serverName
			}
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			chains, err := cs.PeerCertificates[0].Verify(opts)
			if err != nil {
				return err
			}
			copyState.VerifiedChains = chains
			capturedTLS = &copyState
			capturedVerified = true
			return nil
		}
	} else {
		tlsConfig.InsecureSkipVerify = cfg.SkipTLSVerify
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:     dialer.DialContext,
			TLSClientConfig: tlsConfig,
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

	var (
		dnsStart, dnsDone   time.Time
		connStart, connDone time.Time
		tlsStart, tlsDone   time.Time
		reqStart, firstByte time.Time
		resolvedIP          string
		connectedAddr       string
		failingHop          string
		capturedTLSVersion  string
		capturedTLSCipher   string
	)

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			dnsDone = time.Now()
			if len(info.Addrs) > 0 {
				resolvedIP = info.Addrs[0].String()
			}
			if info.Err != nil && failingHop == "" {
				failingHop = "dns"
			}
		},
		ConnectStart: func(network, addr string) {
			connStart = time.Now()
			connectedAddr = addr
		},
		ConnectDone: func(network, addr string, err error) {
			connDone = time.Now()
			if err != nil && failingHop == "" {
				failingHop = "tcp"
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tlsDone = time.Now()
			if state.Version != 0 {
				capturedTLSVersion = tlsVersionString(state.Version)
				capturedTLSCipher = tls.CipherSuiteName(state.CipherSuite)
			}
			if err != nil && failingHop == "" {
				failingHop = "tls"
			}
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			reqStart = time.Now()
			if info.Err != nil && failingHop == "" {
				failingHop = "tcp"
			}
		},
		GotFirstResponseByte: func() {
			firstByte = time.Now()
		},
	}

	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), method, cfg.URL, bodyReader)
	if err != nil {
		return failResult(monitor.ID, "creating request: "+err.Error()), nil
	}

	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := int(time.Since(start).Milliseconds())

	var transferStart, transferDone time.Time

	if err != nil {
		if failingHop == "" {
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "dns") || strings.Contains(errStr, "no such host") {
				failingHop = "dns"
			} else if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "handshake") || strings.Contains(errStr, "tls") {
				failingHop = "tls"
			} else if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "timeout") || strings.Contains(errStr, "i/o timeout") || strings.Contains(errStr, "dial") {
				failingHop = "tcp"
			} else {
				failingHop = "network"
			}
		}

		hopTrace := computeHopTrace(
			dnsStart, dnsDone, resolvedIP,
			connStart, connDone, connectedAddr,
			tlsStart, tlsDone, capturedTLSVersion, capturedTLSCipher,
			reqStart, firstByte, transferStart, transferDone, failingHop,
		)

		res := &models.CheckResult{
			MonitorID: monitor.ID,
			Status:    models.StatusDown,
			LatencyMs: &latency,
			Message:   err.Error(),
			CheckedAt: time.Now(),
		}
		var certMeta json.RawMessage
		if capturedTLS != nil && len(capturedTLS.PeerCertificates) > 0 {
			warnDays := cfg.WarnDays
			if warnDays == 0 {
				warnDays = 14
			}
			cert := capturedTLS.PeerCertificates[0]
			certMeta = buildCertificateMetadata(cert, warnDays, certificateMetadataOptions{
				PeerCertificates: certificateChainForMetadata(capturedTLS),
				VerificationMode: tlsVerificationMode(cfg.SkipTLSVerify),
				Verified:         capturedVerified,
			})
		}
		res.Metadata = mergeHopTraceMetadata(certMeta, hopTrace)
		return res, nil
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
	if cfg.ExpectedStatus > 0 {
		if statusCode != cfg.ExpectedStatus {
			result.Status = models.StatusDown
			result.Message = fmt.Sprintf("expected %d, got %d", cfg.ExpectedStatus, statusCode)
		} else {
			result.Status = models.StatusUp
		}
	} else if statusCode < 200 || statusCode >= 400 {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("HTTP %d", statusCode)
	} else {
		result.Status = models.StatusUp
	}

	// Check expected body keyword
	transferStart = time.Now()
	if cfg.ExpectedBody != "" && result.Status == models.StatusUp {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
		transferDone = time.Now()
		if err != nil {
			result.Status = models.StatusDown
			result.Message = "reading body: " + err.Error()
		} else if !strings.Contains(string(body), cfg.ExpectedBody) {
			result.Status = models.StatusDown
			result.Message = fmt.Sprintf("body missing keyword: %q", cfg.ExpectedBody)
		}
	} else {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8192))
		transferDone = time.Now()
	}

	// Inspect TLS certificate if connection was over TLS
	tlsState := resp.TLS
	if tlsState == nil {
		tlsState = capturedTLS
	}
	var certMeta json.RawMessage
	if tlsState != nil && len(tlsState.PeerCertificates) > 0 {
		if tlsState.Version != 0 {
			capturedTLSVersion = tlsVersionString(tlsState.Version)
			capturedTLSCipher = tls.CipherSuiteName(tlsState.CipherSuite)
		}
		warnDays := cfg.WarnDays
		if warnDays == 0 {
			warnDays = 14
		}
		cert := tlsState.PeerCertificates[0]
		certMeta = buildCertificateMetadata(cert, warnDays, certificateMetadataOptions{
			PeerCertificates: certificateChainForMetadata(tlsState),
			VerificationMode: tlsVerificationMode(cfg.SkipTLSVerify),
			Verified:         !cfg.SkipTLSVerify && (len(tlsState.VerifiedChains) > 0 || capturedVerified),
		})
		if result.Status == models.StatusUp {
			remaining := time.Until(cert.NotAfter)
			if remaining <= 0 {
				result.Status = models.StatusDown
				result.Message = fmt.Sprintf("TLS certificate expired on %s", cert.NotAfter.Format("2006-01-02"))
			} else {
				daysLeft := int(remaining.Hours() / 24)
				if daysLeft < warnDays {
					result.Status = models.StatusDegraded
					result.Message = fmt.Sprintf("TLS certificate expires in %d day(s)", daysLeft)
				}
			}
		}
	}

	if result.Status != models.StatusUp {
		if statusCode == 502 || statusCode == 503 || statusCode == 504 || (statusCode >= 520 && statusCode <= 526) {
			failingHop = "ingress"
		} else if statusCode >= 400 {
			failingHop = "app"
		} else if strings.Contains(result.Message, "TLS certificate") {
			failingHop = "tls"
		} else {
			failingHop = "app"
		}
	}

	hopTrace := computeHopTrace(
		dnsStart, dnsDone, resolvedIP,
		connStart, connDone, connectedAddr,
		tlsStart, tlsDone, capturedTLSVersion, capturedTLSCipher,
		reqStart, firstByte, transferStart, transferDone, failingHop,
	)
	result.Metadata = mergeHopTraceMetadata(certMeta, hopTrace)

	return result, nil
}

func tlsVersionString(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("0x%04x", v)
	}
}

func computeHopTrace(
	dnsStart, dnsDone time.Time, resolvedIP string,
	connStart, connDone time.Time, connectedAddr string,
	tlsStart, tlsDone time.Time, tlsVer, tlsCipher string,
	reqStart, firstByte time.Time,
	transferStart, transferDone time.Time,
	failingHop string,
) *models.HopTrace {
	trace := &models.HopTrace{
		ResolvedIP:    resolvedIP,
		ConnectedAddr: connectedAddr,
		TLSVersion:    tlsVer,
		TLSCipher:     tlsCipher,
		FailingHop:    failingHop,
	}
	if !dnsStart.IsZero() && !dnsDone.IsZero() {
		d := int(dnsDone.Sub(dnsStart).Milliseconds())
		trace.DNSLookupMs = &d
	}
	if !connStart.IsZero() && !connDone.IsZero() {
		d := int(connDone.Sub(connStart).Milliseconds())
		trace.TCPConnectMs = &d
	}
	if !tlsStart.IsZero() && !tlsDone.IsZero() {
		d := int(tlsDone.Sub(tlsStart).Milliseconds())
		trace.TLSHandshakeMs = &d
	}
	if !firstByte.IsZero() {
		ref := reqStart
		if ref.IsZero() {
			ref = connDone
		}
		if !ref.IsZero() {
			d := int(firstByte.Sub(ref).Milliseconds())
			trace.TTFBMs = &d
		}
	}
	if !transferStart.IsZero() && !transferDone.IsZero() {
		d := int(transferDone.Sub(transferStart).Milliseconds())
		trace.TransferMs = &d
	}
	return trace
}

func mergeHopTraceMetadata(existingMeta json.RawMessage, trace *models.HopTrace) json.RawMessage {
	metaMap := make(map[string]any)
	if len(existingMeta) > 0 {
		_ = json.Unmarshal(existingMeta, &metaMap)
	}
	if trace != nil {
		traceBytes, err := json.Marshal(trace)
		if err == nil {
			var traceObj map[string]any
			if err := json.Unmarshal(traceBytes, &traceObj); err == nil {
				metaMap["trace"] = traceObj
			}
		}
	}
	out, err := json.Marshal(metaMap)
	if err != nil {
		return existingMeta
	}
	return out
}

func failResult(monitorID, message string) *models.CheckResult {
	return &models.CheckResult{
		MonitorID: monitorID,
		Status:    models.StatusDown,
		Message:   message,
		CheckedAt: time.Now(),
	}
}
