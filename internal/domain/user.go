package domain

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Role represents a user role in the system.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleDriver   Role = "driver"
	RoleResident Role = "resident"
	RoleOperator Role = "operator"
)

// IsValid returns true if the role is one of the known roles.
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleDriver, RoleResident, RoleOperator:
		return true
	}
	return false
}

// UserStatus represents the status of a user account.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBanned   UserStatus = "banned"
)

// User is the core user entity.
type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	FullName     string     `json:"full_name"`
	Phone        string     `json:"phone"`
	Email        string     `json:"email"`
	Role         Role       `json:"role"`
	Status       UserStatus `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Version      int        `json:"version"`
}

// SetPassword hashes and stores the given plain-text password.
func (u *User) SetPassword(plain string) error {
	if len(plain) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword returns nil if plain matches the stored hash.
func (u *User) CheckPassword(plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(plain))
}

// IsActive returns true when the user account is active.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// ErrUserNotFound is returned when a user cannot be found.
var ErrUserNotFound = errors.New("user not found")

// ErrUserAlreadyExists is returned when a username is already taken.
var ErrUserAlreadyExists = errors.New("user already exists")

// ErrInvalidCredentials is returned for bad login attempts.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrUserInactive is returned when a disabled account tries to log in.
var ErrUserInactive = errors.New("user account is inactive or banned")
