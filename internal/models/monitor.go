package models

import (
	"encoding/json"
	"time"
)

// MonitorStatus represents the current state of a monitor.
type MonitorStatus string

const (
	StatusUp       MonitorStatus = "up"
	StatusDown     MonitorStatus = "down"
	StatusDegraded MonitorStatus = "degraded"
	StatusPending  MonitorStatus = "pending"
	StatusPaused   MonitorStatus = "paused"
)

// Monitor represents a monitoring target.
type Monitor struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config"`
	GroupName string          `json:"group_name,omitempty"`
	Tags      []string        `json:"tags,omitempty"`
	IntervalS int             `json:"interval_s"`
	TimeoutS  int             `json:"timeout_s"`
	Retries   int             `json:"retries"`
	Enabled   bool            `json:"enabled"`
	ParentID  *string         `json:"parent_id,omitempty"`
	CreatedBy string          `json:"created_by"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`

	// Transient fields (not stored, populated at runtime)
	Status      MonitorStatus `json:"status,omitempty"`
	LastCheck   *time.Time    `json:"last_check,omitempty"`
	LastLatency *int          `json:"last_latency_ms,omitempty"`
}

// CheckResult represents the outcome of a single check.
type CheckResult struct {
	ID         int64           `json:"id,omitempty"`
	MonitorID  string          `json:"monitor_id"`
	Status     MonitorStatus   `json:"status"`
	LatencyMs  *int            `json:"latency_ms,omitempty"`
	StatusCode *int            `json:"status_code,omitempty"`
	Message    string          `json:"message,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CheckedAt  time.Time       `json:"checked_at"`
}

// CheckAggregate represents compacted historical data.
type CheckAggregate struct {
	MonitorID   string    `json:"monitor_id"`
	PeriodStart time.Time `json:"period_start"`
	Resolution  string    `json:"resolution"` // "5m", "1h", "1d"
	TotalChecks int       `json:"total_checks"`
	UpCount     int       `json:"up_count"`
	DownCount   int       `json:"down_count"`
	AvgLatency  *float64  `json:"avg_latency,omitempty"`
	MinLatency  *int      `json:"min_latency,omitempty"`
	MaxLatency  *int      `json:"max_latency,omitempty"`
	UptimePct   *float64  `json:"uptime_pct,omitempty"`
}

// HTTPMonitorConfig holds config for HTTP/HTTPS monitors.
type HTTPMonitorConfig struct {
	URL            string            `json:"url"`
	Method         string            `json:"method,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           string            `json:"body,omitempty"`
	ExpectedStatus int               `json:"expected_status,omitempty"`
	ExpectedBody   string            `json:"expected_body,omitempty"`
	SkipTLSVerify  bool              `json:"skip_tls_verify,omitempty"`
}

// TCPMonitorConfig holds config for TCP port monitors.
type TCPMonitorConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// PingMonitorConfig holds config for ICMP ping monitors.
type PingMonitorConfig struct {
	Host  string `json:"host"`
	Count int    `json:"count,omitempty"` // default 3
}

// DNSMonitorConfig holds config for DNS monitors.
type DNSMonitorConfig struct {
	Host       string `json:"host"`
	RecordType string `json:"record_type,omitempty"` // A, AAAA, CNAME, MX, etc.
	Resolver   string `json:"resolver,omitempty"`    // DNS server to query
	Expected   string `json:"expected,omitempty"`    // Expected answer
}

// SSLMonitorConfig holds config for SSL expiration monitors.
type SSLMonitorConfig struct {
	Host             string `json:"host"`
	Port             int    `json:"port,omitempty"`               // defaults to 443
	DaysBeforeExpiry int    `json:"days_before_expiry,omitempty"` // alert if expiring within this many days (default 7)
}

// SSHMonitorConfig holds config for SSH connectivity monitors.
type SSHMonitorConfig struct {
	Host string `json:"host"`
	Port int    `json:"port,omitempty"` // defaults to 22
}

// JSONAPIMonitorConfig holds config for JSON API response monitors.
type JSONAPIMonitorConfig struct {
	URL           string `json:"url"`
	Method        string `json:"method,omitempty"` // defaults to GET
	Field         string `json:"field"`            // JSON field path, e.g. "status" or "data.health"
	ExpectedValue string `json:"expected_value"`   // Expected string value of the field
	SkipTLSVerify bool   `json:"skip_tls_verify,omitempty"`
}
