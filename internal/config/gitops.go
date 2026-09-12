package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/updu/updu/internal/models"
	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the structure of updu.conf
type YAMLConfig struct {
	// Server
	Host    string `yaml:"host,omitempty"`
	Port    int    `yaml:"port,omitempty"`
	BaseURL string `yaml:"base_url,omitempty"`

	// Database
	DBPath string `yaml:"db_path,omitempty"`

	// Logging
	LogLevel string `yaml:"log_level,omitempty"`

	// Auth
	AuthSecret     string `yaml:"auth_secret,omitempty"`
	SessionTTLDays int    `yaml:"session_ttl_days,omitempty"`
	AdminUser      string `yaml:"admin_user,omitempty"`
	AdminPassword        string `yaml:"admin_password,omitempty"`
	PasswordPolicy       string `yaml:"password_policy,omitempty"`
	DisablePasswordLogin *bool  `yaml:"disable_password_login,omitempty"`

	// OIDC (optional)
	OIDCIssuer       string `yaml:"oidc_issuer,omitempty"`
	OIDCClientID     string `yaml:"oidc_client_id,omitempty"`
	OIDCClientSecret string `yaml:"oidc_client_secret,omitempty"`
	OIDCRedirectURL  string `yaml:"oidc_redirect_url,omitempty"`
	OIDCAutoRegister *bool  `yaml:"oidc_auto_register,omitempty"`
	OIDCAdminGroup   string `yaml:"oidc_admin_group,omitempty"`
	OIDCGroupsClaim  string `yaml:"oidc_groups_claim,omitempty"`

	// Forward Auth (optional)
	ForwardAuthEnabled     *bool  `yaml:"forward_auth_enabled,omitempty"`
	ForwardAuthAdminGroup  string `yaml:"forward_auth_admin_group,omitempty"`
	ForwardAuthUserHeader  string `yaml:"forward_auth_user_header,omitempty"`
	ForwardAuthGroupHeader string `yaml:"forward_auth_group_header,omitempty"`
	ForwardAuthEmailHeader string `yaml:"forward_auth_email_header,omitempty"`

	// Scheduler
	WorkerPoolSize int `yaml:"worker_pool_size,omitempty"`
	MinIntervalS   int `yaml:"min_interval_s,omitempty"`

	EnableCustomCSS   *bool    `yaml:"enable_custom_css,omitempty"`
	TrustedProxyCIDRs []string `yaml:"trusted_proxy_cidrs,omitempty"`

	// Metrics
	MetricsToken string `yaml:"metrics_token,omitempty"`

	// GitOps / Fetch
	ConfURL    string `yaml:"conf_url,omitempty"`
	ConfPath   string `yaml:"conf_path,omitempty"`
	ConfigPath string `yaml:"config_path,omitempty"` // For compatibility/internal use

	Monitors []YAMLMonitor `yaml:"monitors"`
	Services []YAMLService `yaml:"services,omitempty"`
}

// YAMLMonitor is a YAML-friendly representation of models.Monitor
type YAMLMonitor struct {
	ID        string    `yaml:"id,omitempty"`
	Name      string    `yaml:"name"`
	Type      string    `yaml:"type"`
	Groups    []string  `yaml:"groups,omitempty"`
	GroupName string    `yaml:"group,omitempty"`
	Tags      []string  `yaml:"tags,omitempty"`
	Interval  string    `yaml:"interval,omitempty"` // e.g. "60s" or "1m"
	IntervalS int       `yaml:"interval_s,omitempty"`
	Timeout   string    `yaml:"timeout,omitempty"`
	TimeoutS  int       `yaml:"timeout_s,omitempty"`
	Retries   int       `yaml:"retries,omitempty"`
	Enabled   *bool     `yaml:"enabled,omitempty"`
	Config    yaml.Node `yaml:"config"`
}

// YAMLService is a YAML-friendly representation of models.Service
type YAMLService struct {
	ID        string         `yaml:"id,omitempty"`
	Name      string         `yaml:"name"`
	Type      string         `yaml:"type,omitempty"` // web, infra, database, host, job
	Zone      string         `yaml:"zone,omitempty"` // default, homelab, vps
	ZoneID    string         `yaml:"zone_id,omitempty"`
	Groups    []string       `yaml:"groups,omitempty"`
	Tags      []string       `yaml:"tags,omitempty"`
	Enabled   *bool          `yaml:"enabled,omitempty"`
	Endpoints []YAMLEndpoint `yaml:"endpoints"`
}

