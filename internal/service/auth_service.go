package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/config"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// AuthService handles authentication and session management.
type AuthService struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
	cfg      config.AuthConfig
	log      zerolog.Logger
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	users repository.UserRepository,
	sessions repository.SessionRepository,
	cfg config.AuthConfig,
	log zerolog.Logger,
) *AuthService {
	return &AuthService{users: users, sessions: sessions, cfg: cfg, log: log}
}

// LoginRequest carries credentials for a login attempt.
type LoginRequest struct {
	Username  string
	Password  string
	UserAgent string
	IPAddress string
}

// LoginResponse is returned on a successful login.
type LoginResponse struct {
	Token     string
	SessionID string
	User      *domain.User
	ExpiresAt time.Time
}

// Login authenticates a user and creates a new session.
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.users.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !user.IsActive() {
		return nil, domain.ErrUserInactive
	}

	if err := user.CheckPassword(req.Password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	token, tokenHash, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(s.cfg.SessionExpiry)
	session := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		UserAgent: req.UserAgent,
		IPAddress: req.IPAddress,
		ExpiresAt: expiresAt,
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	s.log.Info().
		Str("user_id", user.ID).
		Str("session_id", session.ID).
		Str("ip", req.IPAddress).
		Msg("user logged in")

	return &LoginResponse{
		Token:     token,
		SessionID: session.ID,
		User:      user,
		ExpiresAt: expiresAt,
	}, nil
}

// Logout revokes the session associated with the given token.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	tokenHash := hashToken(token)
	session, err := s.sessions.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil // Silently succeed if session not found.
	}
	return s.sessions.Revoke(ctx, session.ID)
}

// ValidateToken verifies the token and returns the associated session and user.
func (s *AuthService) ValidateToken(ctx context.Context, token string) (*domain.Session, *domain.User, error) {
	tokenHash := hashToken(token)
	session, err := s.sessions.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, nil, domain.ErrInvalidToken
	}

	if session.IsRevoked() {
		return nil, nil, domain.ErrSessionRevoked
	}
	if session.IsExpired() {
		return nil, nil, domain.ErrSessionExpired
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, domain.ErrInvalidToken
	}

	if !user.IsActive() {
		return nil, nil, domain.ErrUserInactive
	}

	// Update last seen asynchronously to avoid blocking the request.
	go func() {
		_ = s.sessions.UpdateLastSeen(context.Background(), session.ID)
	}()

	return session, user, nil
}

// PurgeIdleSessions drops sessions that have been idle longer than the configured
// session lifetime so the session table does not grow without bound.
func (s *AuthService) PurgeIdleSessions(ctx context.Context) (int64, error) {
	removed, err := s.sessions.PurgeIdle(ctx, s.cfg.SessionExpiry)
	if err != nil {
		return 0, fmt.Errorf("purge idle sessions: %w", err)
	}
	if removed > 0 {
		s.log.Info().Int64("removed", removed).Msg("idle sessions purged")
	}
	return removed, nil
}

// RegisterRequest carries data for creating a new user account.
type RegisterRequest struct {
	Username string
	Password string
	FullName string
	Phone    string
	Email    string
	Role     domain.Role
}

// Register creates a new user account.
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*domain.User, error) {
	if !req.Role.IsValid() {
		req.Role = domain.RoleResident
	}

	user := &domain.User{
		ID:       uuid.New().String(),
		Username: req.Username,
		FullName: req.FullName,
		Phone:    req.Phone,
		Email:    req.Email,
		Role:     req.Role,
		Status:   domain.UserStatusActive,
	}

	if err := user.SetPassword(req.Password); err != nil {
		return nil, fmt.Errorf("set password: %w", err)
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// generateToken creates a cryptographically random token and its SHA-256 hash.
func generateToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(b)
	hash = hashToken(token)
	return token, hash, nil
}

// hashToken returns the SHA-256 hex digest of a token string.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
