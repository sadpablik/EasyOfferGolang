package handlers

import (
	"time"

	"easyoffer/auth/internal/domain"
)

func toUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		Role:      user.Role,
	}
}
