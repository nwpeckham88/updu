# PostgreSQL Monitor

The PostgreSQL monitor in updu provides connection-level checks and validates that a PostgreSQL RDBMS server is reachable, authenticating, and accepting connections properly.

## Configuration Options

updu supports two parsing strategies for Postgres connection properties: a unified driver connection string (DSN URL), or individual connection parameters declared manually.

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for a response before considering the check failed.

### PostgreSQL Specific Settings

- **Connection String (DSN):** A full URL connection string combining the necessary parameters; the other fields can stay blank when this is provided. Format example: `postgres://username:password@localhost:5432/database_name`. updu establishes a TCP connection to this destination.

### Fallback Individual Settings

- **Hostname:** The IP address or domain name of the Postgres server (e.g., `localhost`).
- **Port:** The connection port the server exposes (standard is `5432`).
- **User / Password:** Authentication details for the requested instance.
- **Database:** The specific database catalog name to query.
- **SSL Mode:** Fine-tune the `sslmode` query parameter, including `disable`, `require`, `verify-ca`, and `verify-full`.

## Example Use Cases

- **Postgres Outages:** Ensure critical web applications backed by a remote SQL RDBMS don't crash or lock up without emitting alerts.
