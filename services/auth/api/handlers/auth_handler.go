package handlers

import (
	"easyoffer/auth/internal/domain"
	"easyoffer/auth/internal/service"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)



type AuthHandler struct {
    authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
    return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type UserResponse struct {
    ID        string `json:"id"`
    Email     string `json:"email"`
    CreatedAt string `json:"created_at"`
    Role      string `json:"role"`
}

type RegisterResponse struct {
    User  UserResponse `json:"user"`
    Token string       `json:"token"`
}

type LoginResponse struct {
    Token string `json:"token"`
}

type ErrorResponse struct {
    Error string `json:"error"`
}

func toUserResponse(user *domain.User) UserResponse {
    return UserResponse{
        ID:        user.ID,
        Email:     user.Email,
        CreatedAt: user.CreatedAt.Format(time.RFC3339),
        Role:      user.Role,
    }
}
// @Summary Register user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "User registration"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /register [post]
func (h *AuthHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
        return
    }

    user, token, err := h.authService.Register(req.Email, req.Password)
    if err != nil {
        if errors.Is(err, service.ErrEmailAlreadyExists) {
            c.JSON(http.StatusConflict, ErrorResponse{Error: "email already exists"})
            return
        }
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
        return
    }

    c.JSON(http.StatusCreated, RegisterResponse{
        User:  toUserResponse(user),
        Token: token,
    })
}

// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "User login"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /login [post]
func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
        return
    }

    token, err := h.authService.Login(req.Email, req.Password)
    if err != nil {
        c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid email or password"})
        return
    }

    c.JSON(http.StatusOK, LoginResponse{Token: token})
}