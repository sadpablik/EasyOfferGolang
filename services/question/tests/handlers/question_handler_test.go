package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"easyoffer/question/api/handlers"
	"easyoffer/question/internal/domain"
	"easyoffer/question/internal/service"

	"github.com/gin-gonic/gin"
)

type questionServiceStub struct {
	createErr error
}

func (s *questionServiceStub) CreateQuestion(_, _, _, _, _, _, _ string) (*domain.Question, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	return &domain.Question{ID: "q-1"}, nil
}

func (s *questionServiceStub) PatchQuestion(_, _ string, _, _, _, _, _, _ *string) (*domain.Question, error) {
	return nil, nil
}

func (s *questionServiceStub) Delete(_, _ string) error {
	return nil
}

func (s *questionServiceStub) ReviewQuestion(_, _, _, _, _ string) (*domain.QuestionReview, error) {
	return nil, nil
}

func (s *questionServiceStub) GetQuestions(_, _, _, _, _, _ string, _ bool, _, _ int, _, _ string) ([]*domain.Question, int64, error) {
	return nil, 0, nil
}

func (s *questionServiceStub) GetQuestionByID(_ string) (*domain.Question, error) {
	return nil, nil
}

func (s *questionServiceStub) GetMyReviews(_, _ string) ([]*domain.QuestionReview, error) {
	return nil, nil
}

func (s *questionServiceStub) GetMyQuestionReview(_, _ string) (*domain.QuestionReview, error) {
	return nil, nil
}

func (s *questionServiceStub) GetMyQuestions(_, _, _ string, _, _ int) ([]*domain.QuestionWithReview, int64, error) {
	return nil, 0, nil
}

func TestCreateQuestion_MapsDuplicateErrorToConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := handlers.NewQuestionHandler(&questionServiceStub{createErr: service.ErrQuestionAlreadyExists})
	router := gin.New()
	router.POST("/questions", handler.CreateQuestion)

	body := map[string]any{
		"title":         "Binary Search",
		"content":       "Explain complexity",
		"category":      "theory",
		"answer_format": "text",
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/questions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-1")

	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", res.Code)
	}

	var payload handlers.ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Error != "question already exists" {
		t.Fatalf("unexpected error message: %q", payload.Error)
	}
}
