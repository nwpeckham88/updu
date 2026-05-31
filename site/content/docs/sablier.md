# Sablier Service State Monitor

The Sablier monitor in updu queries Sablier's direct service-state API at `GET /api/services/{service_name}`. Because it talks to Sablier's API port instead of the reverse-proxy wake-up path, it can report whether a service is `sleeping`, `starting`, or `ready` without waking the target container.

## Configuration Options

When setting up a Sablier monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should query Sablier.
- **Timeout (seconds):** Maximum time updu waits for the Sablier API response.

### Sablier Specific Settings

- **URL:** Base URL of the Sablier API (for example `http://sablier.internal:6660`). updu appends `/api/services/{service_name}` automatically.
- **Service Name:** The exact service name Sablier exposes in the provider state API.
- **Skip TLS Verify:** (Optional) Disable certificate verification when talking to Sablier over HTTPS. Use only with self-signed internal endpoints.

## State Mapping

The Sablier lifecycle maps into updu's monitor statuses like this:

- **`sleeping` → Up:** the service is intentionally scaled to zero and idle.
- **`starting` → Pending:** the service has been triggered and is still booting.
- **`ready` → Up:** the service is awake and serving traffic.
- **`ready` with `replicas: 0` → Degraded:** Sablier reported a contradictory state.
- **Unknown or malformed state → Down:** updu treats unexpected payloads as failures.

The latest check metadata also includes Sablier's raw lifecycle state, current replica count, desired replica count, and TTL.

## Example Use Cases

- **Scale-to-zero media apps:** Keep Jellyfin, Arr apps, or docs tools monitored even while Sablier has them asleep.
- **Warm-up tracking:** Distinguish between a sleeping service and one that is currently starting.
- **Proxy-independent status:** Confirm Sablier's provider state directly when debugging Traefik, Nginx Proxy Manager, or plugin-based wake-up flows.

## Example Configuration

```yaml
monitors:
  - id: sablier-media-state
    name: Sablier Media State
    type: sablier
    groups: [Platform]
    interval: 30s
    timeout: 5s
    config:
      url: http://sablier.internal.example:6660
      service_name: media
```

## Notes

- This monitor only observes Sablier's view of the service lifecycle. Pair it with an HTTP or HTTPS monitor if you also need end-user response validation after wake-up.
- If Sablier's API is unreachable or returns non-2xx responses, the check reports Down.
- Use the direct API port rather than a proxied application URL, otherwise the request can wake the service and defeat the point of the check.