// YAMLEndpoint is a YAML-friendly representation of models.ServiceEndpoint
type YAMLEndpoint struct {
	ID         string    `yaml:"id,omitempty"`
	Name       string    `yaml:"name"`
	Scope      string    `yaml:"scope,omitempty"` // lan, tailnet, public
	ScopeID    string    `yaml:"scope_id,omitempty"`
	Type       string    `yaml:"type,omitempty"` // http, tcp, ping, dns, push
	TargetType string    `yaml:"target_type,omitempty"`
	Interval   string    `yaml:"interval,omitempty"`
	IntervalS  int       `yaml:"interval_s,omitempty"`
	Timeout    string    `yaml:"timeout,omitempty"`
	TimeoutS   int       `yaml:"timeout_s,omitempty"`
	Retries    int       `yaml:"retries,omitempty"`
	Primary    *bool     `yaml:"primary,omitempty"`
	IsPrimary  *bool     `yaml:"is_primary,omitempty"`
	Config     yaml.Node `yaml:"config"`
}

// ParseYAMLConfig reads and parses updu.conf and any *.updu.conf files in the same directory.
func ParseYAMLConfig(path string) (*YAMLConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg YAMLConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling yaml: %w", err)
	}

	// Look for additional config files (*.updu.conf)
	dir := filepath.Dir(path)
	matches, err := filepath.Glob(filepath.Join(dir, "*.updu.conf"))
	if err == nil {
		for _, mPath := range matches {
			// Skip the main config file itself
			absMain, _ := filepath.Abs(path)
			absMatch, _ := filepath.Abs(mPath)
			if absMain == absMatch {
				continue
			}

			mData, err := os.ReadFile(mPath)
			if err != nil {
				continue // Skip files that can't be read
			}

			var extra YAMLConfig
			if err := yaml.Unmarshal(mData, &extra); err == nil {
				cfg.Monitors = append(cfg.Monitors, extra.Monitors...)
				cfg.Services = append(cfg.Services, extra.Services...)
			}
		}
	}

	return &cfg, nil
}

// ToModels converts YAML monitors to models.Monitor
func (yc *YAMLConfig) ToModels() ([]*models.Monitor, error) {
	var monitors []*models.Monitor

	for _, ym := range yc.Monitors {
		m := &models.Monitor{
			ID:      ym.ID,
			Name:    ym.Name,
			Type:    ym.Type,
			Tags:    ym.Tags,
			Enabled: true,
		}

		if len(ym.Groups) > 0 {
			m.Groups = ym.Groups
		} else if ym.GroupName != "" {
			m.Groups = []string{ym.GroupName}
		}

		if ym.Enabled != nil {
			m.Enabled = *ym.Enabled
		}

		// Handle interval (prefer interval_s if set)
		m.IntervalS = ym.IntervalS
		if m.IntervalS == 0 && ym.Interval != "" {
			// Basic parsing for now, maybe use time.ParseDuration later
			m.IntervalS = parseSimpleDuration(ym.Interval)
		}
		if m.IntervalS == 0 {
			m.IntervalS = 60 // Default
		}

		// Handle timeout
		m.TimeoutS = ym.TimeoutS
		if m.TimeoutS == 0 && ym.Timeout != "" {
			m.TimeoutS = parseSimpleDuration(ym.Timeout)
		}
		if m.TimeoutS == 0 {
			m.TimeoutS = 10 // Default
		}

		m.Retries = ym.Retries

		// Convert YAML config node to JSON for models.Monitor
		configBytes, err := yamlNodeToJSON(ym.Config)
		if err != nil {
			return nil, fmt.Errorf("converting config for monitor %s: %w", ym.Name, err)
		}
		m.Config = json.RawMessage(configBytes)

		monitors = append(monitors, m)
	}

	return monitors, nil
}

