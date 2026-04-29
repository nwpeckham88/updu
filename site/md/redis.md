# Redis Monitor

The Redis monitor in updu speaks the RESP (REdis Serialization Protocol) directly. It authenticates and runs a `PING` command against the server to ensure the in-memory data store is responsive and actively processing commands.

## Configuration Options

When setting up a Redis monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for an authenticated connection before considering the check failed.

### Redis Specific Settings

- **Hostname:** The IP address or domain name of your Redis instance (e.g., `cache.example.com`).
- **Port:** The port your Redis instance is running on (standard is `6379`).
- **Auth Password:** (Optional) Provide the password if your instance requires the `AUTH` command to accept connections.
- **Database Index:** (Optional) The integer database index updu will run `SELECT` against to confirm availability. Defaults to `0`.

## Example Use Cases

- **Background Job Queues:** Be alerted when the cache server holding in-flight Celery or Sidekiq jobs goes offline, preventing application hangs before they happen.
