package checker

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/updu/updu/internal/models"
)

func createHTTPSCheckerServer(t *testing.T, validFor time.Duration, handler http.Handler) *httptest.Server {
	t.Helper()

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

	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	server.StartTLS()
	return server
}

func TestHTTPSChecker(t *testing.T) {
	c := &HTTPSChecker{}
	if c.Type() != "https" {
		t.Fatalf("Type() = %q, want %q", c.Type(), "https")
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

	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("X-Test") != "Value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if string(body) != "ping" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("pong"))
	})

	t.Run("degrades when cert is near expiry but request matches", func(t *testing.T) {
		server := createHTTPSCheckerServer(t, 24*time.Hour, handler)
		defer server.Close()

		monitor := &models.Monitor{
			ID: "https-degraded",
			Config: json.RawMessage(`{
				"url": "` + server.URL + `",
				"method": "POST",
				"headers": {"X-Test": "Value"},
				"body": "ping",
				"expected_status": 201,
				"expected_body": "pong",
				"skip_tls_verify": true,
				"warn_days": 7
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
		if !strings.Contains(res.Message, "expires in") {
			t.Fatalf("expected expiry warning, got %q", res.Message)
		}
	})

	t.Run("fails when tls verification is enforced", func(t *testing.T) {
		server := createHTTPSCheckerServer(t, 30*24*time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		monitor := &models.Monitor{
			ID:       "https-tls-error",
			Config:   json.RawMessage(`{"url":"` + server.URL + `"}`),
			TimeoutS: 5,
		}

		res, err := c.Check(ctx, monitor)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if res.Status != models.StatusDown {
			t.Fatalf("expected down, got %s (%s)", res.Status, res.Message)
		}
	})
}
