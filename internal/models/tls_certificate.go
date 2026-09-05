package models

import "time"

// TLSCertificate represents TLS certificate details harvested during probe checks.
type TLSCertificate struct {
	ID                   string     `json:"id"`
	Domain               string     `json:"domain"`
	Issuer               string     `json:"issuer"`
	Subject              string     `json:"subject"`
	SANs                 []string   `json:"sans"`
	ValidFrom            time.Time  `json:"valid_from"`
	ValidUntil           time.Time  `json:"valid_until"`
	DaysRemaining        int        `json:"days_remaining"`
	SerialNumber         string     `json:"serial_number,omitempty"`
	SignatureAlgorithm   string     `json:"signature_algorithm,omitempty"`
	OCSPStatus           string     `json:"ocsp_status,omitempty"`
	LastVerifiedAt       time.Time  `json:"last_verified_at"`
	AssociatedEndpointID *string    `json:"associated_endpoint_id,omitempty"`
	AssociatedService    string     `json:"associated_service,omitempty"`
}
