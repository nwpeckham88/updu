-- 001_initial.sql
-- Initial schema for updu

CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    username    TEXT UNIQUE NOT NULL,
    password    TEXT,
    role        TEXT NOT NULL DEFAULT 'viewer',
    oidc_sub    TEXT,
    oidc_issuer TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_agent  TEXT,
    ip_addr     TEXT,
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS monitors (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    config      JSON NOT NULL,
    group_name  TEXT,
    tags        JSON,
    interval_s  INTEGER NOT NULL DEFAULT 60,
    timeout_s   INTEGER NOT NULL DEFAULT 10,
    retries     INTEGER NOT NULL DEFAULT 3,
    enabled     BOOLEAN NOT NULL DEFAULT 1,
    parent_id   TEXT REFERENCES monitors(id) ON DELETE SET NULL,
    created_by  TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS check_results (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id  TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status      TEXT NOT NULL,
    latency_ms  INTEGER,
    status_code INTEGER,
    message     TEXT,
    metadata    JSON,
    checked_at  DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_checks_monitor_time ON check_results(monitor_id, checked_at);

CREATE TABLE IF NOT EXISTS check_aggregates (
    monitor_id   TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    period_start DATETIME NOT NULL,
    resolution   TEXT NOT NULL,
    total_checks INTEGER NOT NULL,
    up_count     INTEGER NOT NULL,
    down_count   INTEGER NOT NULL,
    avg_latency  REAL,
    min_latency  INTEGER,
    max_latency  INTEGER,
    uptime_pct   REAL,
    PRIMARY KEY (monitor_id, period_start, resolution)
);

CREATE TABLE IF NOT EXISTS incidents (
    id           TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    description  TEXT,
    status       TEXT NOT NULL DEFAULT 'investigating',
    severity     TEXT NOT NULL DEFAULT 'minor',
    monitor_ids  JSON,
    started_at   DATETIME NOT NULL,
    resolved_at  DATETIME,
    created_by   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS maintenance_windows (
    id           TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    monitor_ids  JSON NOT NULL,
    starts_at    DATETIME NOT NULL,
    ends_at      DATETIME NOT NULL,
    recurring    TEXT,
    created_by   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS notification_channels (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    type         TEXT NOT NULL,
    config       JSON NOT NULL,
    enabled      BOOLEAN DEFAULT 1
);

CREATE TABLE IF NOT EXISTS heartbeats (
    slug         TEXT PRIMARY KEY,
    monitor_id   TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    last_ping    DATETIME,
    expected_s   INTEGER NOT NULL,
    grace_s      INTEGER NOT NULL DEFAULT 300
);

CREATE TABLE IF NOT EXISTS status_pages (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    slug         TEXT UNIQUE NOT NULL,
    description  TEXT,
    groups       JSON NOT NULL DEFAULT '[]',
    is_public    BOOLEAN DEFAULT 1,
    password     TEXT
);
