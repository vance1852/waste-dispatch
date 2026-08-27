package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// UserRepository implements repository.UserRepository for SQLite.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new SQLite-backed UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user record.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now
	u.Version = 1

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, full_name, phone, email, role, status, created_at, updated_at, version)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.PasswordHash, u.FullName, u.Phone, u.Email,
		string(u.Role), string(u.Status), u.CreatedAt, u.UpdatedAt, u.Version,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by primary key.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, full_name, phone, email, role, status, created_at, updated_at, version
		 FROM users WHERE id = ?`, id)
	return scanUser(row)
}

// GetByUsername retrieves a user by username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, full_name, phone, email, role, status, created_at, updated_at, version
		 FROM users WHERE username = ?`, username)
	return scanUser(row)
}

// Update updates all mutable user fields.
func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	u.UpdatedAt = time.Now().UTC()
	u.Version++
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET full_name=?, phone=?, email=?, role=?, status=?, updated_at=?, version=?
		 WHERE id=? AND version=?`,
		u.FullName, u.Phone, u.Email, string(u.Role), string(u.Status),
		u.UpdatedAt, u.Version, u.ID, u.Version-1,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// Delete removes a user by ID.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// List returns paginated users matching optional filters.
func (r *UserRepository) List(ctx context.Context, f repository.UserFilter) ([]*domain.User, int, error) {
	args := []interface{}{}
	where := "WHERE 1=1"

	if f.Role != "" {
		where += " AND role = ?"
		args = append(args, string(f.Role))
	}
	if f.Status != "" {
		where += " AND status = ?"
		args = append(args, string(f.Status))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := f.Offset
	query := fmt.Sprintf(
		`SELECT id, username, password_hash, full_name, phone, email, role, status, created_at, updated_at, version
		 FROM users %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(
			&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Phone,
			&u.Email, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.Version,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func scanUser(row *sql.Row) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Phone,
		&u.Email, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return u, nil
}
