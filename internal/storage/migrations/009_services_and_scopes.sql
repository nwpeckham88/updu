-- 009_services_and_scopes.sql
-- Evolution to Services, Zones, Scopes, and TLS Certificates

-- 1. Zones table
CREATE TABLE IF NOT EXISTS zones (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO zones (id, name, description) VALUES 
    ('default', 'Default Zone', 'Primary local hosting zone');

-- 2. Scopes table
CREATE TABLE IF NOT EXISTS scopes (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO scopes (id, name, description) VALUES 
    ('lan', 'Local LAN', 'Private RFC1918 internal network'),
    ('tailnet', 'Tailscale Overlay', 'Encrypted WireGuard/Tailscale mesh'),
    ('public', 'Public Internet', 'Routable public internet / WAN');

-- 3. Node Scopes join table
CREATE TABLE IF NOT EXISTS node_scopes (
    node_id     TEXT NOT NULL,
    scope_id    TEXT NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (node_id, scope_id),
    FOREIGN KEY (scope_id) REFERENCES scopes(id) ON DELETE CASCADE
);

-- 4. Services table (Unified Core Entity)
CREATE TABLE IF NOT EXISTS services (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL, -- 'web', 'infra', 'database', 'host', 'job'
    zone_id         TEXT NOT NULL DEFAULT 'default',
    groups          JSON DEFAULT '[]',
    tags            JSON DEFAULT '[]',
    enabled         BOOLEAN NOT NULL DEFAULT 1,
    maintenance_id  TEXT,
    created_by      TEXT NOT NULL DEFAULT 'admin',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (zone_id) REFERENCES zones(id)
);
CREATE INDEX IF NOT EXISTS idx_services_zone ON services(zone_id);

-- 5. Service Endpoints table (Multiple probe paths per service)
CREATE TABLE IF NOT EXISTS service_endpoints (
    id              TEXT PRIMARY KEY,
    service_id      TEXT NOT NULL,
    name            TEXT NOT NULL,
    scope_id        TEXT NOT NULL DEFAULT 'public',
    target_type     TEXT NOT NULL, -- 'http', 'tcp', 'ping', 'dns', 'push'
    config          JSON NOT NULL,
    interval_s      INTEGER NOT NULL DEFAULT 30,
    timeout_s       INTEGER NOT NULL DEFAULT 10,
    retries         INTEGER NOT NULL DEFAULT 2,
    is_primary      BOOLEAN NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE,
    FOREIGN KEY (scope_id) REFERENCES scopes(id)
);
CREATE INDEX IF NOT EXISTS idx_endpoints_service ON service_endpoints(service_id);
CREATE INDEX IF NOT EXISTS idx_endpoints_scope ON service_endpoints(scope_id);

-- 6. Endpoint Check Results table
CREATE TABLE IF NOT EXISTS endpoint_checks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id  TEXT NOT NULL,
    endpoint_id TEXT NOT NULL,
    node_id     TEXT NOT NULL DEFAULT 'local',
    status      TEXT NOT NULL, -- 'up', 'down', 'degraded'
    latency_ms  INTEGER,
    status_code INTEGER,
    message     TEXT,
    metadata    JSON,
    checked_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE,
    FOREIGN KEY (endpoint_id) REFERENCES service_endpoints(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_endpoint_checks_recent 
ON endpoint_checks(endpoint_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_endpoint_checks_service 
ON endpoint_checks(service_id, checked_at DESC);

-- 7. Dedicated TLS Certificates table
CREATE TABLE IF NOT EXISTS tls_certificates (
    id                      TEXT PRIMARY KEY, -- SHA256 of leaf cert or domain
    domain                  TEXT NOT NULL,
    issuer                  TEXT NOT NULL,
    subject                 TEXT NOT NULL,
    sans                    JSON DEFAULT '[]',
    valid_from              DATETIME NOT NULL,
    valid_until             DATETIME NOT NULL,
    days_remaining          INTEGER NOT NULL,
    serial_number           TEXT,
    signature_algorithm     TEXT,
    ocsp_status             TEXT,
    last_verified_at        DATETIME NOT NULL,
    associated_endpoint_id  TEXT,
    FOREIGN KEY (associated_endpoint_id) REFERENCES service_endpoints(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_tls_certs_expiry ON tls_certificates(valid_until ASC);

-- 8. Backfill legacy monitors into services and service_endpoints
INSERT OR IGNORE INTO services (id, name, type, zone_id, groups, tags, enabled, created_by, created_at, updated_at)
SELECT 
    id, 
    name, 
    CASE 
        WHEN type IN ('http', 'https') THEN 'web'
        WHEN type = 'dns' THEN 'infra'
        WHEN type = 'tcp' THEN 'database'
        WHEN type = 'ping' THEN 'host'
        WHEN type = 'push' THEN 'job'
        ELSE 'web'
    END,
    'default',
    COALESCE(groups, '[]'),
    COALESCE(tags, '[]'),
    enabled,
    COALESCE(created_by, 'admin'),
    created_at,
    updated_at
FROM monitors;

INSERT OR IGNORE INTO service_endpoints (id, service_id, name, scope_id, target_type, config, interval_s, timeout_s, retries, is_primary, created_at)
SELECT 
    id || '-default',
    id,
    'Primary Endpoint',
    'public',
    type,
    config,
    interval_s,
    timeout_s,
    retries,
    1,
    created_at
FROM monitors;
