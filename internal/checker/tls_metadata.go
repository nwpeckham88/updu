package checker

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"time"
)

const certificateChainSummaryLimit = 3

type certificateMetadataOptions struct {
	PeerCertificates []*x509.Certificate
	VerificationMode string
	Verified         bool
}

func buildCertificateMetadata(cert *x509.Certificate, warnDays int, options certificateMetadataOptions) json.RawMessage {
	peerCertificates := options.PeerCertificates
	if len(peerCertificates) == 0 {
		peerCertificates = []*x509.Certificate{cert}
	}

	fingerprint := sha256.Sum256(cert.Raw)

	metadata := map[string]any{
		"cert_not_before": cert.NotBefore.UTC().Format(time.RFC3339),
		"cert_not_after":  cert.NotAfter.UTC().Format(time.RFC3339),
		// cert_days_remaining can be negative when the certificate has already expired.
		"cert_days_remaining":        int(time.Until(cert.NotAfter).Hours() / 24),
		"cert_subject":               cert.Subject.String(),
		"cert_issuer":                cert.Issuer.String(),
		"cert_warn_days":             warnDays,
		"cert_serial_number":         cert.SerialNumber.Text(16),
		"cert_fingerprint_sha256":    hex.EncodeToString(fingerprint[:]),
		"cert_signature_algorithm":   cert.SignatureAlgorithm.String(),
		"cert_public_key_algorithm":  cert.PublicKeyAlgorithm.String(),
		"cert_public_key_bits":       certificatePublicKeyBits(cert),
		"cert_tls_verification_mode": options.VerificationMode,
		"cert_tls_verified":          options.Verified,
		"cert_chain_length":          len(peerCertificates),
		"cert_chain_summary":         summarizeCertificateChain(peerCertificates),
	}

	if len(cert.DNSNames) > 0 {
		metadata["cert_dns_names"] = cert.DNSNames
	}

	if len(cert.IPAddresses) > 0 {
		ipAddresses := make([]string, 0, len(cert.IPAddresses))
		for _, ipAddress := range cert.IPAddresses {
			ipAddresses = append(ipAddresses, ipAddress.String())
		}
		metadata["cert_ip_addresses"] = ipAddresses
	}

	rawMetadata, err := json.Marshal(metadata)
	if err != nil {
		return json.RawMessage(`{}`)
	}

	return rawMetadata
}

func certificatePublicKeyBits(cert *x509.Certificate) int {
	switch publicKey := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		return publicKey.N.BitLen()
	case *ecdsa.PublicKey:
		return publicKey.Params().BitSize
	case ed25519.PublicKey:
		return len(publicKey) * 8
	default:
		return 0
	}
}

func summarizeCertificateChain(certificates []*x509.Certificate) []string {
	if len(certificates) == 0 {
		return nil
	}

	limit := len(certificates)
	if limit > certificateChainSummaryLimit {
		limit = certificateChainSummaryLimit
	}

	summary := make([]string, 0, limit)
	for _, certificate := range certificates[:limit] {
		summary = append(summary, certificate.Subject.String())
	}

	return summary
}
