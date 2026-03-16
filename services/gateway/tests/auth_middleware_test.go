package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gatewayauth "easyoffer/gateway/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type errorResponse struct {
	Error string `json:"error"`
}

func TestJWTAuthMiddleware_RejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gatewayauth.JWTAuthMiddleware("test-secret"))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}

	var payload errorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if payload.Error != "missing bearer token" {
		t.Fatalf("unexpected error message: %q", payload.Error)
	}
}

func TestJWTAuthMiddleware_RejectsTokenWithoutSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gatewayauth.JWTAuthMiddleware("test-secret"))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	token := signToken(t, "test-secret", jwt.MapClaims{"role": "admin"})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}

	var payload errorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if payload.Error != "token missing subject" {
		t.Fatalf("unexpected error message: %q", payload.Error)
	}
}

func TestJWTAuthMiddleware_SetsUserContextForValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gatewayauth.JWTAuthMiddleware("test-secret"))
	router.GET("/protected", func(c *gin.Context) {
		userID, ok := gatewayauth.UserIDFromContext(c.Request.Context())
		role, _ := gatewayauth.RoleFromContext(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{
			"ok":      ok,
			"user_id": userID,
			"role":    role,
		})
	})

	token := signToken(t, "test-secret", jwt.MapClaims{"sub": "user-1", "role": "admin"})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var payload struct {
		OK     bool   `json:"ok"`
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode success response: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected user context to be present")
	}
	if payload.UserID != "user-1" {
		t.Fatalf("unexpected user id: %q", payload.UserID)
	}
	if payload.Role != "admin" {
		t.Fatalf("unexpected role: %q", payload.Role)
	}
}

func signToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return tokenString
}
