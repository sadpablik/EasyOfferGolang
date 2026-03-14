package service

import (
    "easyoffer/auth/internal/domain"
    "easyoffer/auth/internal/repository"
    "github.com/golang-jwt/jwt/v5"
    "time"
	"errors"
	"github.com/google/uuid"
)


type AuthService interface {
	Register(email, password string) (*domain.User, string, error)
	Login(email, password string) (string, error)
}

type authService struct {
	repo repository.UserRepository
	jwtSecret string
}

func NewAuthService(repo repository.UserRepository, jwtSecret string) AuthService {
	return &authService{repo: repo, jwtSecret: jwtSecret}
}


func (s *authService) Register(email, password string) (*domain.User, string, error) {
	user := &domain.User{
		ID: uuid.New().String(),
		Email:email,
		CreatedAt: time.Now(),
	}
	if err := user.HashPassword(password); err != nil {
		return nil, "", err
	}
	if err := s.repo.Create(user); err != nil {
		return nil, "", err
	}
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil,"", err
	}
	return user, token, nil

}	


func (s *authService) Login(email, password string) (string, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil || !user.CheckPassword(password) {
        return "", errors.New("invalid email or password")
	}
	return s.generateToken(user.ID)
}

func (s *authService) generateToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp": time.Now().Add(12 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(s.jwtSecret))
}