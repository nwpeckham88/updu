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
	Groups    []string        `json:"groups,omitempty"`
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
	Status        MonitorStatus         `json:"status,omitempty"`
	LastCheck     *time.Time            `json:"last_check,omitempty"`
	LastLatency   *int                  `json:"last_latency_ms,omitempty"`
	Investigation *MonitorInvestigation `json:"investigation,omitempty"`
}

// MonitorInvestigation represents an in-memory operator marker for an active issue.
type MonitorInvestigation struct {
	MonitorID string    `json:"monitor_id"`
	Active    bool      `json:"active"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
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

// PushMonitorConfig holds config for push-style check-in monitors.
type PushMonitorConfig struct {
	Token        string `json:"token"`                    // Generated API key required in the request
	GracePeriodS *int   `json:"grace_period_s,omitempty"` // Extra tolerance after the expected interval, in seconds
}

// WebSocketMonitorConfig holds config for WebSocket monitors.
type WebSocketMonitorConfig struct {
	URL           string `json:"url"`
	SkipTLSVerify bool   `json:"skip_tls_verify,omitempty"`
}

// SMTPMonitorConfig holds config for SMTP monitors.
type SMTPMonitorConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port,omitempty"` // usually 25, 465, or 587
	RequireTLS bool   `json:"require_tls,omitempty"`
}

// UDPMonitorConfig holds config for UDP monitors.
type UDPMonitorConfig struct {
	Host             string `json:"host"`
	Port             int    `json:"port"`
	SendPayload      string `json:"send_payload,omitempty"`
	ExpectedResponse string `json:"expected_response,omitempty"`
}

// RedisMonitorConfig holds config for Redis monitors.
type RedisMonitorConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password,omitempty"`
	Database int    `json:"database,omitempty"` // defaults to 0
}

// PostgresMonitorConfig holds config for PostgreSQL monitors.
type PostgresMonitorConfig struct {
	ConnectionString string `json:"connection_string,omitempty"`
	Host             string `json:"host,omitempty"`
	Port             int    `json:"port,omitempty"` // usually 5432
	User             string `json:"user,omitempty"`
	Password         string `json:"password,omitempty"`
	Database         string `json:"database,omitempty"`
	SSLMode          string `json:"ssl_mode,omitempty"`
}

// MySQLMonitorConfig holds config for MySQL/MariaDB monitors.
type MySQLMonitorConfig struct {
	ConnectionString string `json:"connection_string,omitempty"`
	Host             string `json:"host,omitempty"`
	Port             int    `json:"port,omitempty"` // usually 3306
	User             string `json:"user,omitempty"`
	Password         string `json:"password,omitempty"`
	Database         string `json:"database,omitempty"`
}

// MongoMonitorConfig holds config for MongoDB monitors.
type MongoMonitorConfig struct {
	ConnectionString string `json:"connection_string"`
}

// HTTPSMonitorConfig holds config for combined HTTP + TLS monitors.
type HTTPSMonitorConfig struct {
	URL            string            `json:"url"`
	Method         string            `json:"method,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           string            `json:"body,omitempty"`
	ExpectedStatus int               `json:"expected_status,omitempty"`
	ExpectedBody   string            `json:"expected_body,omitempty"`
	SkipTLSVerify  bool              `json:"skip_tls_verify,omitempty"`
	WarnDays       int               `json:"warn_days,omitempty"` // default 14
}

// CompositeMonitorConfig holds config for K-of-N quorum monitors.
type CompositeMonitorConfig struct {
	MonitorIDs []string `json:"monitor_ids"`
	Mode       string   `json:"mode"` // "all_up" | "any_up" | "quorum"
	Quorum     int      `json:"quorum,omitempty"`
}

// TransactionStep is a single HTTP step in a transaction monitor.
type TransactionStep struct {
	Method         string            `json:"method,omitempty"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           string            `json:"body,omitempty"`
	Extract        map[string]string `json:"extract,omitempty"` // varName -> dot-path
	ExpectedStatus int               `json:"expected_status,omitempty"`
	ExpectedBody   string            `json:"expected_body,omitempty"`
}

// TransactionMonitorConfig holds config for multi-step HTTP chain monitors.
type TransactionMonitorConfig struct {
	Steps         []TransactionStep `json:"steps"`
	SkipTLSVerify bool              `json:"skip_tls_verify,omitempty"`
}

// DNSHTTPMonitorConfig holds config for DNS validation + HTTP monitors.
type DNSHTTPMonitorConfig struct {
	URL              string `json:"url"`
	ExpectedIPPrefix string `json:"expected_ip_prefix,omitempty"`
	ExpectedCNAME    string `json:"expected_cname,omitempty"`
	ExpectedStatus   int    `json:"expected_status,omitempty"`
	ExpectedBody     string `json:"expected_body,omitempty"`
	SkipTLSVerify    bool   `json:"skip_tls_verify,omitempty"`
}

// GRPCMonitorConfig holds config for gRPC health-check monitors.
// Calls the standard grpc.health.v1.Health/Check RPC and expects SERVING status.
type GRPCMonitorConfig struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Service            string `json:"service,omitempty"`              // service name passed to Health.Check; empty = overall server health
	TLS                bool   `json:"tls,omitempty"`                  // use TLS transport
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"` // skip server cert verification when TLS=true
	Authority          string `json:"authority,omitempty"`            // optional :authority pseudo-header override
}

// RedactMonitorConfig returns a copy of the monitor's Config with sensitive fields
// (passwords, connection strings containing credentials) replaced with a placeholder.
// This should be used when serializing monitors for non-admin API responses.
func RedactMonitorConfig(monitorType string, config json.RawMessage) json.RawMessage {
	const redacted = "**REDACTED**"

	switch monitorType {
	case "redis":
		var cfg RedisMonitorConfig
		if err := json.Unmarshal(config, &cfg); err == nil && cfg.Password != "" {
			cfg.Password = redacted
			if out, err := json.Marshal(cfg); err == nil {
				return out
			}
		}
	case "postgres":
		var cfg PostgresMonitorConfig
		if err := json.Unmarshal(config, &cfg); err == nil {
			if cfg.Password != "" {
				cfg.Password = redacted
			}
			if cfg.ConnectionString != "" {
				cfg.ConnectionString = redacted
			}
			if out, err := json.Marshal(cfg); err == nil {
				return out
			}
		}
	case "mysql":
		var cfg MySQLMonitorConfig
		if err := json.Unmarshal(config, &cfg); err == nil {
			if cfg.Password != "" {
				cfg.Password = redacted
			}
			if cfg.ConnectionString != "" {
				cfg.ConnectionString = redacted
			}
			if out, err := json.Marshal(cfg); err == nil {
				return out
			}
		}
	case "mongo":
		var cfg MongoMonitorConfig
		if err := json.Unmarshal(config, &cfg); err == nil && cfg.ConnectionString != "" {
			cfg.ConnectionString = redacted
			if out, err := json.Marshal(cfg); err == nil {
				return out
			}
		}
	case "push":
		var cfg PushMonitorConfig
		if err := json.Unmarshal(config, &cfg); err == nil && cfg.Token != "" {
			cfg.Token = redacted
			if out, err := json.Marshal(cfg); err == nil {
				return out
			}
		}
	}
	return config
}

// RedactMonitor returns a shallow copy of the monitor with sensitive config fields redacted.
func RedactMonitor(m *Monitor) Monitor {
	redacted := *m
	redacted.Config = RedactMonitorConfig(m.Type, m.Config)
	redacted.Investigation = nil
	return redacted
}
