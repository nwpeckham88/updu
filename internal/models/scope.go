package models

import "time"

// Built-in scope identifiers
const (
	ScopeLAN     = "lan"
	ScopeTailnet = "tailnet"
	ScopePublic  = "public"
)

// Scope represents a reachable network fabric or routing domain.
type Scope struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
