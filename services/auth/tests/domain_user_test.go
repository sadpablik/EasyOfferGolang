package tests

import (
	"testing"

	"easyoffer/auth/internal/domain"
)

func TestUserHashAndCheckPassword(t *testing.T) {
	u := &domain.User{}

	if err := u.HashPassword("secret-123"); err != nil {
		t.Fatalf("expected hash password without error, got: %v", err)
	}
	if u.PasswordHash == "" {
		t.Fatalf("expected non-empty password hash")
	}
	if !u.CheckPassword("secret-123") {
		t.Fatalf("expected correct password to pass check")
	}
	if u.CheckPassword("wrong-password") {
		t.Fatalf("expected wrong password to fail check")
	}
}
