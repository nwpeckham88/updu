# HTTP / HTTPS Monitor

The HTTP monitor in updu allows you to check the availability and responsiveness of web endpoints. It supports both HTTP and HTTPS protocols.

## Configuration Options

When setting up an HTTP monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for a response before considering the check failed.

### HTTP Specific Settings

- **URL / Host:** The full URL to monitor (e.g., `https://example.com/api/health`).
- **Expected Status Codes:** A comma-separated list of HTTP status codes that indicate a successful response (e.g., `200, 201, 301`). By default, `200` is expected.
- **Keyword Matching:** (Optional) A specific keyword or phrase that must be present in the response body for the check to pass. This is useful for verifying that an application is not just responding, but returning the correct content.
- **Invert Keyword:** (Optional) If checked, the monitor will *fail* if the specified keyword is found in the response body. Useful for detecting error pages.

## Example Use Cases

- **Website Uptime:** Monitoring an e-commerce site ensures it returns a `200` status code and contains the word "Checkout."
- **Internal API Health:** Checking a `/healthz` endpoint on a microservice.
