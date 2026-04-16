package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"easyoffer/auth/api/handlers"
	"easyoffer/auth/internal/domain"
	authservice "easyoffer/auth/internal/service"

	"github.com/gin-gonic/gin"
)

type authServiceStub struct {
	registerUser  *domain.User
	registerToken string
	registerErr   error

	loginToken string
	loginErr   error
}

func (s *authServiceStub) Register(_, _ string) (*domain.User, string, error) {
	if s.registerErr != nil {
		return nil, "", s.registerErr
	}
	return s.registerUser, s.registerToken, nil
}

func (s *authServiceStub) Login(_, _ string) (string, error) {
	if s.loginErr != nil {
		return "", s.loginErr
	}
	return s.loginToken, nil
}

func TestAuthHandlerRegister_ReturnsCreatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixedTime := time.Date(2026, 3, 16, 9, 0, 0, 0, time.UTC)

	h := handlers.NewAuthHandler(&authServiceStub{
		registerUser: &domain.User{
			ID:        "user-1",
			Email:     "john@example.com",
			CreatedAt: fixedTime,
			Role:      "user",
		},
		registerToken: "jwt-token",
	})

	r := gin.New()
	r.POST("/register", h.Register)

	body := []byte(`{"email":"john@example.com","password":"secret-123"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.Code)
	}

	var payload handlers.RegisterResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Token != "jwt-token" {
		t.Fatalf("unexpected token: %q", payload.Token)
	}
	if payload.User.ID != "user-1" {
		t.Fatalf("unexpected user id: %q", payload.User.ID)
	}
	if payload.User.Email != "john@example.com" {
		t.Fatalf("unexpected email: %q", payload.User.Email)
	}
	if payload.User.CreatedAt != fixedTime.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", payload.User.CreatedAt)
	}
}

func TestAuthHandlerRegister_MapsDuplicateEmailToConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handlers.NewAuthHandler(&authServiceStub{registerErr: authservice.ErrEmailAlreadyExists})
	r := gin.New()
	r.POST("/register", h.Register)

	body := []byte(`{"email":"john@example.com","password":"secret-123"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.Code)
	}

	var payload handlers.ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error != "email already exists" {
		t.Fatalf("unexpected error message: %q", payload.Error)
	}
}

func TestAuthHandlerRegister_ReturnsBadRequestForInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handlers.NewAuthHandler(&authServiceStub{})
	r := gin.New()
	r.POST("/register", h.Register)

	body := []byte(`{"email":"not-email","password":"123"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestAuthHandlerLogin_ReturnsToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handlers.NewAuthHandler(&authServiceStub{loginToken: "jwt-token"})
	r := gin.New()
	r.POST("/login", h.Login)

	body := []byte(`{"email":"john@example.com","password":"secret-123"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var payload handlers.LoginResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Token != "jwt-token" {
		t.Fatalf("unexpected token: %q", payload.Token)
	}
}

func TestAuthHandlerLogin_InvalidCredentialsReturnsUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handlers.NewAuthHandler(&authServiceStub{loginErr: errors.New("invalid email or password")})
	r := gin.New()
	r.POST("/login", h.Login)

	body := []byte(`{"email":"john@example.com","password":"wrong-pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}

	var payload handlers.ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error != "invalid email or password" {
		t.Fatalf("unexpected error message: %q", payload.Error)
	}
}

func TestAuthHandlerLogin_InvalidPayloadReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handlers.NewAuthHandler(&authServiceStub{})
	r := gin.New()
	r.POST("/login", h.Login)

	body := []byte(`{"email":"bad-email","password":""}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}
