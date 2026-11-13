package models

import (
	"time"
)

// Session represents an active administrator HTTP-only cookie session.
type Session struct {
	ID        string    `json:"id"`
	UserID    uint64    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `json:"user,omitempty"`
}

// IsExpired checks if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

