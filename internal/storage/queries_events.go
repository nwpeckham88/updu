package storage

import (
	"context"
	"database/sql"

	"github.com/updu/updu/internal/models"
)

// --- Event Queries ---

func (db *DB) CreateEvent(ctx context.Context, e *models.Event) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO events (monitor_id, status, message, created_at)
		 VALUES (?, ?, ?, ?)`,
		e.MonitorID, e.Status, e.Message, e.CreatedAt,
	)
	return err
}

func (db *DB) ListRecentEvents(ctx context.Context, limit int) ([]*models.Event, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, monitor_id, status, message, created_at
		 FROM events ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		e := &models.Event{}
		var msg sql.NullString
		if err := rows.Scan(&e.ID, &e.MonitorID, &e.Status, &msg, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Message = msg.String
		events = append(events, e)
	}
	return events, rows.Err()
}

func (db *DB) ListEventsByMonitor(ctx context.Context, monitorID string, limit int) ([]*models.Event, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, monitor_id, status, message, created_at
		 FROM events WHERE monitor_id = ? ORDER BY created_at DESC LIMIT ?`, monitorID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		e := &models.Event{}
		var msg sql.NullString
		if err := rows.Scan(&e.ID, &e.MonitorID, &e.Status, &msg, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Message = msg.String
		events = append(events, e)
	}
	return events, rows.Err()
}
