package models

import "time"

// TicketStatus represents the lifecycle state of an issue ticket.
type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "open"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusResolved   TicketStatus = "resolved"
	TicketStatusClosed     TicketStatus = "closed"
)

// Ticket represents a user-submitted issue or outage report.
type Ticket struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      TicketStatus `json:"status"`
	Severity    string       `json:"severity"` // "low", "medium", "high"
	ServiceID   *string      `json:"service_id,omitempty"`
	ServiceName string       `json:"service_name,omitempty"`
	MonitorID   *string      `json:"monitor_id,omitempty"`
	CreatedBy   string       `json:"created_by"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	ResolvedAt  *time.Time   `json:"resolved_at,omitempty"`
}
