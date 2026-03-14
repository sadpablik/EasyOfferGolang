package service

import (
	"easyoffer/auth/internal/domain"
	"easyoffer/auth/internal/repository"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrEmailAlreadyExists = errors.New("email already exists")


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
		Role: "user",
	}
	if err := user.HashPassword(password); err != nil {
		return nil, "", err
	}
	if err := s.repo.Create(user); err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			return nil, "", ErrEmailAlreadyExists
		}
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
	now := time.Now().UTC()

	claims := jwt.MapClaims{
        "sub": userID,                          // subject: id пользователя
        "iss": "easyoffer-auth",               // issuer
        "iat": now.Unix(),                     // issued at
        "nbf": now.Unix(),                     // not before
        "exp": now.Add(12 * time.Hour).Unix(), // expiration
    }
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}