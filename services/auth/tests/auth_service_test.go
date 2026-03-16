package tests

import (
	"errors"
	"testing"

	"easyoffer/auth/internal/domain"
	"easyoffer/auth/internal/repository"
	authservice "easyoffer/auth/internal/service"

	"github.com/golang-jwt/jwt/v5"
)

type userRepositoryStub struct {
	createErr   error
	createdUser *domain.User
	userByEmail *domain.User
	getByEmail  error
	gotEmail    string
}

func (s *userRepositoryStub) Create(user *domain.User) error {
	s.createdUser = user
	return s.createErr
}

func (s *userRepositoryStub) GetByEmail(email string) (*domain.User, error) {
	s.gotEmail = email
	if s.getByEmail != nil {
		return nil, s.getByEmail
	}
	return s.userByEmail, nil
}

func TestAuthServiceRegister_ReturnsUserAndToken(t *testing.T) {
	repo := &userRepositoryStub{}
	svc := authservice.NewAuthService(repo, "test-secret")

	user, token, err := svc.Register("john@example.com", "secret-123")
	if err != nil {
		t.Fatalf("expected register without error, got: %v", err)
	}
	if user == nil {
		t.Fatalf("expected user result")
	}
	if repo.createdUser == nil {
		t.Fatalf("expected repository create to be called")
	}
	if user.Email != "john@example.com" {
		t.Fatalf("unexpected email: %q", user.Email)
	}
	if user.Role != "user" {
		t.Fatalf("unexpected role: %q", user.Role)
	}
	if user.PasswordHash == "" || user.PasswordHash == "secret-123" {
		t.Fatalf("expected hashed password to be stored")
	}
	if !user.CheckPassword("secret-123") {
		t.Fatalf("expected stored password hash to validate original password")
	}

	claims := parseTokenClaims(t, token, "test-secret")
	if claims["sub"] != user.ID {
		t.Fatalf("expected token subject %q, got %v", user.ID, claims["sub"])
	}
	if claims["iss"] != "easyoffer-auth" {
		t.Fatalf("expected issuer easyoffer-auth, got %v", claims["iss"])
	}
}

func TestAuthServiceRegister_MapsDuplicateEmail(t *testing.T) {
	repo := &userRepositoryStub{createErr: repository.ErrEmailAlreadyExists}
	svc := authservice.NewAuthService(repo, "test-secret")

	user, token, err := svc.Register("john@example.com", "secret-123")
	if !errors.Is(err, authservice.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got: %v", err)
	}
	if user != nil {
		t.Fatalf("expected nil user on duplicate email")
	}
	if token != "" {
		t.Fatalf("expected empty token on duplicate email")
	}
}

func TestAuthServiceLogin_ReturnsTokenForValidCredentials(t *testing.T) {
	user := &domain.User{ID: "user-1", Email: "john@example.com", Role: "user"}
	if err := user.HashPassword("secret-123"); err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	repo := &userRepositoryStub{userByEmail: user}
	svc := authservice.NewAuthService(repo, "test-secret")

	token, err := svc.Login("john@example.com", "secret-123")
	if err != nil {
		t.Fatalf("expected login without error, got: %v", err)
	}
	if repo.gotEmail != "john@example.com" {
		t.Fatalf("expected lookup by email, got %q", repo.gotEmail)
	}

	claims := parseTokenClaims(t, token, "test-secret")
	if claims["sub"] != user.ID {
		t.Fatalf("expected token subject %q, got %v", user.ID, claims["sub"])
	}
}

func TestAuthServiceLogin_ReturnsErrorForWrongPassword(t *testing.T) {
	user := &domain.User{ID: "user-1", Email: "john@example.com", Role: "user"}
	if err := user.HashPassword("secret-123"); err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	repo := &userRepositoryStub{userByEmail: user}
	svc := authservice.NewAuthService(repo, "test-secret")

	token, err := svc.Login("john@example.com", "wrong-password")
	if err == nil {
		t.Fatalf("expected login error for wrong password")
	}
	if token != "" {
		t.Fatalf("expected empty token for wrong password")
	}
}

func parseTokenClaims(t *testing.T, tokenString, secret string) jwt.MapClaims {
	t.Helper()

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if !token.Valid {
		t.Fatalf("expected valid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("expected jwt.MapClaims")
	}
	return claims
}
