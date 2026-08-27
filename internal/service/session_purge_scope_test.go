package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/config"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// TestIdleSessionPurgeKeepsSessionsThatAreStillValid checks that the housekeeping
// job only removes sign-ins that are really finished. A dispatcher who signed in
// last night and is still inside the allowed session window must stay signed in
// even if the console has been sitting untouched on the wall display.
func TestIdleSessionPurgeKeepsSessionsThatAreStillValid(t *testing.T) {
	db := openServiceTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	cfg := config.AuthConfig{TokenSecret: "seed", SessionExpiry: 7 * 24 * time.Hour}
	svc := service.NewAuthService(userRepo, sessionRepo, cfg, zerolog.Nop())
	ctx := context.Background()

	operator := &domain.User{
		ID:       uuid.New().String(),
		Username: "night-dispatcher-" + uuid.New().String()[:8],
		Role:     domain.RoleOperator,
		Status:   domain.UserStatusActive,
	}
	if err := operator.SetPassword("night-shift-pass"); err != nil {
		t.Fatalf("SetPassword error: %v", err)
	}
	if err := userRepo.Create(ctx, operator); err != nil {
		t.Fatalf("Create user error: %v", err)
	}

	// A wall-display console that has not been touched for ten days, yet its
	// sign-in was renewed and is still valid for another five days.
	idleToken := "idle-but-valid-" + uuid.New().String()
	idle := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    operator.ID,
		TokenHash: purgeHash(idleToken),
		ExpiresAt: time.Now().UTC().Add(5 * 24 * time.Hour),
	}
	if err := sessionRepo.Create(ctx, idle); err != nil {
		t.Fatalf("Create idle session error: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ? WHERE id = ?`,
		time.Now().UTC().Add(-10*24*time.Hour), idle.ID,
	); err != nil {
		t.Fatalf("backdate last_seen_at error: %v", err)
	}

	// A sign-in whose window really has run out.
	staleToken := "already-expired-" + uuid.New().String()
	stale := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    operator.ID,
		TokenHash: purgeHash(staleToken),
		ExpiresAt: time.Now().UTC().Add(-3 * 24 * time.Hour),
	}
	if err := sessionRepo.Create(ctx, stale); err != nil {
		t.Fatalf("Create stale session error: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ? WHERE id = ?`,
		time.Now().UTC().Add(-20*24*time.Hour), stale.ID,
	); err != nil {
		t.Fatalf("backdate stale last_seen_at error: %v", err)
	}

	if _, err := svc.PurgeIdleSessions(ctx); err != nil {
		t.Fatalf("PurgeIdleSessions error: %v", err)
	}

	if _, err := sessionRepo.GetByID(ctx, idle.ID); err != nil {
		t.Errorf(
			"a sign-in that is still inside its validity window was removed by housekeeping (%v); "+
				"only finished sign-ins may be purged",
			err,
		)
	}

	survivor, err := sessionRepo.GetByTokenHash(ctx, purgeHash(idleToken))
	if err != nil {
		t.Fatalf("the still valid sign-in must remain usable: %v", err)
	}
	if !survivor.IsValid() {
		t.Errorf("surviving sign-in is no longer usable: expires_at=%v revoked_at=%v", survivor.ExpiresAt, survivor.RevokedAt)
	}

	if _, err := sessionRepo.GetByID(ctx, stale.ID); err == nil {
		t.Error("a sign-in whose validity window already ran out should have been purged")
	}
}

func purgeHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
