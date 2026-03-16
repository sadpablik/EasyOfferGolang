package service_test

import (
	"errors"
	"testing"

	"easyoffer/question/internal/domain"
	"easyoffer/question/internal/repository"
	questionservice "easyoffer/question/internal/service"
)

type questionRepositoryStub struct {
	createErr error
}

func (s *questionRepositoryStub) Create(_ *domain.Question) error {
	return s.createErr
}

func (s *questionRepositoryStub) Update(_ *domain.Question) error {
	return nil
}

func (s *questionRepositoryStub) Delete(_ string) error {
	return nil
}

func (s *questionRepositoryStub) UpsertReview(_ *domain.QuestionReview) error {
	return nil
}

func (s *questionRepositoryStub) GetReviewsByUser(_, _ string) ([]*domain.QuestionReview, error) {
	return nil, nil
}

func (s *questionRepositoryStub) GetAll(_, _, _, _, _, _ string, _ bool, _, _ int, _, _ string) ([]*domain.Question, int64, error) {
	return nil, 0, nil
}

func (s *questionRepositoryStub) GetByID(_ string) (*domain.Question, error) {
	return nil, nil
}

func (s *questionRepositoryStub) GetReviewByUserAndQuestion(_, _ string) (*domain.QuestionReview, error) {
	return nil, nil
}

func (s *questionRepositoryStub) GetMyQuestions(_, _, _ string, _, _ int) ([]*domain.QuestionWithReview, int64, error) {
	return nil, 0, nil
}

func TestCreateQuestion_MapsDuplicateToServiceError(t *testing.T) {
	svc := questionservice.NewQuestionService(&questionRepositoryStub{createErr: repository.ErrQuestionAlreadyExists})

	question, err := svc.CreateQuestion(
		"Title",
		"Content",
		"author-1",
		string(domain.CategoryTheory),
		string(domain.AnswerFormatText),
		"go",
		"",
	)

	if !errors.Is(err, questionservice.ErrQuestionAlreadyExists) {
		t.Fatalf("expected ErrQuestionAlreadyExists, got: %v", err)
	}
	if question != nil {
		t.Fatalf("expected nil question on duplicate error")
	}
}
