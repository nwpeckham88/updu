# updu Documentation

Welcome to the documentation for **updu**! Here you will find detailed guides on how to configure and use the various monitor types available in updu.

## Overview

updu is a lightweight, self-hosted uptime monitoring solution designed for homelabs and small infrastructure. It's built to run essentially anywhere — even on a Raspberry Pi Zero W — while providing the essential monitoring features you need without the bloat.

The dashboard leads with a clear verdict (operational, degraded, outage, or checks pending) and the monitor detail view gives you one-glance status, recent samples, and configuration in a focused dual-column layout.

## Supported Monitor Types

updu currently ships 19 supported monitor types: 15 core probes plus 4 advanced monitors. Every type listed below has its own dedicated guide.

### Core monitor guides

- **[HTTP / HTTPS](/docs/http/index.html)** — Monitor web endpoints, status codes, and response bodies.
- **[TCP Port](/docs/tcp/index.html)** — Verify services are accepting connections on specific ports.
- **[DNS](/docs/dns/index.html)** — Validate DNS record resolution.
- **[ICMP / Ping](/docs/icmp/index.html)** — Check low-level host reachability.
- **[SSH](/docs/ssh/index.html)** — Verify SSH connectivity to remote machines.
- **[SSL Certificate](/docs/ssl/index.html)** — Track certificate expiry dates.
- **[JSON API](/docs/api/index.html)** — Deep-check API responses by validating JSON fields.
- **[Push (Heartbeat)](/docs/push/index.html)** — Accept heartbeats from cron jobs, backups, and external scripts.
- **[WebSocket](/docs/websocket/index.html)** — Verify WebSocket and WSS connection upgrades.
- **[SMTP Server](/docs/smtp/index.html)** — Check mail server reachability and TLS support.
- **[UDP Port](/docs/udp/index.html)** — Send and receive UDP datagrams.
- **[Redis](/docs/redis/index.html)** — Verify Redis connectivity and authentication.
- **[PostgreSQL](/docs/postgres/index.html)** — Verify Postgres database connectivity.
- **[MySQL](/docs/mysql/index.html)** — Verify MySQL and MariaDB connectivity.
- **[MongoDB](/docs/mongo/index.html)** — Verify MongoDB connectivity.

### Advanced monitor guides

- **[HTTPS (with TLS Health)](/docs/https/index.html)** — Combine HTTP expectations with certificate freshness and warning thresholds in one monitor.
- **[Composite](/docs/composite/index.html)** — Roll up existing monitor IDs with `all_up`, `any_up`, or quorum logic.
- **[Transaction](/docs/transaction/index.html)** — Run sequential HTTP flows with per-step assertions and extracted response values.
- **[DNS + HTTP](/docs/dns_http/index.html)** — Validate DNS resolution first, then verify that the origin still responds as expected.

## Notification Channels

updu ships with five built-in notification channels: Webhook, Discord, Slack, Email (SMTP), and ntfy. Channels are configured under **Settings → Notifications** and can be assigned per monitor.

## Configuration

All monitor configuration in updu can be managed through the embedded web dashboard or GitOps YAML. When you add a new monitor, the form shows only the fields relevant to the chosen type. The guides on this page describe every supported type along with the example use cases that motivated each one.

## Rebuilding these docs

These pages are generated from the markdown files in [`site/md/`](https://github.com/nwpeckham88/updu/tree/main/site/md) by `scripts/build-docs/`. To regenerate the HTML after editing the markdown:

```sh
make docs
```
