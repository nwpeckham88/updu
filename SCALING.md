# Scaling Guide

updu uses SQLite as its sole data store. This is a deliberate choice for simplicity
and single-binary deployment, but it comes with trade-offs. This document explains
the limits and how to operate comfortably within them.

## SQLite Configuration

The database is opened with these pragmas (see `internal/storage/sqlite.go`):

| Pragma | Value | Rationale |
|--------|-------|-----------|
| `journal_mode` | WAL | Allows concurrent reads during writes |
| `synchronous` | NORMAL | Faster than FULL; risk of losing last txn on power loss |
| `busy_timeout` | 5000 ms | Retries on SQLITE_BUSY for up to 5 seconds |
| `cache_size` | -1000 (1 MB) | Keeps RAM low on constrained devices |
| `temp_store` | FILE | Avoids RAM pressure on Pi Zero W |
| `mmap_size` | 0 (disabled) | Disabled for ARM compatibility |
| `MaxOpenConns` | 1 | Prevents SQLite multi-writer corruption |

## Practical Limits

### Monitor Count

| Monitors | Check Interval | Writes/min | Assessment |
|----------|---------------|------------|------------|
| 1–50 | 30s | ≤100 | Comfortable on any hardware |
| 50–200 | 30s | ≤400 | Fine on Pi 4 / any VPS |
| 200–500 | 60s | ≤500 | Approaching limit on low-end hardware |
| 500+ | 60s+ | 500+ | May see occasional busy timeouts |

SQLite's write throughput on typical hardware is roughly **500–2,000 transactions/sec**
for small inserts. Each check produces one `INSERT` into `check_results` plus
occasional aggregation writes. The bottleneck is `MaxOpenConns=1`, which serializes
all database access through a single connection.

### Data Retention

- **Raw check results**: Purged after 30 days (background job runs daily)
- **5-minute aggregates**: Retained indefinitely, ~288 rows/monitor/day
- **Events/incidents**: Retained indefinitely

For 200 monitors over 30 days, expect roughly:
- Raw checks: `200 × 2/min × 60 × 24 × 30 ≈ 17M rows` (purged rolling)
- Aggregates: `200 × 288 × 30 ≈ 1.7M rows` (cumulative)

### Database Size

A typical deployment with 50 monitors will produce a database of **50–200 MB** after
several months. With 200+ monitors, expect **500 MB–1 GB**. The WAL file can
temporarily grow large during heavy write bursts; it is checkpointed automatically
by SQLite.

## Recommended Indexes

The default schema includes indexes on primary query patterns. If you observe
slow queries at scale, consider adding:

```sql
-- Speed up dashboard queries (latest check per monitor)
CREATE INDEX IF NOT EXISTS idx_check_results_monitor_checked
  ON check_results(monitor_id, checked_at DESC);

-- Speed up purge operations
CREATE INDEX IF NOT EXISTS idx_check_results_checked_at
  ON check_results(checked_at);

-- Speed up aggregate lookups
CREATE INDEX IF NOT EXISTS idx_check_aggregates_monitor_bucket
  ON check_aggregates(monitor_id, bucket_start DESC);
```

You can apply these via `sqlite3 data/updu.db < indexes.sql` while updu is
running (WAL mode supports concurrent reads).

## Tuning for Higher Scale

If you need to push beyond 200 monitors on constrained hardware:

1. **Increase check intervals** — Moving from 30s to 60s halves write volume.
2. **Increase `busy_timeout`** — Set `UPDU_BUSY_TIMEOUT` higher if you see
   occasional "database is locked" errors.
3. **Reduce purge window** — Modify the purge threshold in `cmd/updu/main.go`
   from 30 days to 7 or 14 days.
4. **Use faster storage** — SQLite throughput is I/O-bound. An SSD or tmpfs
   for the WAL file dramatically improves write performance.
5. **Enable mmap** — On 64-bit non-ARM systems, you can enable memory-mapped I/O
   by modifying the DSN to include `_pragma=mmap_size(268435456)` (256 MB).
   This improves read performance significantly.

## When to Outgrow SQLite

Consider migrating to PostgreSQL if you need:

- **More than ~500 actively-checked monitors** with sub-minute intervals
- **Multiple updu instances** (HA / geographic distribution)
- **Concurrent heavy API readers** during peak dashboard usage
- **Long-term analytics** on raw check data beyond 30 days

updu does not currently support PostgreSQL as a backend, but the storage layer
(`internal/storage/`) is designed around a `*sql.DB` abstraction that could be
extended to support it.

## Monitoring updu Itself

Use the built-in endpoints to watch for scaling issues:

- **`GET /healthz`** — Returns `200 OK` with database health and scheduler state.
  Returns `503` if the database is unreachable.
- **`GET /api/v1/metrics`** — Prometheus-compatible metrics including monitor counts,
  goroutines, memory usage, and GC stats. Point your Prometheus scrape config at this.
- **`GET /api/v1/system/metrics`** — JSON dashboard metrics (requires admin auth).

### Example Prometheus scrape config

```yaml
scrape_configs:
  - job_name: updu
    scrape_interval: 30s
    metrics_path: /api/v1/metrics
    static_configs:
      - targets: ['localhost:3000']
```
