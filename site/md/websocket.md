# WebSocket Monitor

The WebSocket monitor in updu checks whether a WebSocket endpoint properly accepts HTTP `Upgrade` requests, proving the WebSocket server is online and accepting client connections.

## Configuration Options

When setting up a WebSocket monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for a connection response before considering the check failed.

### WebSocket Specific Settings

- **WebSocket URL:** The full `ws://` or `wss://` URL to monitor (e.g., `wss://chat.example.com/socket`).
- **Skip TLS Verification:** (Optional) Check this box if you are connecting to an internal `wss://` endpoint with a self-signed or invalid certificate and want to skip validation.

## Example Use Cases

- **Real-time Applications:** Ensure the WebSocket backend handling chat messages or live dashboard feeds is responsive.
