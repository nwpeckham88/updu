# SSH Monitor

The SSH monitor attempts a TCP connection to an SSH port and, critically, performs the initial SSH handshake to verify that an SSH daemon is actually running and responding correctly, rather than just checking if the port is open.

## Configuration Options

When setting up an SSH monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should connect to the SSH server.
- **Timeout (seconds):** The maximum time updu will wait for the SSH handshake to complete.

### SSH Specific Settings

- **Host / IP:** The hostname or IP address of the server running SSH.
- **Port:** The port number the SSH daemon is listening on (default is 22).

> **Note:** The SSH monitor in updu *does not* require authentication credentials (passwords or private keys). It only verifies the initial protocol handshake to ensure the service is healthy, preventing security risks from storing credentials in your monitoring tool.

## Example Use Cases

- **Server Reachability:** Ensure your primary management pathway (SSH) is available on all your nodes in a cluster.
- **Detecting Hung Resources:** Sometimes a port is open but the underlying application has hung. The SSH handshake check confirms the daemon is actively processing connections.
