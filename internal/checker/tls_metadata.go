package checker

import (
	"crypto/x509"
	"encoding/json"
	"time"
)

func buildCertificateMetadata(cert *x509.Certificate, warnDays int) json.RawMessage {
	metadata, err := json.Marshal(map[string]any{
		"cert_not_before": cert.NotBefore.UTC().Format(time.RFC3339),
		"cert_not_after":  cert.NotAfter.UTC().Format(time.RFC3339),
		// cert_days_remaining can be negative when the certificate has already expired.
		"cert_days_remaining": int(time.Until(cert.NotAfter).Hours() / 24),
		"cert_subject":        cert.Subject.String(),
		"cert_issuer":         cert.Issuer.String(),
		"cert_warn_days":      warnDays,
	})
	if err != nil {
		return json.RawMessage(`{}`)
	}

	return metadata
}
