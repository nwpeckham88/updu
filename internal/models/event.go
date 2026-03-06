package models

import "time"

// Event represents a state transition for a monitor.
type Event struct {
	ID        int64         `json:"id"`
	MonitorID string        `json:"monitor_id"`
	Status    MonitorStatus `json:"status"`
	Message   string        `json:"message,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}
