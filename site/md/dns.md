# DNS Monitor

The DNS monitor validates that a domain name resolves to the correct IP address or value.

## Configuration Options

When setting up a DNS monitor, you can configure the following options:

### Basic Settings
- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.

### DNS Specific Settings
- **Domain:** The domain name to query (e.g., `example.com`).
- **Record Type:** The type of DNS record to look up (A, AAAA, CNAME, TXT, MX).
- **Resolver Server:** (Optional) A specific DNS server to query instead of the system default (e.g., `8.8.8.8` or `1.1.1.1`).
- **Expected Value:** (Optional) The specific IP address or string that the record should return. If not provided, the check simply verifies that *any* record of the requested type exists.

## Example Use Cases

- **Domain Propagation:** Ensure your newly updated A record points to the correct new server IP.
- **Email Security Verification:** Monitor your domain's TXT records to ensure SPF/DKIM policies haven't been modified.
