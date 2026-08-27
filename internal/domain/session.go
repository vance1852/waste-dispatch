package domain

import (
	"errors"
	"time"
)

// Session represents an authenticated user session.
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Token        string    `json:"-"`
	TokenHash    string    `json:"-"`
	UserAgent    string    `json:"user_agent"`
	IPAddress    string    `json:"ip_address"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

// IsExpired returns true when the session has passed its expiry time.
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// IsRevoked returns true when the session has been explicitly revoked.
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

// IsValid returns true when the session is neither expired nor revoked.
func (s *Session) IsValid() bool {
	return !s.IsExpired() && !s.IsRevoked()
}

// Revoke marks the session as revoked at the current time.
func (s *Session) Revoke() {
	now := time.Now().UTC()
	s.RevokedAt = &now
}

// Errors related to sessions.
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
	ErrSessionRevoked  = errors.New("session has been revoked")
	ErrInvalidToken    = errors.New("invalid authentication token")
)
