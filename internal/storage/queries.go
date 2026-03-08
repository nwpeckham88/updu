package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/updu/updu/internal/models"
)

// --- User Queries ---

func (db *DB) CreateUser(ctx context.Context, u *models.User) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, username, password, role, oidc_sub, oidc_issuer, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.Password, u.Role, u.OIDCSub, u.OIDCIssuer, u.CreatedAt,
	)
	return err
}

func (db *DB) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	u := &models.User{}
	err := db.QueryRowContext(ctx,
		`SELECT id, username, password, role, oidc_sub, oidc_issuer, created_at
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.OIDCSub, &u.OIDCIssuer, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (db *DB) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	u := &models.User{}
	err := db.QueryRowContext(ctx,
		`SELECT id, username, password, role, oidc_sub, oidc_issuer, created_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.OIDCSub, &u.OIDCIssuer, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (db *DB) GetUserByOIDCSub(ctx context.Context, sub, issuer string) (*models.User, error) {
	u := &models.User{}
	err := db.QueryRowContext(ctx,
		`SELECT id, username, password, role, oidc_sub, oidc_issuer, created_at
		 FROM users WHERE oidc_sub = ? AND oidc_issuer = ?`, sub, issuer,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.OIDCSub, &u.OIDCIssuer, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (db *DB) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (db *DB) ListUsers(ctx context.Context) ([]*models.User, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, username, role, oidc_sub, oidc_issuer, created_at FROM users ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.OIDCSub, &u.OIDCIssuer, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *DB) UpdateUserRole(ctx context.Context, id string, role models.UserRole) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET role = ? WHERE id = ?", role, id)
	return err
}

func (db *DB) DeleteUser(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}

// UpdateUserPassword updates a user's password hash.
func (db *DB) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET password = ? WHERE id = ?", passwordHash, id)
	return err
}

// --- Session Queries ---

func (db *DB) CreateSession(ctx context.Context, s *models.Session) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, user_agent, ip_addr, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.UserAgent, s.IPAddr, s.ExpiresAt, s.CreatedAt,
	)
	return err
}

