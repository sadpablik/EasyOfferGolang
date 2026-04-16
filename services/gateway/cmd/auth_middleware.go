package main

import (
	"context"
	gatewayauth "easyoffer/gateway/internal/auth"

	"github.com/gin-gonic/gin"
)

func JWTAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return gatewayauth.JWTAuthMiddleware(jwtSecret)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	return gatewayauth.UserIDFromContext(ctx)
}

func RoleFromContext(ctx context.Context) (string, bool) {
	return gatewayauth.RoleFromContext(ctx)
}
