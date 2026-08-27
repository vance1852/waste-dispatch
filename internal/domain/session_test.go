package domain

import (
	"testing"
	"time"
)

func TestSession_IsExpired(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour)
	s := &Session{ExpiresAt: past}
	if !s.IsExpired() {
		t.Error("session with past expiry should be expired")
	}

	future := time.Now().UTC().Add(time.Hour)
	s.ExpiresAt = future
	if s.IsExpired() {
		t.Error("session with future expiry should not be expired")
	}
}

func TestSession_IsRevoked(t *testing.T) {
	s := &Session{}
	if s.IsRevoked() {
		t.Error("session without revoked_at should not be revoked")
	}
	s.Revoke()
	if !s.IsRevoked() {
		t.Error("session should be revoked after Revoke()")
	}
	if s.RevokedAt == nil {
		t.Error("RevokedAt should be set after Revoke()")
	}
}

func TestSession_IsValid(t *testing.T) {
	future := time.Now().UTC().Add(time.Hour)
	s := &Session{ExpiresAt: future}

	if !s.IsValid() {
		t.Error("non-expired, non-revoked session should be valid")
	}

	// Expire it.
	s.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	if s.IsValid() {
		t.Error("expired session should not be valid")
	}

	// Reset and revoke.
	s.ExpiresAt = future
	s.Revoke()
	if s.IsValid() {
		t.Error("revoked session should not be valid")
	}
}

func TestSession_Revoke_SetsTimestamp(t *testing.T) {
	before := time.Now().UTC()
	s := &Session{ExpiresAt: time.Now().UTC().Add(time.Hour)}
	s.Revoke()
	after := time.Now().UTC()

	if s.RevokedAt == nil {
		t.Fatal("RevokedAt must not be nil")
	}
	if s.RevokedAt.Before(before) || s.RevokedAt.After(after) {
		t.Errorf("RevokedAt %v is outside expected range [%v, %v]", *s.RevokedAt, before, after)
	}
}