func (db *DB) GetSession(ctx context.Context, id string) (*models.Session, error) {
	s := &models.Session{}
	err := db.QueryRowContext(ctx,
		`SELECT id, user_id, user_agent, ip_addr, expires_at, created_at
		 FROM sessions WHERE id = ? AND expires_at > ?`, id, time.Now(),
	).Scan(&s.ID, &s.UserID, &s.UserAgent, &s.IPAddr, &s.ExpiresAt, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (db *DB) DeleteSession(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	return err
}

func (db *DB) CleanExpiredSessions(ctx context.Context) error {
	_, err := db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at <= ?", time.Now())
	return err
}

// --- Monitor Queries ---

func (db *DB) CreateMonitor(ctx context.Context, m *models.Monitor) error {
	tags, _ := json.Marshal(m.Tags)
	groups, _ := json.Marshal(m.Groups)
	_, err := db.ExecContext(ctx,
		`INSERT INTO monitors (id, name, type, config, groups, tags, interval_s, timeout_s, retries, enabled, parent_id, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Name, m.Type, m.Config, string(groups), string(tags),
		m.IntervalS, m.TimeoutS, m.Retries, m.Enabled, m.ParentID,
		m.CreatedBy, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

func (db *DB) GetMonitor(ctx context.Context, id string) (*models.Monitor, error) {
	m := &models.Monitor{}
	var tagsJSON, groupsJSON string
	err := db.QueryRowContext(ctx,
		`SELECT id, name, type, config, groups, tags, interval_s, timeout_s, retries, enabled, parent_id, created_by, created_at, updated_at
		 FROM monitors WHERE id = ?`, id,
	).Scan(&m.ID, &m.Name, &m.Type, &m.Config, &groupsJSON, &tagsJSON,
		&m.IntervalS, &m.TimeoutS, &m.Retries, &m.Enabled, &m.ParentID,
		&m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if tagsJSON != "" {
		_ = json.Unmarshal([]byte(tagsJSON), &m.Tags)
	}
	if groupsJSON != "" {
		_ = json.Unmarshal([]byte(groupsJSON), &m.Groups)
	}
	return m, nil
}

func (db *DB) ListMonitors(ctx context.Context) ([]*models.Monitor, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, name, type, config, groups, tags, interval_s, timeout_s, retries, enabled, parent_id, created_by, created_at, updated_at
		 FROM monitors ORDER BY json_extract(groups, '$[0]'), name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []*models.Monitor
	for rows.Next() {
		m := &models.Monitor{}
		var tagsJSON, groupsJSON string
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.Config, &groupsJSON, &tagsJSON,
			&m.IntervalS, &m.TimeoutS, &m.Retries, &m.Enabled, &m.ParentID,
			&m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		if tagsJSON != "" {
			_ = json.Unmarshal([]byte(tagsJSON), &m.Tags)
		}
		if groupsJSON != "" {
			_ = json.Unmarshal([]byte(groupsJSON), &m.Groups)
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

func (db *DB) UpdateMonitor(ctx context.Context, m *models.Monitor) error {
	tags, _ := json.Marshal(m.Tags)
	groups, _ := json.Marshal(m.Groups)
	_, err := db.ExecContext(ctx,
		`UPDATE monitors SET name=?, type=?, config=?, groups=?, tags=?, interval_s=?, timeout_s=?, retries=?, enabled=?, parent_id=?, updated_at=?
		 WHERE id=?`,
		m.Name, m.Type, m.Config, string(groups), string(tags),
		m.IntervalS, m.TimeoutS, m.Retries, m.Enabled, m.ParentID,
		time.Now(), m.ID,
	)
	return err
}

func (db *DB) DeleteMonitor(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM monitors WHERE id = ?", id)
	return err
}

func (db *DB) GetMonitorTags(ctx context.Context, id string) ([]string, error) {
	m, err := db.GetMonitor(ctx, id)
	if err != nil || m == nil {
		return nil, err
	}
	return m.Tags, nil
}

func (db *DB) ListGroups(ctx context.Context) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT DISTINCT value FROM monitors, json_each(groups) WHERE value IS NOT NULL AND value != '' ORDER BY value")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (db *DB) RenameGroup(ctx context.Context, oldName, newName string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE monitors SET groups = COALESCE((
			SELECT json_group_array(CASE WHEN value = ? THEN ? ELSE value END)
			FROM json_each(monitors.groups)
		), '[]') WHERE id IN (
			SELECT m.id FROM monitors m, json_each(m.groups) WHERE value = ?
		)`, oldName, newName, oldName)
	return err
}

func (db *DB) DeleteGroup(ctx context.Context, name string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE monitors SET groups = COALESCE((
			SELECT json_group_array(value)
			FROM json_each(monitors.groups)
			WHERE value != ?
		), '[]') WHERE id IN (
			SELECT m.id FROM monitors m, json_each(m.groups) WHERE value = ?
		)`, name, name)
	return err
}

// --- Check Result Queries ---

func (db *DB) InsertCheckResult(ctx context.Context, r *models.CheckResult) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO check_results (monitor_id, status, latency_ms, status_code, message, metadata, checked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.MonitorID, r.Status, r.LatencyMs, r.StatusCode, r.Message, r.Metadata, r.CheckedAt,
	)
	return err
}

func (db *DB) GetRecentChecks(ctx context.Context, monitorID string, limit int) ([]*models.CheckResult, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, monitor_id, status, latency_ms, status_code, message, COALESCE(metadata, '{}'), checked_at
		 FROM check_results WHERE monitor_id = ? ORDER BY checked_at DESC LIMIT ?`,
		monitorID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.CheckResult
	for rows.Next() {
		r := &models.CheckResult{}
		var metadata []byte
		if err := rows.Scan(&r.ID, &r.MonitorID, &r.Status, &r.LatencyMs, &r.StatusCode, &r.Message, &metadata, &r.CheckedAt); err != nil {
			return nil, err
		}
		r.Metadata = metadata
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) GetLatestCheck(ctx context.Context, monitorID string) (*models.CheckResult, error) {
	r := &models.CheckResult{}
	err := db.QueryRowContext(ctx,
		`SELECT id, monitor_id, status, latency_ms, status_code, message, checked_at
		 FROM check_results WHERE monitor_id = ? ORDER BY checked_at DESC LIMIT 1`, monitorID,
	).Scan(&r.ID, &r.MonitorID, &r.Status, &r.LatencyMs, &r.StatusCode, &r.Message, &r.CheckedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// GetUptimePercent calculates uptime percentage for a monitor over a time range.
func (db *DB) GetUptimePercent(ctx context.Context, monitorID string, since time.Time) (float64, error) {
	var total, up int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END), 0)
		 FROM check_results WHERE monitor_id = ? AND checked_at >= ?`,
		monitorID, since,
	).Scan(&total, &up)
	if err != nil || total == 0 {
		return 100.0, err
	}
	return float64(up) / float64(total) * 100.0, nil
}

// GetCheckAggregateStats calculates aggregate stats for a monitor over a time range.
func (db *DB) GetCheckAggregateStats(ctx context.Context, monitorID string, start, end time.Time) (*models.CheckAggregate, error) {
	agg := &models.CheckAggregate{
		MonitorID:   monitorID,
		PeriodStart: start,
	}

	var uptime sql.NullFloat64
	var avg sql.NullFloat64
	var min, max sql.NullInt64

	err := db.QueryRowContext(ctx,
		`SELECT 
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END), 0),
			AVG(latency_ms),
			MIN(latency_ms),
			MAX(latency_ms),
			CAST(COALESCE(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END), 0) AS FLOAT) / CAST(NULLIF(COUNT(*), 0) AS FLOAT) * 100.0
		 FROM check_results 
		 WHERE monitor_id = ? AND checked_at >= ? AND checked_at < ?`,
		monitorID, start, end,
	).Scan(&agg.TotalChecks, &agg.UpCount, &agg.DownCount, &avg, &min, &max, &uptime)

	if err != nil {
		return nil, err
	}

	if avg.Valid {
		val := avg.Float64
		agg.AvgLatency = &val
	}
	if min.Valid {
		val := int(min.Int64)
		agg.MinLatency = &val
	}
	if max.Valid {
		val := int(max.Int64)
		agg.MaxLatency = &val
	}
	if uptime.Valid {
		val := uptime.Float64
		agg.UptimePct = &val
	} else {
		val := 100.0
		agg.UptimePct = &val
	}

	return agg, nil
}

// PurgeOldChecks removes raw check results older than the given duration.
func (db *DB) PurgeOldChecks(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := db.ExecContext(ctx,
		"DELETE FROM check_results WHERE checked_at < ?", olderThan,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// --- Status Page Queries ---

func (db *DB) CreateStatusPage(ctx context.Context, sp *models.StatusPage) error {
	groupsJSON, _ := json.Marshal(sp.Groups)
	_, err := db.ExecContext(ctx,
		`INSERT INTO status_pages (id, name, slug, description, groups, is_public, password)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sp.ID, sp.Name, sp.Slug, sp.Description, string(groupsJSON), sp.IsPublic, sp.Password,
	)
	return err
}

