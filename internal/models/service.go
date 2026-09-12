package models

import (
	"encoding/json"
	"time"
)

// ServiceType constants
const (
	ServiceTypeWeb      = "web"
	ServiceTypeInfra    = "infra"
	ServiceTypeDatabase = "database"
	ServiceTypeHost     = "host"
	ServiceTypeJob      = "job"
)

// Service represents a unified monitored application or infrastructure resource.
type Service struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Type          string             `json:"type"` // web, infra, database, host, job
	ZoneID        string             `json:"zone_id"`
	Groups        []string           `json:"groups,omitempty"`
	Tags          []string           `json:"tags,omitempty"`
	Enabled       bool               `json:"enabled"`
	MaintenanceID *string            `json:"maintenance_id,omitempty"`
	CreatedBy     string             `json:"created_by"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	Endpoints     []*ServiceEndpoint `json:"endpoints,omitempty"`

	// Transient runtime fields
	Status         MonitorStatus `json:"status,omitempty"`
	Diagnosis      string        `json:"diagnosis,omitempty"`
	LastCheck      *time.Time    `json:"last_check,omitempty"`
	PrimaryLatency *int          `json:"primary_latency_ms,omitempty"`
}

// PrimaryEndpoint returns the primary endpoint of the service, or the first endpoint if none is explicitly primary.
func (s *Service) PrimaryEndpoint() *ServiceEndpoint {
	if len(s.Endpoints) == 0 {
		return nil
	}
	for _, ep := range s.Endpoints {
		if ep.IsPrimary {
			return ep
		}
	}
	return s.Endpoints[0]
}

// ServiceEndpoint represents an individual probe path targeting a Service.
type ServiceEndpoint struct {
	ID          string          `json:"id"`
	ServiceID   string          `json:"service_id"`
	Name        string          `json:"name"`
	ScopeID     string          `json:"scope_id"`     // lan, tailnet, public
	TargetType  string          `json:"target_type"`  // http, tcp, ping, dns, push
	Config      json.RawMessage `json:"config"`
	IntervalS   int             `json:"interval_s"`
	TimeoutS    int             `json:"timeout_s"`
	Retries     int             `json:"retries"`
	IsPrimary   bool            `json:"is_primary"`
	CreatedAt   time.Time       `json:"created_at"`

	// Transient runtime fields
	Status         MonitorStatus   `json:"status,omitempty"`
	LastLatency    *int            `json:"last_latency_ms,omitempty"`
	LastStatusCode *int            `json:"last_status_code,omitempty"`
	LastMessage    string          `json:"last_message,omitempty"`
	LastMetadata   json.RawMessage `json:"last_metadata,omitempty"`
	LastCheckedAt  *time.Time      `json:"last_checked_at,omitempty"`
}

// EndpointCheck represents the result of a probe against a specific ServiceEndpoint.
type EndpointCheck struct {
	ID          int64           `json:"id,omitempty"`
	ServiceID   string          `json:"service_id"`
	EndpointID  string          `json:"endpoint_id"`
	NodeID      string          `json:"node_id"`
	Status      MonitorStatus   `json:"status"`
	LatencyMs   *int            `json:"latency_ms,omitempty"`
	StatusCode  *int            `json:"status_code,omitempty"`
	Message     string          `json:"message,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CheckedAt   time.Time       `json:"checked_at"`
}
