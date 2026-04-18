# updu Documentation

Welcome to the documentation for **updu**! Here you will find detailed guides on how to configure and use the various monitor types available in updu.

## Overview

updu is a lightweight, self-hosted uptime monitoring solution designed specifically for homelabs and small infrastructure. It’s built to run essentially anywhere (even on a Raspberry Pi Zero W) while providing all the essential monitoring features you need without the bloat.

## Supported Monitor Types

updu currently ships 19 supported monitor types: 15 core probes plus 4 advanced monitors.

### Core monitor guides

The 15 core monitor types have dedicated guides:

- **[HTTP / HTTPS](/docs/http/index.html)** - Monitor web endpoints, status codes, and response bodies.
- **[TCP Port](/docs/tcp/index.html)** - Verify services are accepting connections on specific ports.
- **[DNS](/docs/dns/index.html)** - Validate DNS record resolution.
- **[ICMP / Ping](/docs/icmp/index.html)** - Check low-level host reachability.
- **[SSH](/docs/ssh/index.html)** - Verify SSH connectivity to remote machines.
- **[SSL Certificate](/docs/ssl/index.html)** - Track certificate expiry dates.
- **[JSON API](/docs/api/index.html)** - Deep-check API responses by validating JSON fields.
- **[Push (Heartbeat)](/docs/push/index.html)** - Accept heartbeats from cron jobs, backups, and external scripts.
- **[WebSocket](/docs/websocket/index.html)** - Verify WebSocket and WSS connection upgrades.
- **[SMTP Server](/docs/smtp/index.html)** - Check mail server reachability and TLS support.
- **[UDP Port](/docs/udp/index.html)** - Send and receive UDP datagrams.
- **[Redis](/docs/redis/index.html)** - Verify Redis connectivity and authentication.
- **[PostgreSQL](/docs/postgres/index.html)** - Verify Postgres database connectivity.
- **[MySQL](/docs/mysql/index.html)** - Verify MySQL and MariaDB connectivity.
- **[MongoDB](/docs/mongo/index.html)** - Verify MongoDB connectivity.

### Advanced monitor types

These advanced monitors are available in the dashboard, API, and GitOps config today. They do not have standalone doc pages yet; use these summaries alongside the create and edit forms in the app:

- **HTTPS** - Combine HTTP expectations with TLS certificate freshness and warning thresholds in one monitor. The core HTTP guide covers shared request options; this advanced monitor adds certificate-aware behavior.
- **Composite** - Roll up existing monitor IDs with `all_up`, `any_up`, or quorum logic.
- **Transaction** - Run sequential HTTP flows with per-step assertions and extracted response values.
- **DNS + HTTP** - Validate DNS resolution first, then verify that the origin still responds as expected.

## Notification Channels

updu ships with five built-in notification channels: Webhook, Discord, Slack, Email (SMTP), and ntfy.

## Configuration

All monitor configuration in updu can be managed through the embedded web dashboard or GitOps YAML. When you add a new monitor, you'll be presented with the fields relevant to that monitor type. The guides on this page explain the 15 core monitor types, and the advanced monitor summaries map directly to the create and edit forms in the app.
