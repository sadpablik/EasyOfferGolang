package main

import (
	"context"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	ctxUserIDKey ctxKey = "userID"
	ctxRoleKey   ctxKey = "role"
)

func JWTAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	secret := []byte(jwtSecret)
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "missing bearer token"})
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenString == "" {
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "empty bearer token"})
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "invalid claims"})
			return
		}

		userID, _ := claims["sub"].(string)
		role, _ := claims["role"].(string)
		if userID == "" {
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "token missing subject"})
			return
		}

		ctx := context.WithValue(c.Request.Context(), ctxUserIDKey, userID)
		ctx = context.WithValue(ctx, ctxRoleKey, role)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}

}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxUserIDKey).(string)
	return v, ok
}

func RoleFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxRoleKey).(string)
	return v, ok
}