func (db *DB) ListStatusPages(ctx context.Context) ([]*models.StatusPage, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, name, slug, description, groups, is_public 
		 FROM status_pages ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []*models.StatusPage
	for rows.Next() {
		sp := &models.StatusPage{}
		var groupsJSON string
		if err := rows.Scan(&sp.ID, &sp.Name, &sp.Slug, &sp.Description, &groupsJSON, &sp.IsPublic); err != nil {
			return nil, err
		}
		if groupsJSON != "" {
			_ = json.Unmarshal([]byte(groupsJSON), &sp.Groups)
		}
		pages = append(pages, sp)
	}
	return pages, rows.Err()
}

func (db *DB) GetStatusPageBySlug(ctx context.Context, slug string) (*models.StatusPage, error) {
	sp := &models.StatusPage{}
	var groupsJSON string
	err := db.QueryRowContext(ctx,
		`SELECT id, name, slug, description, groups, is_public, password
		 FROM status_pages WHERE slug = ?`, slug,
	).Scan(&sp.ID, &sp.Name, &sp.Slug, &sp.Description, &groupsJSON, &sp.IsPublic, &sp.Password)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if groupsJSON != "" {
		_ = json.Unmarshal([]byte(groupsJSON), &sp.Groups)
	}
	return sp, nil
}

