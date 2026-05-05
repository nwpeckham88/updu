package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/updu/updu/internal/models"
)

// PrometheusChecker scrapes a Prometheus metrics endpoint and validates a metric value.
type PrometheusChecker struct{}

func (pc *PrometheusChecker) Type() string {
	return "prometheus"
}

func (pc *PrometheusChecker) Check(ctx context.Context, conf *models.Monitor) (*models.CheckResult, error) {
	result := &models.CheckResult{
		MonitorID: conf.ID,
		Status:    models.StatusDown,
		CheckedAt: time.Now().UTC(),
	}

	var cfg models.PrometheusMonitorConfig
	if err := json.Unmarshal(conf.Config, &cfg); err != nil {
		result.Message = "Invalid monitor configuration"
		return result, nil
	}

	if cfg.Host == "" {
		result.Message = "host is required"
		return result, nil
	}

	// Defaults
	if cfg.Port == 0 {
		cfg.Port = 9090
	}
	if cfg.Path == "" {
		cfg.Path = "/metrics"
	}
	if cfg.Comparison == "" {
		cfg.Comparison = "eq"
	}

	// Build URL
	scheme := "http"
	if strings.HasSuffix(cfg.Path, "") && strings.Contains(cfg.Host, ":443") {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d%s", scheme, cfg.Host, cfg.Port, cfg.Path)

	// HTTP GET with timeout
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.Message = fmt.Sprintf("request error: %v", err)
		return result, nil
	}
	req.Header.Set("User-Agent", "updu/prometheus-checker")

	client := &http.Client{
		Timeout: time.Duration(conf.TimeoutS) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipTLSVerify,
			},
		},
	}

	resp, err := client.Do(req)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		result.Message = fmt.Sprintf("request failed: %v", err)
		result.LatencyMs = &latency
		return result, nil
	}
	defer resp.Body.Close()
	result.LatencyMs = &latency

	if resp.StatusCode != http.StatusOK {
		result.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		result.StatusCode = &resp.StatusCode
		return result, nil
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Message = fmt.Sprintf("read error: %v", err)
		return result, nil
	}

	// Parse Prometheus text format and extract metric
	value, err := extractPrometheusMetric(string(body), cfg.MetricName)
	if err != nil {
		result.Message = fmt.Sprintf("metric parse error: %v", err)
		return result, nil
	}

	// Compare value
	if compareValues(value, cfg.ExpectedValue, cfg.Comparison) {
		result.Status = models.StatusUp
		result.Message = fmt.Sprintf("metric %s = %s", cfg.MetricName, value)

		// Capture metadata with metric value and comparison
		metadata := map[string]interface{}{
			"metric_name": cfg.MetricName,
			"value":       value,
			"expected":    cfg.ExpectedValue,
			"comparison":  cfg.Comparison,
		}
		if data, err := json.Marshal(metadata); err == nil {
			result.Metadata = data
		}
	} else {
		result.Message = fmt.Sprintf("metric %s = %s (expected %s via %s)", cfg.MetricName, value, cfg.ExpectedValue, cfg.Comparison)
	}

	return result, nil
}

func (pc *PrometheusChecker) Validate(config json.RawMessage) error {
	var conf models.PrometheusMonitorConfig
	if err := json.Unmarshal(config, &conf); err != nil {
		return fmt.Errorf("invalid prometheus config: %w", err)
	}
	if conf.Host == "" {
		return fmt.Errorf("host required")
	}
	if conf.Port != 0 && (conf.Port < 1 || conf.Port > 65535) {
		return fmt.Errorf("port out of range")
	}
	if conf.MetricName == "" {
		return fmt.Errorf("metric_name required")
	}
	if conf.ExpectedValue == "" {
		return fmt.Errorf("expected_value required")
	}
	valid := map[string]bool{"eq": true, "gt": true, "lt": true, "gte": true, "lte": true}
	if conf.Comparison != "" && !valid[conf.Comparison] {
		return fmt.Errorf("invalid comparison: %s", conf.Comparison)
	}
	return nil
}

// extractPrometheusMetric parses Prometheus text format and extracts a metric value.
// Format: metric_name{labels...} value [timestamp]
func extractPrometheusMetric(body string, metricName string) (string, error) {
	// Handle metric names with label matchers, e.g. "http_requests_total{job=\"api-server\"}"
	// For simplicity, we'll support basic metric name matching (exact match or simple label matching)
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Match metric: split on space, first token is metric_name{labels...}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		metricPart := parts[0]
		value := parts[1]

		// Extract metric name from "metric_name{...}" or "metric_name"
		var actualMetricName string
		if idx := strings.Index(metricPart, "{"); idx >= 0 {
			actualMetricName = metricPart[:idx]
		} else {
			actualMetricName = metricPart
		}

		if actualMetricName == metricName {
			return value, nil
		}
	}

	return "", fmt.Errorf("metric %s not found", metricName)
}

// compareValues compares two string values using the given comparison operator.
func compareValues(actual, expected, comparison string) bool {
	// Try numeric comparison first
	actualNum, err1 := strconv.ParseFloat(actual, 64)
	expectedNum, err2 := strconv.ParseFloat(expected, 64)

	if err1 == nil && err2 == nil {
		switch comparison {
		case "eq":
			return actualNum == expectedNum
		case "gt":
			return actualNum > expectedNum
		case "lt":
			return actualNum < expectedNum
		case "gte":
			return actualNum >= expectedNum
		case "lte":
			return actualNum <= expectedNum
		}
	}

	// Fallback to string comparison
	switch comparison {
	case "eq":
		return actual == expected
	default:
		return false
	}
}
