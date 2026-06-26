# WHOIS (Domain Expiry) Monitor

The WHOIS monitor in updu tracks domain name registration lifetimes and alerts you when your domain name is approaching its expiration date, helping prevent unexpected domain loss or hijackings.

## Configuration Options

When setting up a WHOIS monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check (typically set to daily or weekly, e.g. `86400` or `604800` seconds).
- **Timeout (seconds):** The maximum time updu will wait for a response before considering the check failed.

### WHOIS Specific Settings

- **Domain:** The domain name to query (e.g. `example.com` or `myblog.org`). Protocols (`http://`, `https://`) and URI paths/ports are automatically stripped out.
- **Days Before Expiry:** The threshold in days before the expiration date to mark the monitor status as **Degraded** and send alert notifications. Defaults to `14` days if left blank.

---

## Behavior & Inner Workings

The WHOIS check goes through three logical steps:

1. **Resolve Authoritative WHOIS Server:** updu extracts the TLD from the target domain, dials the root registry at `whois.iana.org` on TCP port `43`, and queries for the TLD record to extract the authoritative WHOIS server address (e.g. `whois.verisign-grs.com` for `.com`). If IANA fails to respond, it uses a fallback domain guess.
2. **Query WHOIS Registry:** updu dials the resolved authoritative WHOIS server on TCP port `43` and queries the domain directly.
3. **Parse Expiry Dates:** The returned raw text is parsed against a series of common date headers and patterns. If the domain is expired, it returns **Down**. If the domain's remaining days are less than or equal to the "Days Before Expiry" threshold, it returns **Degraded** with an alert message. Otherwise, it returns **Up**.

## Example Use Cases

- **Domain Registrations:** Protect critical marketing domains or SaaS hostnames from expiring. Domain registrars can send warning emails that land in spam folders; checking domain expiry via WHOIS ensures you receive alerts through Slack, Discord, or Email notifications directly.
