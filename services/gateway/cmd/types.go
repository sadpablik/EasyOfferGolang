package main

import "net/http"

// @title EasyOffer Gateway API
// @version 1.0
// @description Public API exposed by the Gateway service.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

type gateway struct {
	client      *http.Client
	authURL     string
	questionURL string
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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

type HealthResponse struct {
	Status string `json:"status"`
}

type CreateQuestionRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	AuthorID string `json:"author_id"`
}

type QuestionResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	AuthorID  string `json:"author_id"`
	CreatedAt string `json:"created_at"`
}