func (db *DB) GetStatusPageByID(ctx context.Context, id string) (*models.StatusPage, error) {
	sp := &models.StatusPage{}
	var groupsJSON string
	err := db.QueryRowContext(ctx,
		`SELECT id, name, slug, description, groups, is_public, password
		 FROM status_pages WHERE id = ?`, id,
	).Scan(&sp.ID, &sp.Name, &sp.Slug, &sp.Description, &groupsJSON, &sp.IsPublic, &sp.Password)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if groupsJSON != "" {
		_ = json.Unmarshal([]byte(groupsJSON), &sp.Groups)
	}
	return sp, nil
}

func (db *DB) UpdateStatusPage(ctx context.Context, sp *models.StatusPage) error {
	groupsJSON, _ := json.Marshal(sp.Groups)
	_, err := db.ExecContext(ctx,
		`UPDATE status_pages SET name=?, slug=?, description=?, groups=?, is_public=?, password=?
		 WHERE id=?`,
		sp.Name, sp.Slug, sp.Description, string(groupsJSON), sp.IsPublic, sp.Password, sp.ID,
	)
	return err
}

func (db *DB) DeleteStatusPage(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM status_pages WHERE id = ?", id)
	return err
}

// GetMonitorsSummary returns id, name, type, groups, and latest status for all monitors.
func (db *DB) GetMonitorsSummary(ctx context.Context) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT m.id, m.name, m.type, m.groups, m.enabled,
		       cr.status, cr.latency_ms, cr.checked_at
		FROM monitors m
		LEFT JOIN (
			SELECT monitor_id, status, latency_ms, checked_at,
			       ROW_NUMBER() OVER (PARTITION BY monitor_id ORDER BY checked_at DESC) as rn
			FROM check_results
		) cr ON m.id = cr.monitor_id AND cr.rn = 1
		ORDER BY json_extract(m.groups, '$[0]'), m.name
	`)
	if err != nil {
		return nil, fmt.Errorf("querying monitors summary: %w", err)
	}
	defer rows.Close()

	var summaries []map[string]any
	for rows.Next() {
		var (
			id, name, typ string
			groupsJSON    string
			enabled       bool
			status        sql.NullString
			latencyMs     sql.NullInt64
			checkedAt     sql.NullTime
		)
		if err := rows.Scan(&id, &name, &typ, &groupsJSON, &enabled, &status, &latencyMs, &checkedAt); err != nil {
			return nil, err
		}
		var groups []string
		if groupsJSON != "" {
			_ = json.Unmarshal([]byte(groupsJSON), &groups)
		}
		s := map[string]any{
			"id":      id,
			"name":    name,
			"type":    typ,
			"groups":  groups,
			"enabled": enabled,
			"status":  status.String,
		}
		if latencyMs.Valid {
			s["latency_ms"] = latencyMs.Int64
		}
		if checkedAt.Valid {
			s["last_check"] = checkedAt.Time
		}
		if !enabled {
			s["status"] = "paused"
		} else if !status.Valid {
			s["status"] = "pending"
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// --- Notification Channel Queries ---

func (db *DB) CreateNotificationChannel(ctx context.Context, nc *models.NotificationChannel) error {
	config, _ := json.Marshal(nc.Config)
	_, err := db.ExecContext(ctx,
		`INSERT INTO notification_channels (id, name, type, config, enabled)
		 VALUES (?, ?, ?, ?, ?)`,
		nc.ID, nc.Name, nc.Type, string(config), nc.Enabled,
	)
	return err
}

func (db *DB) ListNotificationChannels(ctx context.Context) ([]*models.NotificationChannel, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, name, type, config, enabled 
		 FROM notification_channels ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*models.NotificationChannel
	for rows.Next() {
		nc := &models.NotificationChannel{}
		var configJSON string
		if err := rows.Scan(&nc.ID, &nc.Name, &nc.Type, &configJSON, &nc.Enabled); err != nil {
			return nil, err
		}
		if configJSON != "" {
			_ = json.Unmarshal([]byte(configJSON), &nc.Config)
		}
		channels = append(channels, nc)
	}
	return channels, rows.Err()
}

func (db *DB) GetNotificationChannel(ctx context.Context, id string) (*models.NotificationChannel, error) {
	nc := &models.NotificationChannel{}
	var configJSON string
	err := db.QueryRowContext(ctx,
		`SELECT id, name, type, config, enabled 
		 FROM notification_channels WHERE id = ?`, id,
	).Scan(&nc.ID, &nc.Name, &nc.Type, &configJSON, &nc.Enabled)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if configJSON != "" {
		_ = json.Unmarshal([]byte(configJSON), &nc.Config)
	}
	return nc, nil
}

func (db *DB) UpdateNotificationChannel(ctx context.Context, nc *models.NotificationChannel) error {
	config, _ := json.Marshal(nc.Config)
	_, err := db.ExecContext(ctx,
		`UPDATE notification_channels SET name=?, type=?, config=?, enabled=?
		 WHERE id=?`,
		nc.Name, nc.Type, string(config), nc.Enabled, nc.ID,
	)
	return err
}

func (db *DB) DeleteNotificationChannel(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM notification_channels WHERE id = ?", id)
	return err
}

// --- Incident Queries ---

func (db *DB) CreateIncident(ctx context.Context, i *models.Incident) error {
	monitors, _ := json.Marshal(i.MonitorIDs)
	_, err := db.ExecContext(ctx,
		`INSERT INTO incidents (id, title, description, status, severity, monitor_ids, started_at, resolved_at, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		i.ID, i.Title, i.Description, i.Status, i.Severity, string(monitors), i.StartedAt, i.ResolvedAt, i.CreatedBy,
	)
	return err
}

