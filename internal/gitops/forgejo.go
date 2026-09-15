package gitops

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/config"
	"github.com/updu/updu/internal/scheduler"
	"github.com/updu/updu/internal/storage"
	"gopkg.in/yaml.v3"
)

// SyncResult holds summary statistics of a completed GitOps synchronization.
type SyncResult struct {
	MonitorsSynced int      `json:"monitors_synced"`
	ServicesSynced int      `json:"services_synced"`
	Errors         []string `json:"errors,omitempty"`
}

// VerifySignature verifies a Forgejo/Gitea webhook request using HMAC-SHA256 or token.
func VerifySignature(payload []byte, secret, signature, token string) bool {
	if secret == "" {
		return true // No secret configured
	}

	// 1. Check direct token header (e.g. X-Forgejo-Token or X-Gitea-Token)
	if token != "" && hmac.Equal([]byte(token), []byte(secret)) {
		return true
	}

	// 2. Check HMAC-SHA256 signature
	if signature != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		expected := hex.EncodeToString(mac.Sum(nil))
		if hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected))) {
			return true
		}
	}

	return false
}

// ResolveRawURL computes the raw URL for fetching updu.conf from Forgejo.
func ResolveRawURL(cfg *config.GitOpsConfig) (string, error) {
	if cfg.RawURL != "" {
		return cfg.RawURL, nil
	}

	serverURL := strings.TrimRight(cfg.ServerURL, "/")
	if serverURL == "" {
		return "", fmt.Errorf("gitops: server_url is required")
	}
	repo := strings.Trim(cfg.Repository, "/")
	if repo == "" {
		return "", fmt.Errorf("gitops: repository is required")
	}
	branch := cfg.Branch
	if branch == "" {
		branch = "main"
	}
	path := cfg.Path
	if path == "" {
		path = "updu.conf"
	}

	// Forgejo/Gitea raw API endpoint
	return fmt.Sprintf("%s/api/v1/repos/%s/raw/%s?ref=%s", serverURL, repo, path, branch), nil
}

// FetchRawConfig fetches the raw configuration file from Forgejo.
func FetchRawConfig(ctx context.Context, cfg *config.GitOpsConfig) ([]byte, error) {
	fetchURL, err := ResolveRawURL(cfg)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	if cfg.Token != "" {
		req.Header.Set("Authorization", "token "+cfg.Token)
	}
	req.Header.Set("User-Agent", "updu-gitops-sync")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching from forgejo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("forgejo returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB limit
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return data, nil
}

// SyncConfig parses raw YAML bytes, validates checker configurations, persists them to SQLite,
// and hot-reloads running schedulers.
func SyncConfig(ctx context.Context, data []byte, reg *checker.Registry, db *storage.DB, sched *scheduler.Scheduler) (*SyncResult, error) {
	var yCfg config.YAMLConfig
	if err := yaml.Unmarshal(data, &yCfg); err != nil {
		return nil, fmt.Errorf("unmarshaling yaml: %w", err)
	}

	monitors, err := yCfg.ToModels()
	if err != nil {
		return nil, fmt.Errorf("converting monitors: %w", err)
	}

	// Validate checkers
	if reg != nil {
		for _, m := range monitors {
			c := reg.Get(m.Type)
			if c == nil {
				return nil, fmt.Errorf("unknown monitor type: %s (monitor: %s)", m.Type, m.Name)
			}
			if err := c.Validate(m.Config); err != nil {
				return nil, fmt.Errorf("invalid config for monitor %s: %w", m.Name, err)
			}
		}
	}

	services, err := yCfg.ServicesToModels()
	if err != nil {
		return nil, fmt.Errorf("converting services: %w", err)
	}

	if reg != nil && len(services) > 0 {
		for _, svc := range services {
			for _, ep := range svc.Endpoints {
				c := reg.Get(ep.TargetType)
				if c == nil {
					return nil, fmt.Errorf("unknown endpoint type: %s (service: %s, ep: %s)", ep.TargetType, svc.Name, ep.Name)
				}
				if err := c.Validate(ep.Config); err != nil {
					return nil, fmt.Errorf("invalid config for service %s endpoint %s: %w", svc.Name, ep.Name, err)
				}
			}
		}
	}

	// Commit to SQLite
	if err := db.SyncMonitors(ctx, monitors); err != nil {
		return nil, fmt.Errorf("syncing monitors to database: %w", err)
	}

	if len(services) > 0 {
		if err := db.SyncServices(ctx, services); err != nil {
			return nil, fmt.Errorf("syncing services to database: %w", err)
		}
	}

	// Live-reload scheduler monitors without downtime
	if sched != nil {
		for _, m := range monitors {
			sched.ReloadMonitor(context.Background(), m)
		}
	}

	return &SyncResult{
		MonitorsSynced: len(monitors),
		ServicesSynced: len(services),
	}, nil
}
