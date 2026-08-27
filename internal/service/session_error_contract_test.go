package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/config"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// TestRevokedAndExpiredSessionsStayDistinguishable checks that callers can still
// tell apart the reasons a session is refused. A logged-out session must be
// reported as revoked and an outdated session as expired, so the HTTP layer can
// answer "please sign in again" instead of a generic rejection.
func TestRevokedAndExpiredSessionsStayDistinguishable(t *testing.T) {
	db := openServiceTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	cfg := config.AuthConfig{TokenSecret: "seed", SessionExpiry: 24 * time.Hour}
	svc := service.NewAuthService(userRepo, sessionRepo, cfg, zerolog.Nop())
	ctx := context.Background()

	if _, err := svc.Register(ctx, service.RegisterRequest{
		Username: "dispatch-operator-" + uuid.New().String()[:8],
		Password: "operator-pass",
		FullName: "调度值班员",
		Role:     domain.RoleOperator,
	}); err != nil {
		t.Fatalf("Register error: %v", err)
	}

	users, _, err := userRepo.List(ctx, listAllUsers())
	if err != nil || len(users) == 0 {
		t.Fatalf("List users error: %v (count=%d)", err, len(users))
	}
	operator := users[0]

	// Case 1: an operator signs out, so the session is revoked.
	revokedToken := "revoked-token-" + uuid.New().String()
	revoked := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    operator.ID,
		TokenHash: hashForTest(revokedToken),
		ExpiresAt: time.Now().UTC().Add(12 * time.Hour),
	}
	if err := sessionRepo.Create(ctx, revoked); err != nil {
		t.Fatalf("Create revoked session error: %v", err)
	}
	if err := sessionRepo.Revoke(ctx, revoked.ID); err != nil {
		t.Fatalf("Revoke session error: %v", err)
	}

	_, _, revokedErr := svc.ValidateToken(ctx, revokedToken)
	if revokedErr == nil {
		t.Fatal("a revoked session must be refused")
	}
	if !errors.Is(revokedErr, domain.ErrSessionRevoked) {
		t.Errorf(
			"refusal for a signed-out session is no longer recognisable as a revoked session (got %v); "+
				"callers can no longer tell revocation apart from other refusals",
			revokedErr,
		)
	}

	// Case 2: a session that simply aged out.
	expiredToken := "expired-token-" + uuid.New().String()
	expired := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    operator.ID,
		TokenHash: hashForTest(expiredToken),
		ExpiresAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	if err := sessionRepo.Create(ctx, expired); err != nil {
		t.Fatalf("Create expired session error: %v", err)
	}

	_, _, expiredErr := svc.ValidateToken(ctx, expiredToken)
	if expiredErr == nil {
		t.Fatal("an expired session must be refused")
	}
	if !errors.Is(expiredErr, domain.ErrSessionExpired) {
		t.Errorf(
			"refusal for an aged-out session is no longer recognisable as an expired session (got %v)",
			expiredErr,
		)
	}
	if errors.Is(expiredErr, domain.ErrSessionRevoked) {
		t.Errorf("an expired session must not be reported as revoked (got %v)", expiredErr)
	}
}

func hashForTest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func listAllUsers() repository.UserFilter {
	return repository.UserFilter{Limit: 10}
}
