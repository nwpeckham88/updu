package channels

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// AllowLocalhostKey is a context key that, when set to true, bypasses SSRF
// protection (loopback / private IPs). Only used in tests.
type contextKey string

const AllowLocalhostKey contextKey = "allow_localhost"

// newSafeHTTPClient returns an *http.Client whose dialer blocks connections to
// loopback, link-local, and RFC-1918 addresses (SSRF protection). It honours
// the AllowLocalhostKey context value so tests using httptest.Server still work.
func newSafeHTTPClient(timeout time.Duration) *http.Client {
	base := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		// DialContext is called with the per-request context, which lets us
		// capture it in the Control closure for the AllowLocalhostKey check.
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := *base // shallow copy; safe because all fields are value types or immutable refs
			d.Control = ssrfControl(ctx)
			return d.DialContext(ctx, network, addr)
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// ssrfControl returns a Control function that rejects private/loopback IPs
// unless AllowLocalhostKey is set in ctx.
func ssrfControl(ctx context.Context) func(network, address string, c syscall.RawConn) error {
	allow, _ := ctx.Value(AllowLocalhostKey).(bool)
	return func(network, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return nil
		}
		if allow {
			return nil
		}
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
			return fmt.Errorf("connection to %s blocked (SSRF protection)", ip)
		}
		if ip4 := ip.To4(); ip4 != nil && ip4[0] == 169 && ip4[1] == 254 {
			return fmt.Errorf("connection to %s blocked (SSRF protection)", ip)
		}
		return nil
	}
}

// checkHostSSRF resolves a hostname and rejects any private/loopback IPs.
// Respects AllowLocalhostKey from ctx.
func checkHostSSRF(ctx context.Context, host string) error {
	allow, _ := ctx.Value(AllowLocalhostKey).(bool)

	check := func(ipStr string) error {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil
		}
		if allow {
			return nil
		}
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
			return fmt.Errorf("connection to %s blocked (SSRF protection)", ip)
		}
		if ip4 := ip.To4(); ip4 != nil && ip4[0] == 169 && ip4[1] == 254 {
			return fmt.Errorf("connection to %s blocked (SSRF protection)", ip)
		}
		return nil
	}

	if ip := net.ParseIP(host); ip != nil {
		return check(host)
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return err
	}
	for _, a := range addrs {
		if err := check(a); err != nil {
			return err
		}
	}
	return nil
}