func (db *DB) ListIncidents(ctx context.Context) ([]*models.Incident, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, title, description, status, severity, monitor_ids, started_at, resolved_at, created_by FROM incidents ORDER BY started_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []*models.Incident
	for rows.Next() {
		i := &models.Incident{}
		var monitorsJSON string
		if err := rows.Scan(&i.ID, &i.Title, &i.Description, &i.Status, &i.Severity, &monitorsJSON, &i.StartedAt, &i.ResolvedAt, &i.CreatedBy); err != nil {
			return nil, err
		}
		if monitorsJSON != "" {
			_ = json.Unmarshal([]byte(monitorsJSON), &i.MonitorIDs)
		}
		incidents = append(incidents, i)
	}
	return incidents, rows.Err()
}

func (db *DB) GetIncident(ctx context.Context, id string) (*models.Incident, error) {
	i := &models.Incident{}
	var monitorsJSON string
	err := db.QueryRowContext(ctx, `SELECT id, title, description, status, severity, monitor_ids, started_at, resolved_at, created_by FROM incidents WHERE id = ?`, id).
		Scan(&i.ID, &i.Title, &i.Description, &i.Status, &i.Severity, &monitorsJSON, &i.StartedAt, &i.ResolvedAt, &i.CreatedBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if monitorsJSON != "" {
		_ = json.Unmarshal([]byte(monitorsJSON), &i.MonitorIDs)
	}
	return i, nil
}

func (db *DB) UpdateIncident(ctx context.Context, i *models.Incident) error {
	monitors, _ := json.Marshal(i.MonitorIDs)
	_, err := db.ExecContext(ctx,
		`UPDATE incidents SET title=?, description=?, status=?, severity=?, monitor_ids=?, started_at=?, resolved_at=?
		 WHERE id=?`,
		i.Title, i.Description, i.Status, i.Severity, string(monitors), i.StartedAt, i.ResolvedAt, i.ID,
	)
	return err
}

func (db *DB) DeleteIncident(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM incidents WHERE id = ?", id)
	return err
}

// --- Maintenance Window Queries ---

func (db *DB) CreateMaintenanceWindow(ctx context.Context, mw *models.MaintenanceWindow) error {
	monitors, _ := json.Marshal(mw.MonitorIDs)
	_, err := db.ExecContext(ctx,
		`INSERT INTO maintenance_windows (id, title, monitor_ids, starts_at, ends_at, recurring, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		mw.ID, mw.Title, string(monitors), mw.StartsAt, mw.EndsAt, mw.Recurring, mw.CreatedBy,
	)
	return err
}

func (db *DB) ListMaintenanceWindows(ctx context.Context) ([]*models.MaintenanceWindow, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, title, monitor_ids, starts_at, ends_at, recurring, created_by FROM maintenance_windows ORDER BY starts_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var windows []*models.MaintenanceWindow
	for rows.Next() {
		mw := &models.MaintenanceWindow{}
		var monitorsJSON string
		if err := rows.Scan(&mw.ID, &mw.Title, &monitorsJSON, &mw.StartsAt, &mw.EndsAt, &mw.Recurring, &mw.CreatedBy); err != nil {
			return nil, err
		}
		if monitorsJSON != "" {
			_ = json.Unmarshal([]byte(monitorsJSON), &mw.MonitorIDs)
		}
		windows = append(windows, mw)
	}
	return windows, rows.Err()
}

func (db *DB) UpdateMaintenanceWindow(ctx context.Context, mw *models.MaintenanceWindow) error {
	monitors, _ := json.Marshal(mw.MonitorIDs)
	_, err := db.ExecContext(ctx,
		`UPDATE maintenance_windows SET title=?, monitor_ids=?, starts_at=?, ends_at=?, recurring=?
		 WHERE id=?`,
		mw.Title, string(monitors), mw.StartsAt, mw.EndsAt, mw.Recurring, mw.ID,
	)
	return err
}

func (db *DB) DeleteMaintenanceWindow(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM maintenance_windows WHERE id = ?", id)
	return err
}

func (db *DB) GetMaintenanceWindow(ctx context.Context, id string) (*models.MaintenanceWindow, error) {
	mw := &models.MaintenanceWindow{}
	var monitorsJSON string
	err := db.QueryRowContext(ctx, `SELECT id, title, monitor_ids, starts_at, ends_at, recurring, created_by FROM maintenance_windows WHERE id = ?`, id).
		Scan(&mw.ID, &mw.Title, &monitorsJSON, &mw.StartsAt, &mw.EndsAt, &mw.Recurring, &mw.CreatedBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if monitorsJSON != "" {
		_ = json.Unmarshal([]byte(monitorsJSON), &mw.MonitorIDs)
	}
	return mw, nil
}

// IsMonitorUnderMaintenance checks if a monitor is currently in an active maintenance window.
func (db *DB) IsMonitorUnderMaintenance(ctx context.Context, monitorID string) (bool, error) {
	// For v1, we only support explicit start/end dates for maintenance windows.
	query := `
		SELECT COUNT(*)
		FROM maintenance_windows, json_each(monitor_ids)
		WHERE json_each.value = ?
		  AND starts_at <= ? 
		  AND ends_at >= ?
	`
	var count int
	err := db.QueryRowContext(ctx, query, monitorID, time.Now(), time.Now()).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// --- Heartbeat Queries ---

func (db *DB) UpsertHeartbeat(ctx context.Context, h *models.Heartbeat) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO heartbeats (slug, monitor_id, token, last_ping, expected_s, grace_s)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(slug) DO UPDATE SET 
		 	monitor_id=excluded.monitor_id,
		 	token=excluded.token,
		 	last_ping=excluded.last_ping,
		 	expected_s=excluded.expected_s,
		 	grace_s=excluded.grace_s`,
		h.Slug, h.MonitorID, h.Token, h.LastPing, h.ExpectedS, h.GraceS,
	)
	return err
}

func (db *DB) GetHeartbeat(ctx context.Context, slug string) (*models.Heartbeat, error) {
	h := &models.Heartbeat{}
	err := db.QueryRowContext(ctx,
		"SELECT slug, monitor_id, token, last_ping, expected_s, grace_s FROM heartbeats WHERE slug = ?", slug,
	).Scan(&h.Slug, &h.MonitorID, &h.Token, &h.LastPing, &h.ExpectedS, &h.GraceS)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return h, err
}

func (db *DB) GetHeartbeatByToken(ctx context.Context, token string) (*models.Heartbeat, error) {
	h := &models.Heartbeat{}
	err := db.QueryRowContext(ctx,
		"SELECT slug, monitor_id, token, last_ping, expected_s, grace_s FROM heartbeats WHERE token = ?", token,
	).Scan(&h.Slug, &h.MonitorID, &h.Token, &h.LastPing, &h.ExpectedS, &h.GraceS)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return h, err
}

// --- Aggregate Queries ---

func (db *DB) UpsertCheckAggregate(ctx context.Context, a *models.CheckAggregate) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO check_aggregates (monitor_id, period_start, resolution, total_checks, up_count, down_count, avg_latency, min_latency, max_latency, uptime_pct)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(monitor_id, period_start, resolution) DO UPDATE SET
		 	total_checks=excluded.total_checks,
		 	up_count=excluded.up_count,
		 	down_count=excluded.down_count,
		 	avg_latency=excluded.avg_latency,
		 	min_latency=excluded.min_latency,
		 	max_latency=excluded.max_latency,
		 	uptime_pct=excluded.uptime_pct`,
		a.MonitorID, a.PeriodStart, a.Resolution, a.TotalChecks, a.UpCount, a.DownCount, a.AvgLatency, a.MinLatency, a.MaxLatency, a.UptimePct,
	)
	return err
}

// --- System Queries ---

func (db *DB) GetSystemMetrics(ctx context.Context) (*models.SystemMetrics, error) {
	metrics := &models.SystemMetrics{}

	// Query monitor counts by latest status
	query := `
		WITH LatestChecks AS (
			SELECT monitor_id, status,
				ROW_NUMBER() OVER (PARTITION BY monitor_id ORDER BY checked_at DESC) as rn
			FROM check_results
		)
		SELECT
			COUNT(m.id) as total,
			SUM(CASE WHEN m.enabled = 0 THEN 1 ELSE 0 END) as paused,
			SUM(CASE WHEN m.enabled = 1 AND lc.status = 'up' THEN 1 ELSE 0 END) as up,
			SUM(CASE WHEN m.enabled = 1 AND lc.status = 'down' THEN 1 ELSE 0 END) as down,
			SUM(CASE WHEN m.enabled = 1 AND lc.status = 'degraded' THEN 1 ELSE 0 END) as degraded
		FROM monitors m
		LEFT JOIN LatestChecks lc ON m.id = lc.monitor_id AND lc.rn = 1;
	`

	// Handle NULL SUMs if there are no monitors
	var total sql.NullInt64
	var paused sql.NullInt64
	var up sql.NullInt64
	var down sql.NullInt64
	var degraded sql.NullInt64

	err := db.QueryRowContext(ctx, query).Scan(&total, &paused, &up, &down, &degraded)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	metrics.TotalMonitors = int(total.Int64)
	metrics.MonitorsPaused = int(paused.Int64)
	metrics.MonitorsUp = int(up.Int64)
	metrics.MonitorsDown = int(down.Int64)
	metrics.MonitorsDegraded = int(degraded.Int64)

	// Query active incidents
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM incidents WHERE status != 'resolved'").Scan(&metrics.ActiveIncidents)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return metrics, nil
}

// --- Settings Queries ---

func (db *DB) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (db *DB) SetSetting(ctx context.Context, key, value string) error {
	_, err := db.ExecContext(ctx,
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	return err
}

func (db *DB) ListSettings(ctx context.Context) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}
	return settings, nil
}
