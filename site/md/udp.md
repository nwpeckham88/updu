# UDP Port Monitor

The UDP monitor in updu enables connectionless health checks by sending a datagram to an endpoint and verifying that your service answers back, bypassing the TCP handshake.

## Configuration Options

When setting up a UDP monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for a response datagram before returning a timeout error.

### UDP Specific Settings

- **Hostname:** The IP address or domain name of the service (e.g., `192.168.1.5`).
- **Port:** The UDP port the server is listening on.
- **Send Payload:** (Optional) An ASCII string describing the payload to insert into the outgoing UDP datagram (e.g., `\xff\xff\xff\xffTSource Engine Query\x00` for a Steam game server).
- **Expected Response String:** (Optional) An ASCII string updu must find in the return packet payload from the server.

## Example Use Cases

- **Dedicated Game Servers:** Check whether older Source-engine games such as Counter-Strike or TF2 are accepting traffic and player queries.
