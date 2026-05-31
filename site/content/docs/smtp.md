# SMTP Server Monitor

The SMTP monitor in updu tests connections to an email server. It establishes a TCP connection to the host and port and verifies that the server responds with a standard SMTP `220` greeting indicating it is ready.

## Configuration Options

When setting up an SMTP monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for a response before considering the check failed.

### SMTP Specific Settings

- **Hostname:** The IP address or domain name of the email server (e.g., `smtp.gmail.com`).
- **Port:** The port the email server is listening on. Standard ports are `25` (unencrypted), `465` (SMTPS), or `587` (STARTTLS).
- **Require TLS:** (Optional) Check this box if the server requires an initial TLS handshake (e.g., port 465). This determines how updu builds the initial connection.

## Example Use Cases

- **Mail Relay Availability:** Monitor your self-hosted Postfix or external email gateway to catch hangs and firewall drops that would otherwise silently break delivery.
