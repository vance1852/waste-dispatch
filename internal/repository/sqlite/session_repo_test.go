package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
)

func TestSessionRepository_CreateAndGetByToken(t *testing.T) {
	db := openTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	ctx := context.Background()

	// Create a user first.
	u := &domain.User{
		ID:       uuid.New().String(),
		Username: "session-user-1",
		Role:     domain.RoleOperator,
		Status:   domain.UserStatusActive,
	}
	_ = u.SetPassword("pass1234")
	_ = userRepo.Create(ctx, u)

	tokenHash := "hash-abc123"
	s := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    u.ID,
		TokenHash: tokenHash,
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	if err := sessionRepo.Create(ctx, s); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := sessionRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash() error: %v", err)
	}
	if got.UserID != u.ID {
		t.Errorf("user_id = %q, want %q", got.UserID, u.ID)
	}
	if got.RevokedAt != nil {
		t.Error("new session should not be revoked")
	}
}

func TestSessionRepository_Revoke(t *testing.T) {
	db := openTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	ctx := context.Background()

	u := &domain.User{ID: uuid.New().String(), Username: "session-user-2", Role: domain.RoleDriver, Status: domain.UserStatusActive}
	_ = u.SetPassword("pass1234")
	_ = userRepo.Create(ctx, u)

	s := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    u.ID,
		TokenHash: "hash-revoke-test",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	_ = sessionRepo.Create(ctx, s)

	if err := sessionRepo.Revoke(ctx, s.ID); err != nil {
		t.Fatalf("Revoke() error: %v", err)
	}

	got, err := sessionRepo.GetByTokenHash(ctx, "hash-revoke-test")
	if err != nil {
		t.Fatalf("GetByTokenHash() after revoke: %v", err)
	}
	if !got.IsRevoked() {
		t.Error("session should be revoked after Revoke()")
	}
}

func TestSessionRepository_RevokeAllForUser(t *testing.T) {
	db := openTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	ctx := context.Background()

	u := &domain.User{ID: uuid.New().String(), Username: "session-user-3", Role: domain.RoleResident, Status: domain.UserStatusActive}
	_ = u.SetPassword("pass1234")
	_ = userRepo.Create(ctx, u)

	// Create two sessions.
	for i, hash := range []string{"hash-a", "hash-b"} {
		_ = i
		s := &domain.Session{
			ID:        uuid.New().String(),
			UserID:    u.ID,
			TokenHash: hash,
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}
		_ = sessionRepo.Create(ctx, s)
	}

	if err := sessionRepo.RevokeAllForUser(ctx, u.ID); err != nil {
		t.Fatalf("RevokeAllForUser() error: %v", err)
	}

	// Both sessions should now be revoked.
	for _, hash := range []string{"hash-a", "hash-b"} {
		got, err := sessionRepo.GetByTokenHash(ctx, hash)
		if err != nil {
			t.Fatalf("GetByTokenHash(%q): %v", hash, err)
		}
		if !got.IsRevoked() {
			t.Errorf("session %s should be revoked after RevokeAllForUser", hash)
		}
	}
}

func TestSessionRepository_GetByTokenHash_NotFound(t *testing.T) {
	db := openTestDB(t)
	sessionRepo := reposqlite.NewSessionRepository(db)
	ctx := context.Background()

	_, err := sessionRepo.GetByTokenHash(ctx, "does-not-exist")
	if err != domain.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}
