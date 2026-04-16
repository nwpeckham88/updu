package models

import "time"

type AuditLog struct {
	ID           int64     `json:"id"`
	ActorType    string    `json:"actor_type"`
	ActorID      string    `json:"actor_id"`
	ActorName    string    `json:"actor_name"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Summary      string    `json:"summary,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuditLogFilters struct {
	Limit        int
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
}
