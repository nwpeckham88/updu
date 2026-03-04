package models

import "time"

// UserRole represents the authorization level.
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleViewer UserRole = "viewer"
)

// User represents an authenticated user.
type User struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	Password   string    `json:"-"` // bcrypt hash, never serialized
	Role       UserRole  `json:"role"`
	OIDCSub    *string   `json:"oidc_sub,omitempty"`
	OIDCIssuer *string   `json:"oidc_issuer,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// Session represents an active user session.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	UserAgent string    `json:"user_agent,omitempty"`
	IPAddr    string    `json:"ip_addr,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
