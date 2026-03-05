# TCP Monitor

The TCP monitor checks if a specific port on a server is open and accepting connections.

## Configuration Options

When setting up a TCP monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for a connection before considering the check failed.

### TCP Specific Settings

- **Host / IP:** The hostname or IP address of the server (e.g., `192.168.1.100` or `db.example.com`).
- **Port:** The specific network port to check (e.g., `3306` for MySQL, `5432` for PostgreSQL).

## Example Use Cases

- **Database Availability:** Ensure your internal database server is accepting connections on its standard port.
- **Network Services:** Check if an FTP server is running on port `21`.
