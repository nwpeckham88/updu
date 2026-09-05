CREATE TABLE IF NOT EXISTS peers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    public_key TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'peer',
    status TEXT NOT NULL DEFAULT 'pending',
    last_seen DATETIME,
    metadata JSON,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
