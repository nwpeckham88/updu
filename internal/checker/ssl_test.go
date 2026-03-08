package checker

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func createTestTLSServer(t *testing.T, validFor time.Duration) *httptest.Server {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	notBefore := time.Now().Add(-1 * time.Hour)
	notAfter := time.Now().Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		t.Fatalf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}

	server := httptest.NewUnstartedServer(nil)
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server.StartTLS()
	return server
}

func parseURL(urlStr string) (host string, port int) {
	urlStr = strings.TrimPrefix(urlStr, "https://")
	h, pStr, err := net.SplitHostPort(urlStr)
	if err != nil {
		return urlStr, 443
	}
	fmt.Sscanf(pStr, "%d", &port)
	return h, port
}

func TestSSLChecker_Validate(t *testing.T) {
	c := &SSLChecker{}

	if err := c.Validate([]byte(`{}`)); err == nil {
		t.Error("expected error for empty config")
	}

	if err := c.Validate([]byte(`{"host": ""}`)); err == nil {
		t.Error("expected error for missing host")
	}

	if err := c.Validate([]byte(`{"host": "example.com"}`)); err != nil {
		t.Error("expected valid config")
	}
}

func TestSSLChecker_Check(t *testing.T) {
	// 1. Valid Cert (> 7 days)
	srvValid := createTestTLSServer(t, 30*24*time.Hour)
	defer srvValid.Close()

	// 2. Expiring Cert (< 7 days)
	srvExpiring := createTestTLSServer(t, 5*24*time.Hour)
	defer srvExpiring.Close()

	// 3. Expired Cert (in the past)
	srvExpired := createTestTLSServer(t, -24*time.Hour)
	defer srvExpired.Close()

	c := &SSLChecker{}

	tests := []struct {
		name           string
		cfg            models.SSLMonitorConfig
		expectedStatus models.MonitorStatus
	}{
		{
			name: "valid cert",
			cfg: func() models.SSLMonitorConfig {
				h, p := parseURL(srvValid.URL)
				return models.SSLMonitorConfig{Host: h, Port: p, DaysBeforeExpiry: 7}
			}(),
			expectedStatus: models.StatusUp,
		},
		{
			name: "expiring cert",
			cfg: func() models.SSLMonitorConfig {
				h, p := parseURL(srvExpiring.URL)
				return models.SSLMonitorConfig{Host: h, Port: p, DaysBeforeExpiry: 7}
			}(),
			expectedStatus: models.StatusDegraded,
		},
		{
			name: "expired cert",
			cfg: func() models.SSLMonitorConfig {
				h, p := parseURL(srvExpired.URL)
				return models.SSLMonitorConfig{Host: h, Port: p, DaysBeforeExpiry: 7}
			}(),
			expectedStatus: models.StatusDown,
		},
		{
			name:           "unreachable host",
			cfg:            models.SSLMonitorConfig{Host: "127.0.0.1", Port: 65535},
			expectedStatus: models.StatusDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBytes, _ := json.Marshal(tt.cfg)
			monitor := &models.Monitor{
				ID:       "ssl-test",
				Config:   cfgBytes,
				TimeoutS: 5,
			}

			result, err := c.Check(context.WithValue(context.Background(), AllowLocalhostKey, true), monitor)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s. message: %s", tt.expectedStatus, result.Status, result.Message)
			}
		})
	}
}
