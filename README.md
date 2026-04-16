# updu

Lightweight, self-hosted uptime monitoring in a single binary. Designed to run
on anything from a Raspberry Pi Zero W to a cloud VM.

## Features

- **15 monitor types** — HTTP, JSON API, TCP, ICMP Ping, DNS, SSL, SSH, Push/Heartbeat, WebSocket, SMTP, UDP, Redis, PostgreSQL, MySQL, MongoDB
- **5 notification channels** — Webhook, Discord, Slack, Email (SMTP), Ntfy
- **Public status pages** — Custom slugs, grouped monitors, custom CSS
- **Incident management** — Severity levels, status progression, per-monitor tracking
- **Maintenance windows** — One-time or recurring windows to suppress alerts during planned work
- **GitOps** — Declarative YAML config (`updu.conf`) with deterministic IDs
- **Real-time dashboard** — SSE-powered live updates, no polling
- **Single binary** — Go backend + embedded SvelteKit SPA, zero runtime dependencies
- **SQLite** — No external database required; WAL mode, tuned for low-resource devices
- **OIDC** — Optional SSO via build tag
- **Self-update** — One-click updates from GitHub Releases with checksum verification
- **Prometheus metrics** — `GET /api/v1/metrics` exposes monitor, runtime, and incident gauges
- **Health check** — `GET /healthz` for load balancers, Docker, and Kubernetes probes

## Quick Start

### Binary

```bash
# Download the latest release for your platform
curl -LO https://github.com/nwpeckham88/updu/releases/latest/download/updu-linux-amd64
chmod +x updu-linux-amd64
./updu-linux-amd64
```

Open `http://localhost:3000` and register your admin account.

### Docker

```bash
docker run -d \
  --name updu \
  -p 3000:3000 \
  -v updu-data:/data \
  -e UPDU_AUTH_SECRET=$(openssl rand -hex 32) \
  ghcr.io/nwpeckham88/updu:latest
```

### Docker Compose

```bash
cp docker-compose.yml .
echo "UPDU_AUTH_SECRET=$(openssl rand -hex 32)" > .env
docker compose up -d
```

### systemd

```bash
sudo ./updu install
sudo systemctl start updu
```

The installer writes a hardened unit that runs from the current binary directory.
For a cleaner service setup, place the binary in a dedicated directory such as `/opt/updu`
before running `install`.

## Configuration

updu uses a three-tier config: defaults → YAML → environment variables (highest priority).

| Variable | Default | Description |
|----------|---------|-------------|
| `UPDU_PORT` | `3000` | Listen port |
| `UPDU_HOST` | `0.0.0.0` | Bind address |
| `UPDU_DB_PATH` | `./data/updu.db` | SQLite database path |
| `UPDU_AUTH_SECRET` | *(auto-generated)* | Session signing key (set for persistence across restarts) |
| `UPDU_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `UPDU_BASE_URL` | `http://localhost:3000` | Public URL (for OIDC redirects, links) |
| `UPDU_SESSION_TTL_DAYS` | `7` | Session cookie lifetime |
| `UPDU_WORKER_POOL_SIZE` | `0` (auto) | Concurrent check workers (auto = CPU×4, clamped 4–50) |
| `UPDU_MIN_INTERVAL_S` | `30` | Minimum allowed check interval |
| `UPDU_ALLOW_LOCALHOST` | `false` | Allow monitors to target `127.0.0.1` / `localhost` |
| `UPDU_ENABLE_CUSTOM_CSS` | `false` | Allow custom CSS on status pages |

See [sample.updu.conf](sample.updu.conf) for a full YAML configuration example with all 15 monitor types.

## GitOps

Drop one or more `*.updu.conf` files in the working directory (or set `UPDU_CONF_PATH`):

