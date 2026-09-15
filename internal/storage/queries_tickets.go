package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/updu/updu/internal/models"
)

// CreateTicket inserts a new ticket.
func (db *DB) CreateTicket(ctx context.Context, t *models.Ticket) error {
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = models.TicketStatusOpen
	}
	if t.Severity == "" {
		t.Severity = "medium"
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO tickets (id, title, description, status, severity, service_id, service_name, monitor_id, created_by, created_at, updated_at, resolved_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.Title, t.Description, string(t.Status), t.Severity, t.ServiceID, t.ServiceName, t.MonitorID, t.CreatedBy, t.CreatedAt, t.UpdatedAt, t.ResolvedAt)
	if err != nil {
		return fmt.Errorf("inserting ticket: %w", err)
	}
	return nil
}

// ListTickets returns all tickets, optionally filtered by created_by user.
func (db *DB) ListTickets(ctx context.Context, filterUser string) ([]*models.Ticket, error) {
	var rows *sql.Rows
	var err error

	if filterUser != "" {
		rows, err = db.QueryContext(ctx, `
			SELECT id, title, description, status, severity, service_id, service_name, monitor_id, created_by, created_at, updated_at, resolved_at
			FROM tickets
			WHERE created_by = ?
			ORDER BY created_at DESC
		`, filterUser)
	} else {
		rows, err = db.QueryContext(ctx, `
			SELECT id, title, description, status, severity, service_id, service_name, monitor_id, created_by, created_at, updated_at, resolved_at
			FROM tickets
			ORDER BY created_at DESC
		`)
	}
	if err != nil {
		return nil, fmt.Errorf("listing tickets: %w", err)
	}
	defer rows.Close()

	var tickets []*models.Ticket
	for rows.Next() {
		var t models.Ticket
		var statusStr string
		var svcID, svcName, monID sql.NullString
		var resAt sql.NullTime

		if err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.Description,
			&statusStr,
			&t.Severity,
			&svcID,
			&svcName,
			&monID,
			&t.CreatedBy,
			&t.CreatedAt,
			&t.UpdatedAt,
			&resAt,
		); err != nil {
			return nil, fmt.Errorf("scanning ticket: %w", err)
		}

		t.Status = models.TicketStatus(statusStr)
		if svcID.Valid {
			t.ServiceID = &svcID.String
		}
		if svcName.Valid {
			t.ServiceName = svcName.String
		}
		if monID.Valid {
			t.MonitorID = &monID.String
		}
		if resAt.Valid {
			t.ResolvedAt = &resAt.Time
		}

		tickets = append(tickets, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tickets: %w", err)
	}

	return tickets, nil
}

// GetTicket retrieves a single ticket by its ID.
func (db *DB) GetTicket(ctx context.Context, id string) (*models.Ticket, error) {
	var t models.Ticket
	var statusStr string
	var svcID, svcName, monID sql.NullString
	var resAt sql.NullTime

	err := db.QueryRowContext(ctx, `
		SELECT id, title, description, status, severity, service_id, service_name, monitor_id, created_by, created_at, updated_at, resolved_at
		FROM tickets
		WHERE id = ?
	`, id).Scan(
		&t.ID,
		&t.Title,
		&t.Description,
		&statusStr,
		&t.Severity,
		&svcID,
		&svcName,
		&monID,
		&t.CreatedBy,
		&t.CreatedAt,
		&t.UpdatedAt,
		&resAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting ticket: %w", err)
	}

	t.Status = models.TicketStatus(statusStr)
	if svcID.Valid {
		t.ServiceID = &svcID.String
	}
	if svcName.Valid {
		t.ServiceName = svcName.String
	}
	if monID.Valid {
		t.MonitorID = &monID.String
	}
	if resAt.Valid {
		t.ResolvedAt = &resAt.Time
	}

	return &t, nil
}

// UpdateTicket updates an existing ticket.
func (db *DB) UpdateTicket(ctx context.Context, t *models.Ticket) error {
	t.UpdatedAt = time.Now()
	res, err := db.ExecContext(ctx, `
		UPDATE tickets
		SET title = ?, description = ?, status = ?, severity = ?, service_id = ?, service_name = ?, monitor_id = ?, updated_at = ?, resolved_at = ?
		WHERE id = ?
	`, t.Title, t.Description, string(t.Status), t.Severity, t.ServiceID, t.ServiceName, t.MonitorID, t.UpdatedAt, t.ResolvedAt, t.ID)
	if err != nil {
		return fmt.Errorf("updating ticket: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking updated ticket: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("ticket not found: %s", t.ID)
	}
	return nil
}

// DeleteTicket deletes a ticket by ID.
func (db *DB) DeleteTicket(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM tickets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting ticket: %w", err)
	}
	return nil
}
