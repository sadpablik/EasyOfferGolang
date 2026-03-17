package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"easyoffer/interview/api/handlers"
	"easyoffer/interview/internal/domain"
	"easyoffer/interview/internal/service"

	"github.com/gin-gonic/gin"
)

type interviewServiceStub struct {
	startErr  error
	nextErr   error
	submitErr error
	resultErr error
	replayErr error
}

func (s *interviewServiceStub) StartSession(_ context.Context, _ string, _ service.StartSessionInput) (*domain.InterviewSession, *domain.QuestionSnapshot, error) {
	if s.startErr != nil {
		return nil, nil, s.startErr
	}
	return &domain.InterviewSession{ID: "s-1", Questions: []domain.QuestionSnapshot{{ID: "q-1"}}}, &domain.QuestionSnapshot{ID: "q-1"}, nil
}

func (s *interviewServiceStub) GetNextQuestion(_ context.Context, _, _ string) (*domain.QuestionSnapshot, bool, error) {
	if s.nextErr != nil {
		return nil, false, s.nextErr
	}
	return nil, false, nil
}

func (s *interviewServiceStub) SubmitAnswer(_ context.Context, _, _ string, _ service.SubmitAnswerInput) error {
	return s.submitErr
}

func (s *interviewServiceStub) FinishSession(_ context.Context, _, _ string) (*domain.InterviewResult, error) {
	return &domain.InterviewResult{}, nil
}

func (s *interviewServiceStub) GetResult(_ context.Context, _, _ string) (*domain.InterviewResult, error) {
	if s.resultErr != nil {
		return nil, s.resultErr
	}
	return &domain.InterviewResult{}, nil
}

func (s *interviewServiceStub) ReplaySession(_ context.Context, _, _ string) (*domain.InterviewSession, error) {
	if s.replayErr != nil {
		return nil, s.replayErr
	}
	return &domain.InterviewSession{
		ID:        "s-1",
		Questions: []domain.QuestionSnapshot{{ID: "q-1"}, {ID: "q-2"}},
		Answers: map[string]domain.SessionAnswer{
			"q-1": {QuestionID: "q-1", Status: domain.StatusKnow},
		},
	}, nil
}

func TestErrorMapping_InvalidCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewInterviewHandler(&interviewServiceStub{startErr: service.ErrInvalidCount})
	r := gin.New()
	r.POST("/interviews/start", h.StartInterview)

	req := httptest.NewRequest(http.MethodPost, "/interviews/start", bytes.NewBufferString(`{"count":100}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-1")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
	var payload handlers.ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error != "question count must be between 1 and 50" {
		t.Fatalf("unexpected error message: %q", payload.Error)
	}
}

func TestErrorMapping_SessionNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewInterviewHandler(&interviewServiceStub{nextErr: service.ErrSessionNotFound})
	r := gin.New()
	r.GET("/interviews/:id/next", h.NextQuestion)

	req := httptest.NewRequest(http.MethodGet, "/interviews/s-1/next", nil)
	req.Header.Set("X-User-ID", "user-1")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
	var payload handlers.ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error != "session not found" {
		t.Fatalf("unexpected error message: %q", payload.Error)
	}
}

func TestErrorMapping_InvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewInterviewHandler(&interviewServiceStub{submitErr: service.ErrInvalidStatus})
	r := gin.New()
	r.POST("/interviews/:id/answer", h.SubmitAnswer)

	req := httptest.NewRequest(http.MethodPost, "/interviews/s-1/answer", bytes.NewBufferString(`{"question_id":"q-1","status":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-1")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
	var payload handlers.ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error != "invalid review status" {
		t.Fatalf("unexpected error message: %q", payload.Error)
	}
}

func TestErrorMapping_SessionNotFinished(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewInterviewHandler(&interviewServiceStub{resultErr: service.ErrSessionNotFinished})
	r := gin.New()
	r.GET("/interviews/:id/result", h.GetResult)

	req := httptest.NewRequest(http.MethodGet, "/interviews/s-1/result", nil)
	req.Header.Set("X-User-ID", "user-1")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.Code)
	}
	var payload handlers.ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error != "session is not finished yet" {
		t.Fatalf("unexpected error message: %q", payload.Error)
	}
}

func TestReplayInterview_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewInterviewHandler(&interviewServiceStub{})
	r := gin.New()
	r.POST("/interviews/:id/replay", h.ReplayInterview)

	req := httptest.NewRequest(http.MethodPost, "/interviews/s-1/replay", nil)
	req.Header.Set("X-User-ID", "user-1")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var payload handlers.ReplayInterviewResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.SessionID != "s-1" {
		t.Fatalf("unexpected session id: %q", payload.SessionID)
	}
	if !payload.Replayed {
		t.Fatalf("expected replayed=true")
	}
	if payload.Total != 2 || payload.Answered != 1 {
		t.Fatalf("unexpected replay payload: %#v", payload)
	}
}

func TestReplayInterview_WhenSessionMissing_ReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewInterviewHandler(&interviewServiceStub{replayErr: service.ErrSessionNotFound})
	r := gin.New()
	r.POST("/interviews/:id/replay", h.ReplayInterview)

	req := httptest.NewRequest(http.MethodPost, "/interviews/s-1/replay", nil)
	req.Header.Set("X-User-ID", "user-1")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
	var payload handlers.ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error != "session not found" {
		t.Fatalf("unexpected error message: %q", payload.Error)
	}
}