```yaml
host: 0.0.0.0
port: 3000
log_level: info

monitors:
  - name: "Production API"
    type: "http"
    groups: ["Backend"]
    interval: "30s"
    config:
      url: "https://api.example.com/health"
      expected_status: 200

  - name: "Database"
    type: "postgres"
    groups: ["Infrastructure"]
    interval: "1m"
    config:
      host: "db.internal"
      port: 5432
      user: "monitor"
      database: "app"
```

Monitors are synced on startup with deterministic IDs (SHA256 of name+type), so config changes are idempotent.

## API

All endpoints are under `/api/v1/`. Authentication is cookie-based (session token).

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /healthz` | — | Health check (DB, scheduler, SSE) |
| `GET /api/v1/metrics` | bearer token when `UPDU_METRICS_TOKEN` is set | Prometheus metrics |
| `GET /api/v1/status-pages/{slug}` | — | Public status page |
| `POST /api/v1/heartbeat/{slug}` | — | Push monitor heartbeat |
| `GET\|POST\|PUT /heartbeat/{token}` | — | Simplified heartbeat endpoint |
| `GET /api/v1/monitors` | user | List monitors |
| `POST /api/v1/monitors` | admin | Create monitor |
| `GET /api/v1/dashboard` | user | Dashboard with recent checks |
| `GET /api/v1/stats` | user | Analytics (uptime, P95 latency, timeline) |
| `GET /api/v1/events` | user | SSE real-time event stream |
| **Full CRUD** | admin | Monitors, status pages, incidents, maintenance, notifications, users, settings |
| `GET /api/v1/system/backup` | admin | Export config (JSON) |
| `GET /api/v1/system/export/yaml` | admin | Export config (YAML/GitOps) |
| `POST /api/v1/system/backup` | admin | Import config |

## Building from Source

```bash
# Prerequisites: Go 1.26+, Node.js 20+, pnpm

# Full build (frontend + backend)
make build

# Cross-compile for all architectures
make build-amd64    # AMD64 (Generic Linux / VPS)
make build-arm      # ARMv6 (Pi Zero W)
make build-armv7    # ARMv7 (Pi 2/3)
make build-arm64    # ARM64 (Pi 3/4/5)

# With OIDC support (adds SSO, larger binary)
make build-amd64-oidc
make build-arm-oidc
make build-armv7-oidc
make build-arm64-oidc

# Build everything at once
make build-all

# Development
make dev-backend    # Go backend on :3000
make dev-frontend   # SvelteKit dev server with Vite proxy

# Local browser E2E
pnpm --dir frontend install
pnpm --dir frontend run test:e2e:install
make e2e-frontend
```

The local E2E target builds the embedded frontend, starts the real Go binary with a disposable SQLite database, launches a local fixture server for deterministic monitor checks, and runs the Playwright suite against the live app.

The suite currently covers:

- login, session persistence, and logout
- monitor list search, sorting, and empty state behavior
- monitor CRUD through the UI against the real API
- edit failure handling for monitors
- settings, incidents, and public status page smoke flows

## Architecture

```
cmd/updu/           → Entrypoint, CLI subcommands, embedded frontend
internal/
  api/              → REST API handlers, auth middleware, rate limiting
  auth/             → bcrypt auth, sessions, RBAC (admin/viewer), OIDC
  checker/          → 15 monitor implementations + SSRF protection
  config/           → Three-tier config loading, GitOps YAML parser
  models/           → Domain types (Monitor, Event, Incident, StatusPage, …)
  notifier/         → Dispatcher + 5 channel implementations
  realtime/         → SSE hub for live dashboard updates
  scheduler/        → Worker pool, jitter, stagger, retry, maintenance-aware
  storage/          → SQLite (WAL), embedded migrations, aggregator, GitOps sync
  updater/          → Self-update from GitHub releases
  version/          → Build-time version injection
frontend/           → SvelteKit (Svelte 5) + TailwindCSS v4
site/               → Landing page (updu.dev)
```

## Scaling

See [SCALING.md](SCALING.md) for SQLite limits, recommended indexes, tuning guidance, and Prometheus integration.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and pull request process.

## License

MIT
