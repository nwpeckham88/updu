package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	AdminPassword  string `yaml:"admin_password,omitempty"`

	// OIDC (optional)
	OIDCIssuer       string `yaml:"oidc_issuer,omitempty"`
	OIDCClientID     string `yaml:"oidc_client_id,omitempty"`
	OIDCClientSecret string `yaml:"oidc_client_secret,omitempty"`
	OIDCRedirectURL  string `yaml:"oidc_redirect_url,omitempty"`
	OIDCAutoRegister *bool  `yaml:"oidc_auto_register,omitempty"`

	// Scheduler
	WorkerPoolSize int `yaml:"worker_pool_size,omitempty"`
	MinIntervalS   int `yaml:"min_interval_s,omitempty"`

	EnableCustomCSS *bool `yaml:"enable_custom_css,omitempty"`

	// GitOps / Fetch
	ConfURL    string `yaml:"conf_url,omitempty"`
	ConfPath   string `yaml:"conf_path,omitempty"`
	ConfigPath string `yaml:"config_path,omitempty"` // For compatibility/internal use

	Monitors []YAMLMonitor `yaml:"monitors"`
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
