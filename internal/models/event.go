package models

import "time"

// Event represents a service state transition in the service history timeline (e.g. up, down, degraded).
type Event struct {
	ID        int64         `json:"id"`
	MonitorID string        `json:"monitor_id"`
	Status    MonitorStatus `json:"status"`
	Message   string        `json:"message,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}

// ServiceHistoryEntry is an alias for Event to explicitly denote service transition history.
type ServiceHistoryEntry = Event
