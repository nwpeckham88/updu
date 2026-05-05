# Prometheus Monitor

Scrape a Prometheus `/metrics` endpoint and validate that a specific metric meets expected criteria.

## Use Cases

- **Metric Validation**: Ensure application metrics are within expected ranges
- **Health Threshold Monitoring**: Alert when system metrics exceed thresholds
- **Prometheus Instance Health**: Monitor that Prometheus scrapers are healthy and updating metrics
- **Custom Application Metrics**: Track application-specific metrics exposed via Prometheus format

## Configuration

```yaml
monitors:
  - id: prometheus-request-rate
    name: Prometheus Request Rate
    type: prometheus
    interval: 2m
    timeout: 10s
    config:
      host: prometheus.internal.example
      port: 9090
      path: /metrics
      metric_name: http_requests_total
      expected_value: "1"
      comparison: gt
      skip_tls_verify: false
```

## Configuration Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `host` | ✓ | — | Prometheus server hostname or IP address |
| `port` | ✗ | 9090 | Prometheus server port |
| `path` | ✗ | /metrics | HTTP path to metrics endpoint |
| `metric_name` | ✓ | — | Prometheus metric name to extract (e.g., `http_requests_total`, `up`) |
| `expected_value` | ✓ | — | Expected value for comparison (numeric or string) |
| `comparison` | ✗ | eq | Comparison operator: `eq` (equal), `gt` (greater than), `lt` (less than), `gte` (≥), `lte` (≤) |
| `skip_tls_verify` | ✗ | false | Skip TLS certificate verification (not recommended for production) |

## Metric Format

The monitor extracts metrics from Prometheus text format (OpenMetrics):

```
# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200"} 1234
http_requests_total{method="POST",status="201"} 567
```

The monitor extracts the first matching metric name and uses its value for comparison.

## Comparison Operators

- **eq**: Value equals expected (default)
- **gt**: Value greater than expected
- **lt**: Value less than expected
- **gte**: Value greater than or equal to expected
- **lte**: Value less than or equal to expected

## Example: Monitor Application Requests

```yaml
monitors:
  - id: app-request-rate
    name: App Request Rate
    type: prometheus
    groups: [Application]
    interval: 1m
    timeout: 5s
    config:
      host: app.internal.example
      port: 8080
      path: /metrics
      metric_name: app_http_requests_total
      expected_value: "100"
      comparison: gt
```

This checks that the app has processed more than 100 HTTP requests.

## Example: Monitor System Health

```yaml
monitors:
  - id: prometheus-up
    name: Prometheus Up
    type: prometheus
    groups: [Observability]
    interval: 5m
    timeout: 10s
    config:
      host: prometheus.internal.example
      port: 9090
      metric_name: up
      expected_value: "1"
      comparison: eq
```

This checks that the `up` metric (standard Prometheus metric) is 1 (healthy).

## Metadata

The monitor captures the following metadata in check results:

```json
{
  "metric_name": "http_requests_total",
  "value": "1234",
  "expected": "1000",
  "comparison": "gt"
}
```

## Common Issues

**Metric not found**: Ensure the metric is exposed by the Prometheus target and the metric name matches exactly (metric names are case-sensitive).

**TLS certificate errors**: Use `skip_tls_verify: true` only in development. For production, ensure the certificate is valid or use a proxy with valid certificates.

**Timeout errors**: Increase the `timeout` if the Prometheus instance is slow to respond or the metrics endpoint is large.

## See Also

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Prometheus Text Format](https://prometheus.io/docs/instrumenting/exposition_formats/)
- [Prometheus HTTP API](https://prometheus.io/docs/prometheus/latest/querying/api/)