// ServicesToModels converts YAML services to models.Service
func (yc *YAMLConfig) ServicesToModels() ([]*models.Service, error) {
	var services []*models.Service

	for _, ys := range yc.Services {
		svcType := ys.Type
		if svcType == "" {
			svcType = models.ServiceTypeWeb
		}

		zoneID := ys.ZoneID
		if zoneID == "" {
			zoneID = ys.Zone
		}
		if zoneID == "" {
			zoneID = "default"
		}

		s := &models.Service{
			ID:        ys.ID,
			Name:      ys.Name,
			Type:      svcType,
			ZoneID:    zoneID,
			Groups:    ys.Groups,
			Tags:      ys.Tags,
			Enabled:   true,
			Endpoints: make([]*models.ServiceEndpoint, 0, len(ys.Endpoints)),
		}

		if ys.Enabled != nil {
			s.Enabled = *ys.Enabled
		}

		// Check if any endpoint is explicitly marked primary
		hasExplicitPrimary := false
		for _, ye := range ys.Endpoints {
			if (ye.IsPrimary != nil && *ye.IsPrimary) || (ye.Primary != nil && *ye.Primary) {
				hasExplicitPrimary = true
				break
			}
		}

		for i, ye := range ys.Endpoints {
			scopeID := ye.ScopeID
			if scopeID == "" {
				scopeID = ye.Scope
			}
			if scopeID == "" {
				scopeID = models.ScopePublic
			}

			targetType := ye.TargetType
			if targetType == "" {
				targetType = ye.Type
			}
			if targetType == "" {
				targetType = svcType
			}

			intervalS := ye.IntervalS
			if intervalS == 0 && ye.Interval != "" {
				intervalS = parseSimpleDuration(ye.Interval)
			}
			if intervalS == 0 {
				intervalS = 30
			}

			timeoutS := ye.TimeoutS
			if timeoutS == 0 && ye.Timeout != "" {
				timeoutS = parseSimpleDuration(ye.Timeout)
			}
			if timeoutS == 0 {
				timeoutS = 10
			}

			retries := ye.Retries
			if retries <= 0 {
				retries = 2
			}

			isPrimary := false
			if ye.IsPrimary != nil {
				isPrimary = *ye.IsPrimary
			} else if ye.Primary != nil {
				isPrimary = *ye.Primary
			} else if !hasExplicitPrimary && i == 0 {
				isPrimary = true
			}

			configBytes, err := yamlNodeToJSON(ye.Config)
			if err != nil {
				return nil, fmt.Errorf("converting config for service %s endpoint %s: %w", ys.Name, ye.Name, err)
			}

			ep := &models.ServiceEndpoint{
				ID:         ye.ID,
				Name:       ye.Name,
				ScopeID:    scopeID,
				TargetType: targetType,
				Config:     json.RawMessage(configBytes),
				IntervalS:  intervalS,
				TimeoutS:   timeoutS,
				Retries:    retries,
				IsPrimary:  isPrimary,
			}
			s.Endpoints = append(s.Endpoints, ep)
		}

		services = append(services, s)
	}

	return services, nil
}

// FromModels converts models.Monitor and settings to YAMLConfig
func FromModels(monitors []*models.Monitor, settings map[string]string) *YAMLConfig {
	yc := &YAMLConfig{
		Monitors: make([]YAMLMonitor, 0, len(monitors)),
	}

	// Map key settings if present
	if v, ok := settings["host"]; ok {
		yc.Host = v
	}
	if v, ok := settings["port"]; ok {
		if p, err := strconv.Atoi(v); err == nil {
			yc.Port = p
		}
	}
	if v, ok := settings["base_url"]; ok {
		yc.BaseURL = v
	}
	if v, ok := settings["db_path"]; ok {
		yc.DBPath = v
	}
	if v, ok := settings["log_level"]; ok {
		yc.LogLevel = v
	}
	if v, ok := settings["trusted_proxy_cidrs"]; ok && strings.TrimSpace(v) != "" {
		yc.TrustedProxyCIDRs = strings.Split(v, ",")
		for i := range yc.TrustedProxyCIDRs {
			yc.TrustedProxyCIDRs[i] = strings.TrimSpace(yc.TrustedProxyCIDRs[i])
		}
	}

	for _, m := range monitors {
		ym := YAMLMonitor{
			ID:        m.ID,
			Name:      m.Name,
			Type:      m.Type,
			Groups:    m.Groups,
			Tags:      m.Tags,
			IntervalS: m.IntervalS,
			TimeoutS:  m.TimeoutS,
			Retries:   m.Retries,
			Enabled:   &m.Enabled,
		}

		// Convert JSON config to YAML Node
		var configObj interface{}
		if err := json.Unmarshal(m.Config, &configObj); err == nil {
			var node yaml.Node
			if err := node.Encode(configObj); err == nil {
				ym.Config = node
			}
		}

		yc.Monitors = append(yc.Monitors, ym)
	}

	return yc
}

func yamlNodeToJSON(node yaml.Node) ([]byte, error) {
	var obj interface{}
	if err := node.Decode(&obj); err != nil {
		return nil, err
	}
	return json.Marshal(obj)
}

func parseSimpleDuration(s string) int {
	// Very simple parser for now
	var val int
	var unit string
	fmt.Sscanf(s, "%d%s", &val, &unit)

	switch unit {
	case "s":
		return val
	case "m":
		return val * 60
	case "h":
		return val * 3600
	default:
		return val
	}
}
