package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// TransactionChecker runs a sequential multi-step HTTP chain where each step can
// extract values from the response and inject them into subsequent steps.
type TransactionChecker struct{}

func (c *TransactionChecker) Type() string { return "transaction" }

func (c *TransactionChecker) Validate(config json.RawMessage) error {
	var cfg models.TransactionMonitorConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid transaction config: %w", err)
	}
	if len(cfg.Steps) == 0 {
		return fmt.Errorf("steps must not be empty")
	}
	for i, step := range cfg.Steps {
		if step.URL == "" {
			return fmt.Errorf("step %d: url is required", i+1)
		}
	}
	return nil
}

func (c *TransactionChecker) Check(ctx context.Context, monitor *models.Monitor) (*models.CheckResult, error) {
	var cfg models.TransactionMonitorConfig
	if err := json.Unmarshal(monitor.Config, &cfg); err != nil {
		return failResult(monitor.ID, "invalid config: "+err.Error()), nil
	}

	timeout := time.Duration(monitor.TimeoutS) * time.Second
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

	type stepMeta struct {
		Step       int `json:"step"`
		StatusCode int `json:"status_code"`
		LatencyMs  int `json:"latency_ms"`
	}

	vars := make(map[string]string)
	var metas []stepMeta
	chainStart := time.Now()

	for i, step := range cfg.Steps {
		stepNum := i + 1

		rawURL := substituteVars(step.URL, vars)
		rawBody := substituteVars(step.Body, vars)

		method := step.Method
		if method == "" {
			method = "GET"
		}

		var bodyReader io.Reader
		if rawBody != "" {
			bodyReader = strings.NewReader(rawBody)
		}

		req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
		if err != nil {
			totalLatency := int(time.Since(chainStart).Milliseconds())
			return &models.CheckResult{
				MonitorID: monitor.ID,
				Status:    models.StatusDown,
				LatencyMs: &totalLatency,
				Message:   fmt.Sprintf("step %d: building request: %s", stepNum, err.Error()),
				CheckedAt: time.Now(),
			}, nil
		}
		for k, v := range step.Headers {
			req.Header.Set(k, substituteVars(v, vars))
		}

		stepStart := time.Now()
		resp, err := client.Do(req)
		stepLatency := int(time.Since(stepStart).Milliseconds())

		if err != nil {
			totalLatency := int(time.Since(chainStart).Milliseconds())
			return &models.CheckResult{
				MonitorID: monitor.ID,
				Status:    models.StatusDown,
				LatencyMs: &totalLatency,
				Message:   fmt.Sprintf("step %d: %s", stepNum, err.Error()),
				CheckedAt: time.Now(),
			}, nil
		}

		statusCode := resp.StatusCode
		metas = append(metas, stepMeta{Step: stepNum, StatusCode: statusCode, LatencyMs: stepLatency})

		// Evaluate status code before reading body
		if statusCode < 200 || statusCode >= 400 {
			resp.Body.Close()
			totalLatency := int(time.Since(chainStart).Milliseconds())
			sc := statusCode
			return &models.CheckResult{
				MonitorID:  monitor.ID,
				Status:     models.StatusDown,
				LatencyMs:  &totalLatency,
				StatusCode: &sc,
				Message:    fmt.Sprintf("step %d: HTTP %d", stepNum, statusCode),
				CheckedAt:  time.Now(),
			}, nil
		}
		if step.ExpectedStatus > 0 && statusCode != step.ExpectedStatus {
			resp.Body.Close()
			totalLatency := int(time.Since(chainStart).Milliseconds())
			sc := statusCode
			return &models.CheckResult{
				MonitorID:  monitor.ID,
				Status:     models.StatusDown,
				LatencyMs:  &totalLatency,
				StatusCode: &sc,
				Message:    fmt.Sprintf("step %d: expected status %d, got %d", stepNum, step.ExpectedStatus, statusCode),
				CheckedAt:  time.Now(),
			}, nil
		}

		// Read body (needed for ExpectedBody check and Extract)
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil {
			totalLatency := int(time.Since(chainStart).Milliseconds())
			sc := statusCode
			return &models.CheckResult{
				MonitorID:  monitor.ID,
				Status:     models.StatusDown,
				LatencyMs:  &totalLatency,
				StatusCode: &sc,
				Message:    fmt.Sprintf("step %d: reading body: %s", stepNum, err.Error()),
				CheckedAt:  time.Now(),
			}, nil
		}

		if step.ExpectedBody != "" && !strings.Contains(string(bodyBytes), step.ExpectedBody) {
			totalLatency := int(time.Since(chainStart).Milliseconds())
			sc := statusCode
			return &models.CheckResult{
				MonitorID:  monitor.ID,
				Status:     models.StatusDown,
				LatencyMs:  &totalLatency,
				StatusCode: &sc,
				Message:    fmt.Sprintf("step %d: body missing keyword %q", stepNum, step.ExpectedBody),
				CheckedAt:  time.Now(),
			}, nil
		}

		// Extract variables from JSON response body
		if len(step.Extract) > 0 {
			var jsonData map[string]any
			if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {
				for varName, path := range step.Extract {
					if val, found := navigateJSON(jsonData, path); found {
						vars[varName] = fmt.Sprintf("%v", val)
					}
				}
			}
			// If body is not JSON, extraction is silently skipped.
		}
	}

	totalLatency := int(time.Since(chainStart).Milliseconds())
	metadata, _ := json.Marshal(map[string]any{"steps": metas})

	return &models.CheckResult{
		MonitorID: monitor.ID,
		Status:    models.StatusUp,
		LatencyMs: &totalLatency,
		Message:   fmt.Sprintf("all %d step(s) passed", len(cfg.Steps)),
		Metadata:  metadata,
		CheckedAt: time.Now(),
	}, nil
}

// substituteVars replaces all {{varName}} occurrences in s using the provided map.
func substituteVars(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}
