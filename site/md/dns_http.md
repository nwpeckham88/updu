# DNS + HTTP Monitor

The DNS + HTTP monitor performs a DNS resolution check first and then an HTTP request, recording both results in a single monitor. It is designed to surface CDN misrouting, DNS failover regressions, and stale records that would otherwise hide behind a "the site loads" check.

DNS results never short-circuit the HTTP step — both are recorded so you can see when DNS is wrong even though the origin still answers.

## Configuration Options

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for either step before considering the check failed.

### DNS + HTTP Specific Settings

- **URL:** The full HTTP/HTTPS URL to monitor. The hostname is extracted automatically for the DNS step. Required.
- **Expected IP Prefix:** (Optional) A string prefix that at least one resolved IP must match (e.g. `203.0.113.` for a known CDN edge range). A mismatch flags the monitor as degraded but the HTTP step still runs.
- **Expected CNAME:** (Optional) The CNAME that the hostname must resolve to (e.g. `app.cdn.example.net.`). A mismatch is reported in metadata and flags the monitor as degraded.
- **Expected Status:** (Optional) A specific HTTP status that the response must return.
- **Expected Body:** (Optional) Substring that must appear in the HTTP response body.
- **Skip TLS Verification:** (Optional) Disable certificate validation for the HTTP step.

## Example Use Cases

- **CDN failover detection:** Pin `Expected CNAME` to your active edge — when traffic is silently steered elsewhere, the monitor flags the change before customers notice routing weirdness.
- **DNS propagation after migration:** After updating an `A` record, monitor the `Expected IP Prefix` for the new origin block to confirm the change has propagated.
- **End-to-end correctness:** Combine DNS expectations with an HTTP body assertion for the highest-confidence "this is serving the right thing" check.
