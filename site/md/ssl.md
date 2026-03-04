# SSL Certificate Monitor

The SSL monitor tracks the health and expiration dates of TLS/SSL certificates on your domains. It provides advance warning before a certificate expires, helping prevent unexpected outages.

## Configuration Options

When setting up an SSL monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should check the certificate details. Since certificates don't change often, longer intervals (like 86400 seconds / 24 hours) are often sufficient, though checking more frequently is perfectly fine.

### SSL Specific Settings

- **Host / Domain:** The hostname to connect to (e.g., `updu.dev`).
- **Port:** The port where the TLS service is running (default is 443).
- **Expiration Threshold (days):** Important: This determines when the monitor transitions to a "Warning" or "Down" state. If the certificate expires in *fewer* days than this threshold, the monitor will alert you. (For example, setting this to `14` means you will be alerted 14 days before the certificate actually expires).

## Example Use Cases

- **Expiration Warnings:** Catch expiring Let's Encrypt certificates if an auto-renewal script fails.
- **Misconfigured Proxies:** Verify that your reverse proxy is still presenting the correct, valid certificate for a specific domain name.
