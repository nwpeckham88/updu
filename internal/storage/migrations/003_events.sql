-- 003_events.sql
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id TEXT NOT NULL,
    status TEXT NOT NULL,
    message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_events_monitor_id ON events(monitor_id);
