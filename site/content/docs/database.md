# Database Monitor

The Database monitor in updu provides connection-level health checks and validates that a database server is reachable, authenticating properly, and accepting query commands. It unifies PostgreSQL, MySQL, and Redis monitoring into a single monitor type.

## Configuration Options

When setting up a Database monitor, choose the appropriate **Engine** (`postgres`, `mysql`, or `redis`) and supply the connection details.

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for a response before considering the check failed.

### PostgreSQL & MySQL Settings

For relational engines (`postgres` or `mysql`), updu supports two strategies for declaring connection parameters:

1. **Connection String (DSN):** A unified connection URI. Leave individual fields blank when using this format.
   - **PostgreSQL DSN format:** `postgres://username:password@localhost:5432/database_name?sslmode=disable`
   - **MySQL DSN format:** `username:password@tcp(localhost:3306)/dbname`
2. **Fallback Individual Settings:** Declare individual parameters separately.
   - **Hostname / IP:** The network address of the database server.
   - **Port:** The port the service is listening on (default `5432` for Postgres, `3306` for MySQL).
   - **User / Password:** Authentication credentials.
   - **Database:** The database name/catalog to target.
   - **SSL Mode:** (PostgreSQL only) Controls TLS encryption requirements (e.g. `disable`, `require`, `verify-ca`, `verify-full`).

### Redis Settings

For `redis` engine checks, the following options are supported:

- **Hostname / IP:** The address of your Redis instance (e.g., `localhost` or `cache.internal`).
- **Port:** The port your Redis instance is running on (default `6379`).
- **Auth Password:** (Optional) Provide the password if your instance requires the `AUTH` command to accept connections.
- **Database Index:** (Optional) The integer database index (e.g., `0`, `1`) updu will run `SELECT` against to confirm availability.

---

## Behavior & Inner Workings

- **SQL Engines (`postgres`, `mysql`):** updu attempts to establish an authenticated connection, issues a driver ping, and then executes a lightweight `SELECT 1` query to verify that the query processing engine is fully functional.
- **In-Memory Engine (`redis`):** updu opens a raw TCP connection, communicates directly using the RESP (Redis Serialization Protocol), runs `AUTH` if a password is configured, runs `SELECT` if a non-zero database index is targeted, and finally verifies responsiveness by asserting the expected `+PONG` response to a `PING` command.

## Example Use Cases

- **Application SQL Backends:** Ensure the PostgreSQL or MySQL database powering your API or CMS is accepting queries. If the DB locks up or runs out of connections, get alerted before users notice.
- **Job Queue Cache:** Watch your Redis cache/broker server to avoid background job pipelines stopping silently due to authentication changes or capacity limits.
