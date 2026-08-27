package domain

import "testing"

func TestUser_SetPassword_And_CheckPassword(t *testing.T) {
	u := &User{}
	if err := u.SetPassword("secret123"); err != nil {
		t.Fatalf("SetPassword() unexpected error: %v", err)
	}
	if u.PasswordHash == "" {
		t.Error("PasswordHash should not be empty after SetPassword")
	}
	if u.PasswordHash == "secret123" {
		t.Error("PasswordHash must not store plain text")
	}
	if err := u.CheckPassword("secret123"); err != nil {
		t.Errorf("CheckPassword() with correct password returned error: %v", err)
	}
	if err := u.CheckPassword("wrongpass"); err == nil {
		t.Error("CheckPassword() should fail with wrong password")
	}
}

func TestUser_SetPassword_TooShort(t *testing.T) {
	u := &User{}
	if err := u.SetPassword("abc"); err == nil {
		t.Error("expected error for password shorter than 6 characters")
	}
}

func TestUser_IsActive(t *testing.T) {
	u := &User{Status: UserStatusActive}
	if !u.IsActive() {
		t.Error("active user should return true from IsActive()")
	}
	u.Status = UserStatusBanned
	if u.IsActive() {
		t.Error("banned user should return false from IsActive()")
	}
	u.Status = UserStatusInactive
	if u.IsActive() {
		t.Error("inactive user should return false from IsActive()")
	}
}

func TestRole_IsValid(t *testing.T) {
	for _, r := range []Role{RoleAdmin, RoleDriver, RoleResident, RoleOperator} {
		if !r.IsValid() {
			t.Errorf("%s should be valid role", r)
		}
	}
	if Role("superuser").IsValid() {
		t.Error("superuser is not a valid role")
	}
}
