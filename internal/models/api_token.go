package models

import "time"

type APITokenScope string

const (
	APITokenScopeRead  APITokenScope = "read"
	APITokenScopeWrite APITokenScope = "write"
)

func (scope APITokenScope) IsValid() bool {
	return scope == APITokenScopeRead || scope == APITokenScopeWrite
}

type APIToken struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Prefix     string        `json:"prefix"`
	Scope      APITokenScope `json:"scope"`
	CreatedBy  string        `json:"created_by"`
	CreatedAt  time.Time     `json:"created_at"`
	LastUsedAt *time.Time    `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time    `json:"revoked_at,omitempty"`
}

type APITokenSecret struct {
	APIToken
	Token string `json:"token"`
}
