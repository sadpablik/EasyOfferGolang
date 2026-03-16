package handlers

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
