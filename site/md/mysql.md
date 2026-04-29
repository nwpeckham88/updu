# MySQL / MariaDB Monitor

The MySQL monitor in updu actively validates that a MySQL or MariaDB RDBMS connection can authenticate and respond properly.

## Configuration Options

Much like the PostgreSQL monitor, you can specify an entire connection string or break the connection inputs out field by field, depending on your preference.

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for an authenticated connection before returning offline.

### MySQL Specific Settings

- **Connection String (DSN):** A full URL connection string combining the necessary parameters in the standard `go-sql-driver` structure. Leave the other config fields empty when using this format: `username:password@tcp(host:port)/dbname?timeout=5s`.

### Fallback Individual Settings

- **Hostname:** The IP address or domain name of the MySQL server (e.g., `localhost`).
- **Port:** The connection port the server exposes (standard is `3306`).
- **User / Password:** Authentication details for the requested instance.
- **Database:** The specific database catalog name to query.

## Example Use Cases

- **Relational DB Health Check:** Ping the core MariaDB datastore behind a Nextcloud instance to catch performance degradation instantly.
