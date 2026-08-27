package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/config"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
	"time"
)

func TestAuthService_RegisterAndLogin(t *testing.T) {
	db := openServiceTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	cfg := config.AuthConfig{
		TokenSecret:   "test-secret",
		TokenExpiry:   time.Hour,
		SessionExpiry: 24 * time.Hour,
	}
	svc := service.NewAuthService(userRepo, sessionRepo, cfg, zerolog.Nop())
	ctx := context.Background()

	// Register.
	user, err := svc.Register(ctx, service.RegisterRequest{
		Username: "testuser",
		Password: "password123",
		FullName: "Test User",
		Role:     domain.RoleOperator,
	})
	if err != nil {
		t.Fatalf("Register() error: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("username = %q, want testuser", user.Username)
	}

	// Login.
	res, err := svc.Login(ctx, service.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		UserAgent: "test",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}
	if res.Token == "" {
		t.Error("token should not be empty")
	}
	if res.User.ID != user.ID {
		t.Errorf("returned user ID mismatch")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	db := openServiceTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	cfg := config.AuthConfig{TokenSecret: "s", SessionExpiry: time.Hour}
	svc := service.NewAuthService(userRepo, sessionRepo, cfg, zerolog.Nop())
	ctx := context.Background()

	_, _ = svc.Register(ctx, service.RegisterRequest{Username: "wrongpw", Password: "correct123", Role: domain.RoleResident})
	_, err := svc.Login(ctx, service.LoginRequest{Username: "wrongpw", Password: "wrongpassword", UserAgent: "t", IPAddress: "1"})
	if err != domain.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	db := openServiceTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	cfg := config.AuthConfig{TokenSecret: "s", SessionExpiry: 24 * time.Hour}
	svc := service.NewAuthService(userRepo, sessionRepo, cfg, zerolog.Nop())
	ctx := context.Background()

	_, _ = svc.Register(ctx, service.RegisterRequest{Username: "validateuser", Password: "pass1234", Role: domain.RoleDriver})
	loginRes, _ := svc.Login(ctx, service.LoginRequest{Username: "validateuser", Password: "pass1234", UserAgent: "t", IPAddress: "1"})

	session, user, err := svc.ValidateToken(ctx, loginRes.Token)
	if err != nil {
		t.Fatalf("ValidateToken() error: %v", err)
	}
	if session == nil {
		t.Error("session should not be nil")
	}
	if user.Username != "validateuser" {
		t.Errorf("user.Username = %q, want validateuser", user.Username)
	}
}

func TestAuthService_Logout_RevokesSession(t *testing.T) {
	db := openServiceTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	cfg := config.AuthConfig{TokenSecret: "s", SessionExpiry: 24 * time.Hour}
	svc := service.NewAuthService(userRepo, sessionRepo, cfg, zerolog.Nop())
	ctx := context.Background()

	_, _ = svc.Register(ctx, service.RegisterRequest{Username: "logoutuser", Password: "pass1234", Role: domain.RoleResident})
	loginRes, _ := svc.Login(ctx, service.LoginRequest{Username: "logoutuser", Password: "pass1234", UserAgent: "t", IPAddress: "1"})

	if err := svc.Logout(ctx, loginRes.Token); err != nil {
		t.Fatalf("Logout() error: %v", err)
	}

	// Subsequent token validation should fail.
	_, _, err := svc.ValidateToken(ctx, loginRes.Token)
	if err == nil {
		t.Error("ValidateToken should fail after logout")
	}
}

func TestAuthService_ValidateToken_InvalidToken(t *testing.T) {
	db := openServiceTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	cfg := config.AuthConfig{TokenSecret: "s", SessionExpiry: time.Hour}
	svc := service.NewAuthService(userRepo, sessionRepo, cfg, zerolog.Nop())
	ctx := context.Background()

	_, _, err := svc.ValidateToken(ctx, "completelyfaketoken")
	if err != domain.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_Register_DuplicateUsername(t *testing.T) {
	db := openServiceTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	cfg := config.AuthConfig{TokenSecret: "s", SessionExpiry: time.Hour}
	svc := service.NewAuthService(userRepo, sessionRepo, cfg, zerolog.Nop())
	ctx := context.Background()

	_, _ = svc.Register(ctx, service.RegisterRequest{Username: "dupuser", Password: "pass1234", Role: domain.RoleResident})
	_, err := svc.Register(ctx, service.RegisterRequest{Username: "dupuser", Password: "pass5678", Role: domain.RoleResident})
	if err != domain.ErrUserAlreadyExists {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestAuthService_Register_DefaultRole(t *testing.T) {
	db := openServiceTestDB(t)
	userRepo := reposqlite.NewUserRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	cfg := config.AuthConfig{TokenSecret: "s", SessionExpiry: time.Hour}
	svc := service.NewAuthService(userRepo, sessionRepo, cfg, zerolog.Nop())
	ctx := context.Background()

	// Pass an empty/invalid role - should default to resident.
	u, err := svc.Register(ctx, service.RegisterRequest{
		Username: "defaultrole-" + uuid.New().String()[:8],
		Password: "pass1234",
		Role:     domain.Role("invalid"),
	})
	if err != nil {
		t.Fatalf("Register() error: %v", err)
	}
	if u.Role != domain.RoleResident {
		t.Errorf("role = %s, want resident", u.Role)
	}
}
