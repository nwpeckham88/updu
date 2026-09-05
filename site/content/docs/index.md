# updu Documentation

Welcome to the documentation for **updu**! Here you will find detailed guides on how to configure and use the core monitor types available in updu.

## Overview

updu is a lightweight, self-hosted uptime monitoring solution designed for homelabs and infrastructure. It's built as a lean, single static binary with embedded SQLite that runs essentially anywhere — even on a Raspberry Pi Zero W — providing the core monitoring probes you need without external database bloat.

The dashboard leads with a clear verdict (operational, degraded, outage, or checks pending) and the monitor detail view gives you one-glance status, recent samples, and configuration in a focused dual-column layout.

## Supported Monitor Types

updu ships with 5 lean, battle-tested core probes:

- **[HTTP / HTTPS](/docs/http/index.html)** — Monitor web endpoints, status codes, response bodies, and TLS certificate expiration warnings.
- **[TCP Port](/docs/tcp/index.html)** — Verify services are accepting TCP connections on specific ports.
- **[DNS](/docs/dns/index.html)** — Validate domain resolution against custom or system resolvers.
- **[ICMP / Ping](/docs/icmp/index.html)** — Check low-level host reachability via ICMP echo.
- **[Push (Heartbeat)](/docs/push/index.html)** — Accept inbound heartbeats from cron jobs, backups, and external scripts (dead man's snitch).

## Notification Channels

updu ships with 3 built-in notification channels: Webhook, Discord, and ntfy. Channels are configured under **Settings → Notifications** and can be assigned per monitor.

## Configuration

All monitor configuration in updu can be managed through the embedded web dashboard or GitOps YAML. When you add a new monitor, the form shows only the fields relevant to the chosen type.

## Rebuilding these docs

These pages are generated from the markdown files in [`site/content/docs/`](https://github.com/nwpeckham88/updu/tree/main/site/content/docs) by `scripts/build-docs/`. To regenerate the HTML after editing the markdown:

```sh
make docs
```
