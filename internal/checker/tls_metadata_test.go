package checker

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

func parseTLSMetadata(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	if len(raw) == 0 {
		t.Fatal("expected metadata to be populated")
	}

	metadata := make(map[string]any)
	if err := json.Unmarshal(raw, &metadata); err != nil {
		t.Fatalf("failed to decode metadata: %v", err)
	}
	return metadata
}

func createCertificateChainForMetadataTest(t *testing.T) (*x509.Certificate, []*x509.Certificate) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate ca key: %v", err)
	}

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}

	now := time.Now().UTC()
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(100),
		Subject: pkix.Name{
			CommonName:   "Acme Root",
			Organization: []string{"Acme Root CA"},
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create ca cert: %v", err)
	}

	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse ca cert: %v", err)
	}

	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(0x1A2B3C),
		Subject: pkix.Name{
			CommonName:   "secure.example.test",
			Organization: []string{"Acme Co"},
		},
		NotBefore:             now.Add(-2 * time.Hour),
		NotAfter:              now.Add(30 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"secure.example.test", "api.example.test"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		SignatureAlgorithm:    x509.SHA256WithRSA,
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caTemplate, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf cert: %v", err)
	}

	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("parse leaf cert: %v", err)
	}

	return leafCert, []*x509.Certificate{leafCert, caCert}
}

func createSelfSignedCertificateForMetadataTest(t *testing.T, commonName string) *x509.Certificate {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano()),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Acme Co"},
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{commonName},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}

	return cert
}

func TestBuildCertificateMetadata_ExpandedFields(t *testing.T) {
	leaf, chain := createCertificateChainForMetadataTest(t)

	metadata := parseTLSMetadata(t, buildCertificateMetadata(leaf, 14, certificateMetadataOptions{
		PeerCertificates: chain,
		VerificationMode: "verified",
		Verified:         true,
	}))

	if got, ok := metadata["cert_serial_number"].(string); !ok || !strings.EqualFold(got, "1a2b3c") {
		t.Fatalf("expected serial number metadata, got %#v", metadata["cert_serial_number"])
	}
	if got, ok := metadata["cert_fingerprint_sha256"].(string); !ok || len(got) != 64 {
		t.Fatalf("expected sha256 fingerprint metadata, got %#v", metadata["cert_fingerprint_sha256"])
	}
	if got, ok := metadata["cert_signature_algorithm"].(string); !ok || got == "" {
		t.Fatalf("expected signature algorithm metadata, got %#v", metadata["cert_signature_algorithm"])
	}
	if got, ok := metadata["cert_public_key_algorithm"].(string); !ok || got != "RSA" {
		t.Fatalf("expected RSA public key algorithm, got %#v", metadata["cert_public_key_algorithm"])
	}
	if got, ok := metadata["cert_public_key_bits"].(float64); !ok || got != 2048 {
		t.Fatalf("expected 2048-bit public key, got %#v", metadata["cert_public_key_bits"])
	}
	if got, ok := metadata["cert_tls_verification_mode"].(string); !ok || got != "verified" {
		t.Fatalf("expected verified tls mode, got %#v", metadata["cert_tls_verification_mode"])
	}
	if got, ok := metadata["cert_tls_verified"].(bool); !ok || !got {
		t.Fatalf("expected tls verified metadata, got %#v", metadata["cert_tls_verified"])
	}
	if got, ok := metadata["cert_chain_length"].(float64); !ok || got != 2 {
		t.Fatalf("expected two certificates in chain, got %#v", metadata["cert_chain_length"])
	}

	dnsNames, ok := metadata["cert_dns_names"].([]any)
	if !ok || len(dnsNames) != 2 {
		t.Fatalf("expected dns names metadata, got %#v", metadata["cert_dns_names"])
	}
	ipAddresses, ok := metadata["cert_ip_addresses"].([]any)
	if !ok || len(ipAddresses) != 1 || ipAddresses[0] != "127.0.0.1" {
		t.Fatalf("expected ip address metadata, got %#v", metadata["cert_ip_addresses"])
	}
	chainSummary, ok := metadata["cert_chain_summary"].([]any)
	if !ok || len(chainSummary) != 2 {
		t.Fatalf("expected chain summary metadata, got %#v", metadata["cert_chain_summary"])
	}
}

func TestBuildCertificateMetadata_CapsChainSummary(t *testing.T) {
	leaf := createSelfSignedCertificateForMetadataTest(t, "leaf.example.test")
	peers := []*x509.Certificate{
		leaf,
		createSelfSignedCertificateForMetadataTest(t, "intermediate-a.example.test"),
		createSelfSignedCertificateForMetadataTest(t, "intermediate-b.example.test"),
		createSelfSignedCertificateForMetadataTest(t, "root.example.test"),
	}

	metadata := parseTLSMetadata(t, buildCertificateMetadata(leaf, 14, certificateMetadataOptions{
		PeerCertificates: peers,
		VerificationMode: "skipped",
		Verified:         false,
	}))

	if got, ok := metadata["cert_chain_length"].(float64); !ok || got != 4 {
		t.Fatalf("expected full chain length metadata, got %#v", metadata["cert_chain_length"])
	}
	chainSummary, ok := metadata["cert_chain_summary"].([]any)
	if !ok || len(chainSummary) != 3 {
		t.Fatalf("expected capped chain summary metadata, got %#v", metadata["cert_chain_summary"])
	}
}

func TestExtractTLSCertificate(t *testing.T) {
	leaf := createSelfSignedCertificateForMetadataTest(t, "vault.example.test")
	leaf.DNSNames = []string{"vault.example.test", "vault-alt.example.test"}
	rawMeta := buildCertificateMetadata(leaf, 14, certificateMetadataOptions{
		PeerCertificates: []*x509.Certificate{leaf},
		VerificationMode: "verified",
		Verified:         true,
	})

	epID := "ep-123"
	cert := ExtractTLSCertificate(rawMeta, &epID, "vault.example.test")
	if cert == nil {
		t.Fatalf("expected extracted cert, got nil")
	}
	if cert.Domain != "vault.example.test" {
		t.Errorf("expected domain vault.example.test, got %q", cert.Domain)
	}
	if cert.ID != "cert-vault.example.test" {
		t.Errorf("expected cert-vault.example.test, got %q", cert.ID)
	}
	if len(cert.SANs) != 2 {
		t.Errorf("expected 2 SANs, got %d", len(cert.SANs))
	}
	if cert.AssociatedEndpointID == nil || *cert.AssociatedEndpointID != "ep-123" {
		t.Errorf("expected associated endpoint ep-123, got %v", cert.AssociatedEndpointID)
	}

	// Test nested metadata backwards compatibility
	nestedMeta := []byte(`{"tls":{"domain":"nested.example.test","issuer":"Test CA","subject":"CN=nested.example.test","sans":["nested.example.test"],"days_remaining":45}}`)
	nestedCert := ExtractTLSCertificate(nestedMeta, nil, "")
	if nestedCert == nil {
		t.Fatalf("expected extracted nested cert, got nil")
	}
	if nestedCert.Domain != "nested.example.test" {
		t.Errorf("expected domain nested.example.test, got %q", nestedCert.Domain)
	}

	// Empty metadata
	if ExtractTLSCertificate(nil, nil, "") != nil {
		t.Errorf("expected nil for empty metadata")
	}
}
