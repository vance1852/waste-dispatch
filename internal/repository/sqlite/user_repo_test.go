package sqlite_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
)

func TestUserRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewUserRepository(db)
	ctx := context.Background()

	u := &domain.User{
		ID:       uuid.New().String(),
		Username: "alice",
		FullName: "Alice Wang",
		Role:     domain.RoleOperator,
		Status:   domain.UserStatusActive,
	}
	if err := u.SetPassword("password123"); err != nil {
		t.Fatal(err)
	}

	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Username != "alice" {
		t.Errorf("username = %q, want alice", got.Username)
	}
	if got.Role != domain.RoleOperator {
		t.Errorf("role = %s, want operator", got.Role)
	}
	if got.Version != 1 {
		t.Errorf("version = %d, want 1", got.Version)
	}
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewUserRepository(db)
	ctx := context.Background()

	u := &domain.User{
		ID:       uuid.New().String(),
		Username: "bob",
		Role:     domain.RoleDriver,
		Status:   domain.UserStatusActive,
	}
	_ = u.SetPassword("pass1234")
	_ = repo.Create(ctx, u)

	got, err := repo.GetByUsername(ctx, "bob")
	if err != nil {
		t.Fatalf("GetByUsername() error: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("ID = %q, want %q", got.ID, u.ID)
	}
}

func TestUserRepository_DuplicateUsername_ReturnsError(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewUserRepository(db)
	ctx := context.Background()

	u1 := &domain.User{ID: uuid.New().String(), Username: "carol", Role: domain.RoleResident, Status: domain.UserStatusActive}
	_ = u1.SetPassword("pass1234")
	_ = repo.Create(ctx, u1)

	u2 := &domain.User{ID: uuid.New().String(), Username: "carol", Role: domain.RoleResident, Status: domain.UserStatusActive}
	_ = u2.SetPassword("pass5678")
	err := repo.Create(ctx, u2)
	if err != domain.ErrUserAlreadyExists {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err != domain.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepository_List_Pagination(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewUserRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		u := &domain.User{
			ID:       uuid.New().String(),
			Username: uuid.New().String()[:8],
			Role:     domain.RoleResident,
			Status:   domain.UserStatusActive,
		}
		_ = u.SetPassword("pass1234")
		_ = repo.Create(ctx, u)
	}

	users, total, err := repo.List(ctx, struct {
		Role   domain.Role
		Status domain.UserStatus
		Limit  int
		Offset int
	}{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total < 5 {
		t.Errorf("total = %d, want >= 5", total)
	}
	if len(users) > 2 {
		t.Errorf("got %d users, want <= 2", len(users))
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewUserRepository(db)
	ctx := context.Background()

	u := &domain.User{ID: uuid.New().String(), Username: "delme", Role: domain.RoleResident, Status: domain.UserStatusActive}
	_ = u.SetPassword("pass1234")
	_ = repo.Create(ctx, u)

	if err := repo.Delete(ctx, u.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	if _, err := repo.GetByID(ctx, u.ID); err != domain.ErrUserNotFound {
		t.Error("expected ErrUserNotFound after deletion")
	}
}
