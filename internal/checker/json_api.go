package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// JSONAPIChecker monitors JSON API endpoints by validating a specific field value.
type JSONAPIChecker struct{}

func (c *JSONAPIChecker) Type() string { return "json" }

func (c *JSONAPIChecker) Validate(config json.RawMessage) error {
	var cfg models.JSONAPIMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid JSON API config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("url is required")
	}
	if cfg.Field == "" {
		return fmt.Errorf("field is required")
	}
	if cfg.ExpectedValue == "" {
		return fmt.Errorf("expected_value is required")
	}
	return nil
}

func (c *JSONAPIChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.JSONAPIMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	method := cfg.Method
	if method == "" {
		method = "GET"
	}

	timeout := time.Duration(monitor.TimeoutS) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipTLSVerify,
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, nil)
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

	if statusCode < 200 || statusCode >= 400 {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("HTTP %d", statusCode)
		return result, nil
	}

	// Read and parse JSON body (1MB limit)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		result.Status = models.StatusDown
		result.Message = "reading body: " + err.Error()
		return result, nil
	}

	// Parse into generic map
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		result.Status = models.StatusDown
		result.Message = "invalid JSON response: " + err.Error()
		return result, nil
	}

	// Navigate dot-separated field path (e.g., "data.status")
	actual, found := navigateJSON(data, cfg.Field)
	if !found {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("field %q not found in response", cfg.Field)
		return result, nil
	}

	actualStr := fmt.Sprintf("%v", actual)
	if actualStr != cfg.ExpectedValue {
		result.Status = models.StatusDown
		result.Message = fmt.Sprintf("field %q = %q, expected %q", cfg.Field, actualStr, cfg.ExpectedValue)
		return result, nil
	}

	result.Status = models.StatusUp
	result.Message = fmt.Sprintf("%s = %s", cfg.Field, actualStr)
	return result, nil
}

// navigateJSON traverses a dot-separated path in a JSON object.
func navigateJSON(data map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
