package service

import (
	"easyoffer/question/internal/domain"
	"easyoffer/question/internal/repository"
	"strings"
	"time"

	"github.com/google/uuid"
)

type QuestionService interface {
	CreateQuestion(title, content, authorID string) (*domain.Question, error)
}

type questionService struct{
	repo repository.QuestionRepository
}

func NewQuestionService(repo repository.QuestionRepository) QuestionService {
	return &questionService{repo: repo}
}

func (s *questionService) CreateQuestion(title, content, authorID string) (*domain.Question, error) {

	q := &domain.Question{
		ID:        uuid.NewString(),
		Title:     strings.TrimSpace(title),
		Content:   strings.TrimSpace(content),
		AuthorID:  strings.TrimSpace(authorID),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.Create(q); err != nil {
		return nil, err
	}
	return q, nil
}
