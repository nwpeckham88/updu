package models

import "time"

// Zone represents a physical or administrative location / failure domain.
type Zone struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
