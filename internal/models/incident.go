package models

import "time"

// IncidentStatus represents the state of an incident.
type IncidentStatus string

const (
	IncidentInvestigating IncidentStatus = "investigating"
	IncidentIdentified    IncidentStatus = "identified"
	IncidentMonitoring    IncidentStatus = "monitoring"
	IncidentResolved      IncidentStatus = "resolved"
)

// Incident represents a service incident.
type Incident struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Status      IncidentStatus `json:"status"`
	Severity    string         `json:"severity"` // critical, major, minor
	MonitorIDs  []string       `json:"monitor_ids,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	CreatedBy   string         `json:"created_by"`
}

// StatusPage represents a public/internal status page.
type StatusPage struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description string            `json:"description,omitempty"`
	Groups      []StatusPageGroup `json:"groups"`
	IsPublic    bool              `json:"is_public"`
	Password    string            `json:"-"` // optional password, never serialized
}

// StatusPageGroup organizes monitors on a status page.
type StatusPageGroup struct {
	Name       string   `json:"name"`
	MonitorIDs []string `json:"monitor_ids"`
}

// NotificationChannel represents a configured notification target.
type NotificationChannel struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Type    string         `json:"type"` // discord, slack, email, ntfy, etc.
	Config  map[string]any `json:"config"`
	Enabled bool           `json:"enabled"`
}

// MaintenanceWindow represents a scheduled maintenance period.
type MaintenanceWindow struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	MonitorIDs []string  `json:"monitor_ids"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	Recurring  *string   `json:"recurring,omitempty"` // daily, weekly, monthly
	CreatedBy  string    `json:"created_by"`
}

// Heartbeat represents a passive "dead man's switch" monitor.
type Heartbeat struct {
	Slug      string     `json:"slug"`
	MonitorID string     `json:"monitor_id"`
	LastPing  *time.Time `json:"last_ping,omitempty"`
	ExpectedS int        `json:"expected_s"` // frequency in seconds
	GraceS    int        `json:"grace_s"`    // grace period in seconds
}
