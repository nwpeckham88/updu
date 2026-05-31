# HTTPS Monitor (with TLS Health)

The HTTPS monitor combines a regular HTTP request check with TLS certificate freshness inspection in a single monitor. Use it when you want one alert source for both "the page broke" and "the certificate is about to expire".

For request-only checks, the [HTTP / HTTPS](/docs/http/index.html) monitor is sufficient. Choose this monitor when certificate expiry warnings matter as much as the response itself.

## Configuration Options

When setting up an HTTPS monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for a response before considering the check failed.

### Request Settings

- **URL:** The full HTTPS URL to monitor (e.g., `https://app.example.com/healthz`). Required.
- **Method:** The HTTP method to use. Defaults to `GET`.
- **Headers:** (Optional) Custom HTTP headers to send with the request.
- **Body:** (Optional) Request body for `POST`/`PUT` methods.
- **Expected Status:** (Optional) A specific status code that the response must match. By default any `2xx`/`3xx` response is treated as up.
- **Expected Body:** (Optional) Substring that must appear in the response body for the check to pass.

### TLS Settings

- **Warning Days:** (Optional) Move the monitor into a warning state when the certificate has fewer than this many days remaining. Defaults to `14`.
- **Skip TLS Verification:** (Optional) Disable certificate validation entirely. Useful for self-signed internal services, but it disables the certificate-expiry feature.

## Example Use Cases

- **Renewal Visibility:** Get a warning fourteen days before a Let's Encrypt certificate expires, even if the underlying renewal cron job has been failing silently.
- **Origin Behind a Proxy:** Verify that the certificate served by your reverse proxy still chains correctly and that the origin still returns the expected health payload.
