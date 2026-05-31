package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

type sablierServiceSpec struct {
	Replicas int    `json:"replicas"`
	TTL      string `json:"ttl"`
}

type sablierServiceResponse struct {
	Name     string             `json:"name"`
	Status   string             `json:"status"`
	Replicas int                `json:"replicas"`
	Spec     sablierServiceSpec `json:"spec"`
}

// SablierChecker queries Sablier's direct service-state API so checks do not
// wake sleeping containers through the reverse-proxy path.
type SablierChecker struct{}

func (c *SablierChecker) Type() string { return "sablier" }

func (c *SablierChecker) Validate(config json.RawMessage) error {
	var cfg models.SablierMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid sablier config: %w", err)
	}
	if strings.TrimSpace(cfg.URL) == "" {
		return fmt.Errorf("url is required")
	}
	if strings.TrimSpace(cfg.ServiceName) == "" {
		return fmt.Errorf("service_name is required")
	}
	return nil
}

func (c *SablierChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.SablierMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	endpoint := strings.TrimRight(strings.TrimSpace(cfg.URL), "/") + "/api/services/" + url.PathEscape(strings.TrimSpace(cfg.ServiceName))

	timeout := time.Duration(monitor.TimeoutS) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	dialer := &net.Dialer{
		Timeout: timeout,
		Control: SafeDialer(ctx),
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipTLSVerify, // #nosec G402
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return failResult(monitor.ID, "creating request: "+err.Error()), nil
	}
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := client.Do(req)
	latency := int(time.Since(start).Milliseconds())

	result := &models.CheckResult{
		MonitorID: monitor.ID,
		LatencyMs: &latency,
		CheckedAt: time.Now(),
	}

	if err != nil {
		result.Status = models.StatusDown
		result.Message = err.Error()
		return result, nil
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	result.StatusCode = &statusCode

	if statusCode < 200 || statusCode >= 300 {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("HTTP %d", statusCode)
		return result, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		result.Status = models.StatusDown
		result.Message = "reading body: " + err.Error()
		return result, nil
	}

	var payload sablierServiceResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		result.Status = models.StatusDown
		result.Message = "invalid JSON response: " + err.Error()
		return result, nil
	}

	serviceState := strings.ToLower(strings.TrimSpace(payload.Status))
	if serviceState == "" {
		result.Status = models.StatusDown
		result.Message = "missing sablier state"
		return result, nil
	}

	metadata, _ := json.Marshal(map[string]any{
		"service_name":     payload.Name,
		"sablier_status":   serviceState,
		"replicas":         payload.Replicas,
		"desired_replicas": payload.Spec.Replicas,
		"ttl":              payload.Spec.TTL,
	})
	result.Metadata = metadata

	statusSummary := fmt.Sprintf("%d/%d replicas", payload.Replicas, payload.Spec.Replicas)

	switch serviceState {
	case "sleeping":
		if payload.Replicas != 0 {
			result.Status = models.StatusDegraded
			result.Message = fmt.Sprintf("service sleeping but has %s", statusSummary)
			return result, nil
		}
		result.Status = models.StatusUp
		result.Message = fmt.Sprintf("service sleeping (%s)", statusSummary)
		return result, nil
	case "starting":
		result.Status = models.StatusPending
		result.Message = fmt.Sprintf("service starting (%s)", statusSummary)
		return result, nil
	case "ready":
		if payload.Replicas == 0 {
			result.Status = models.StatusDegraded
			result.Message = "service ready but has 0 replicas"
			return result, nil
		}
		result.Status = models.StatusUp
		result.Message = fmt.Sprintf("service ready (%s)", statusSummary)
		return result, nil
	default:
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("unknown sablier state %q", serviceState)
		return result, nil
	}
}