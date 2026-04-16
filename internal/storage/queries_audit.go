package storage

import (
	"context"
	"strings"

	"github.com/updu/updu/internal/models"
)

func (db *DB) CreateAuditLog(ctx context.Context, entry *models.AuditLog) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO audit_logs (actor_type, actor_id, actor_name, action, resource_type, resource_id, summary, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ActorType, entry.ActorID, entry.ActorName, entry.Action, entry.ResourceType, entry.ResourceID, entry.Summary, entry.CreatedAt,
	)
	return err
}

func (db *DB) ListAuditLogs(ctx context.Context, filters models.AuditLogFilters) ([]*models.AuditLog, error) {
	if filters.Limit <= 0 || filters.Limit > 500 {
		filters.Limit = 100
	}

	query := `SELECT id, actor_type, actor_id, actor_name, action, resource_type, resource_id, COALESCE(summary, ''), created_at FROM audit_logs`
	var where []string
	var args []any

	if filters.ActorID != "" {
		where = append(where, "actor_id = ?")
		args = append(args, filters.ActorID)
	}
	if filters.Action != "" {
		where = append(where, "action = ?")
		args = append(args, filters.Action)
	}
	if filters.ResourceType != "" {
		where = append(where, "resource_type = ?")
		args = append(args, filters.ResourceType)
	}
	if filters.ResourceID != "" {
		where = append(where, "resource_id = ?")
		args = append(args, filters.ResourceID)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, filters.Limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.AuditLog
	for rows.Next() {
		entry := &models.AuditLog{}
		if err := rows.Scan(&entry.ID, &entry.ActorType, &entry.ActorID, &entry.ActorName, &entry.Action, &entry.ResourceType, &entry.ResourceID, &entry.Summary, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
