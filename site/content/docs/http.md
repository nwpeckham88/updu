# HTTP / HTTPS Monitor

The HTTP monitor in updu checks the availability, performance, and TLS certificate health of web endpoints. It supports both HTTP and HTTPS protocols in a single unified probe.

## Configuration Options

When setting up an HTTP monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval:** How frequently updu should perform the check (e.g. 60s).
- **Timeout:** The maximum time updu will wait for a response before considering the check failed.

### HTTP & TLS Settings

- **URL:** The full URL to monitor (e.g., `https://example.com` or `http://localhost:8080/health`).
- **Method:** HTTP method (`GET`, `POST`, `PUT`, `HEAD`). Defaults to `GET`.
- **Expected Status:** The expected HTTP status code (defaults to `200`).
- **Expected Body:** (Optional) Substring that must be present in the response body for the check to pass.
- **TLS Expiry Warning Threshold (`warn_days`):** Number of days before certificate expiration to emit a warning (defaults to 14 days).
- **Skip TLS Verification (`skip_tls_verify`):** Set to true for self-signed certificates or internal homelab services.

## Example Use Cases

- **Website Uptime & Certificate Tracking:** Monitor a public site, assert `200 OK` and body content, while continuously watching TLS certificate validity without needing a separate SSL checker.
- **API Health Check:** Query an internal `/healthz` microservice endpoint.
