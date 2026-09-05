# Agent Guidelines for updu

## 1. Version Comparison & Self-Update Lifecycle
- **SemVer 2.0 Prerelease Precedence**: When implementing version comparison, never rely solely on `major.minor.patch` or simple string checks. Adhere to SemVer 2.0 precedence: numeric parts compare numerically, alphanumeric parts compare lexicographically, and normal releases take precedence over prereleases.
- **The Updater Bootstrap Catch-22**: Remember that the *currently running* instance evaluates `update_available` using its own compiled-in code. If an in-app updater bug exists for a given patch (e.g., `0.8.0-beta.N`), pushing `0.8.0-beta.N+1` will not automatically make the button appear on running nodes. Fixes intended to be pulled via the UI must either bump `patch` (e.g., `0.8.1-beta.1`) so `lp > cp` in older binaries, or include explicit manual binary upgrade instructions.

## 2. Defensive Operational UI Actions
- **Never Hide Critical Actions Behind Fragile Booleans**: UI action buttons for critical operations (system updates, node pairing, service failovers) must not be rendered exclusively behind server booleans like `update_available`.
- **Provide Fallback & Force Overrides**: If `latest_version !== current_version`, always provide the primary action button. Even when versions are identical, provide a secondary option (e.g. "Reinstall / Force Update") backed by `?force=true` in the API.

## 3. SQLite Concurrency & Connection Pool
- **Single-Connection WAL Deadlocks**: `storage.Open` sets `db.SetMaxOpenConns(1)` for SQLite concurrency safety.
- **Sequential Scan Rule**: Never execute queries or database transactions while an `sql.Rows` iterator is open. Always read all rows into memory, call `rows.Close()`, and only then perform subsequent queries.

## 4. Safe HTTP Clients & SSRF Guardrails
- **Outbound HTTP Probing**: Outbound checkers and notifiers block loopback/localhost (`127.0.0.1`, `::1`) by default for SSRF defense.
- **Testing Guardrail**: When testing checkers or notifier channels against local `httptest.Server` instances, always inject `context.WithValue(ctx, channels.AllowLocalhostKey, true)`.
