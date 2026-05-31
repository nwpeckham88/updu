# Push (Heartbeat) Monitor

The Push monitor in updu is a passive monitoring tool. Instead of updu actively checking a service endpoint, this monitor waits to receive a "ping" or heartbeat from your scripts, cron jobs, backups, or external services within a specified interval.

## Configuration Options

When setting up a Push monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Expected Interval (seconds):** How frequently updu expects to receive a heartbeat before marking the monitor down.
- **Grace Period (seconds):** Additional allowable delay on top of the interval before triggering a down incident.

### Push Specific Settings

- **Push Token:** A randomly generated API key that updu creates automatically when the monitor is saved. The recommended endpoint is `https://updu.yourdomain.com/heartbeat/YOUR_TOKEN`, which accepts `GET`, `POST`, and `PUT`. A legacy slug route is also available as `POST https://updu.yourdomain.com/api/v1/heartbeat/MONITOR_ID?token=YOUR_TOKEN`.

## Example Use Cases

- **Cron Job Monitoring:** Append `curl -fsS https://updu.yourdomain.com/heartbeat/YOUR_TOKEN` to the end of a scheduled bash backup script to be alerted when the backup fails to run.
- **Long-running Process Health:** Embed an HTTP request in a worker daemon to signal it is alive and processing events every few minutes.
