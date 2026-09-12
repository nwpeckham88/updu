package checker

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
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

func certificateChainForMetadata(state *tls.ConnectionState) []*x509.Certificate {
	if state == nil {
		return nil
	}
	if len(state.VerifiedChains) > 0 {
		return state.VerifiedChains[0]
	}
	return state.PeerCertificates
}

func tlsVerificationMode(skipTLSVerify bool) string {
	if skipTLSVerify {
		return "skipped"
	}
	return "verified"
}

// ExtractTLSCertificate extracts a TLSCertificate model from monitor/probe metadata.
func ExtractTLSCertificate(rawMetadata []byte, endpointID *string, fallbackHost string) *models.TLSCertificate {
	if len(rawMetadata) == 0 {
		return nil
	}

	var flat struct {
		Subject            string   `json:"cert_subject"`
		Issuer             string   `json:"cert_issuer"`
		NotBefore          string   `json:"cert_not_before"`
		NotAfter           string   `json:"cert_not_after"`
		DaysRemaining      int      `json:"cert_days_remaining"`
		SerialNumber       string   `json:"cert_serial_number"`
		SignatureAlgorithm string   `json:"cert_signature_algorithm"`
		DNSNames           []string `json:"cert_dns_names"`
		TLSVerified        bool     `json:"cert_tls_verified"`
	}

	if err := json.Unmarshal(rawMetadata, &flat); err == nil && (flat.Subject != "" || flat.Issuer != "" || len(flat.DNSNames) > 0) {
		validFrom, _ := time.Parse(time.RFC3339, flat.NotBefore)
		validUntil, _ := time.Parse(time.RFC3339, flat.NotAfter)

		domain := fallbackHost
		if len(flat.DNSNames) > 0 && flat.DNSNames[0] != "" {
			domain = flat.DNSNames[0]
		} else if flat.Subject != "" {
			if idx := strings.Index(flat.Subject, "CN="); idx != -1 {
				cn := flat.Subject[idx+3:]
				if comma := strings.Index(cn, ","); comma != -1 {
					cn = cn[:comma]
				}
				if cn != "" {
					domain = cn
				}
			} else {
				domain = flat.Subject
			}
		}

		if domain == "" {
			return nil
		}

		cleanDomain := strings.TrimPrefix(domain, "*.")
		certID := fmt.Sprintf("cert-%s", cleanDomain)

		ocsp := "verified"
		if !flat.TLSVerified {
			ocsp = "unverified"
		}

		return &models.TLSCertificate{
			ID:                   certID,
			Domain:               cleanDomain,
			Issuer:               flat.Issuer,
			Subject:              flat.Subject,
			SANs:                 flat.DNSNames,
			ValidFrom:            validFrom,
			ValidUntil:           validUntil,
			DaysRemaining:        flat.DaysRemaining,
			SerialNumber:         flat.SerialNumber,
			SignatureAlgorithm:   flat.SignatureAlgorithm,
			OCSPStatus:           ocsp,
			LastVerifiedAt:       time.Now(),
			AssociatedEndpointID: endpointID,
		}
	}

	// Also support nested { "tls": { ... } } structure
	var nested struct {
		TLS *struct {
			Domain        string    `json:"domain"`
			Issuer        string    `json:"issuer"`
			Subject       string    `json:"subject"`
			SANs          []string  `json:"sans"`
			ValidFrom     time.Time `json:"valid_from"`
			ValidUntil    time.Time `json:"valid_until"`
			DaysRemaining int       `json:"days_remaining"`
		} `json:"tls"`
	}
	if err := json.Unmarshal(rawMetadata, &nested); err == nil && nested.TLS != nil && nested.TLS.Domain != "" {
		cleanDomain := strings.TrimPrefix(nested.TLS.Domain, "*.")
		return &models.TLSCertificate{
			ID:                   fmt.Sprintf("cert-%s", cleanDomain),
			Domain:               cleanDomain,
			Issuer:               nested.TLS.Issuer,
			Subject:              nested.TLS.Subject,
			SANs:                 nested.TLS.SANs,
			ValidFrom:            nested.TLS.ValidFrom,
			ValidUntil:           nested.TLS.ValidUntil,
			DaysRemaining:        nested.TLS.DaysRemaining,
			LastVerifiedAt:       time.Now(),
			AssociatedEndpointID: endpointID,
		}
	}

	return nil
}
