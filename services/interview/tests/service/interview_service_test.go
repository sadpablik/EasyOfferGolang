package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"easyoffer/interview/internal/client"
	"easyoffer/interview/internal/domain"
	"easyoffer/interview/internal/repository"
	interviewservice "easyoffer/interview/internal/service"
)

type sessionRepositoryStub struct {
	session   *domain.InterviewSession
	getErr    error
	saveErr   error
	saveCalls int
	getCalls  int
}

func (s *sessionRepositoryStub) Save(_ context.Context, session *domain.InterviewSession) error {
	s.saveCalls++
	if s.saveErr != nil {
		return s.saveErr
	}
	s.session = session
	return nil
}

func (s *sessionRepositoryStub) Get(_ context.Context, _ string) (*domain.InterviewSession, error) {
	s.getCalls++
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.session == nil {
		return nil, repository.ErrSessionNotFound
	}
	return s.session, nil
}

func (s *sessionRepositoryStub) Delete(_ context.Context, _ string) error {
	return nil
}

type questionClientStub struct{}

func (q *questionClientStub) ListQuestions(_ context.Context, _ client.ListQuestionsParams) ([]domain.QuestionSnapshot, error) {
	return nil, nil
}

func TestGetNextQuestion_WhenAllQuestionsAnswered_ReturnsDoneStateWithoutError(t *testing.T) {
	now := time.Now().UTC()
	repo := &sessionRepositoryStub{
		session: &domain.InterviewSession{
			ID:           "session-1",
			UserID:       "user-1",
			Questions:    []domain.QuestionSnapshot{{ID: "q-1", Title: "Q1"}},
			CurrentIndex: 0,
			Answers: map[string]domain.SessionAnswer{
				"q-1": {QuestionID: "q-1", Status: domain.StatusKnow, AnsweredAt: now},
			},
			StartedAt: now,
		},
	}

	svc := interviewservice.NewInterviewService(repo, &questionClientStub{}, time.Minute)

	question, hasMore, err := svc.GetNextQuestion(context.Background(), "user-1", "session-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if question != nil {
		t.Fatalf("expected nil question when interview is done")
	}
	if hasMore {
		t.Fatalf("expected hasMore=false when interview is done")
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no repository save when interview is done")
	}
}

func TestSubmitAnswer_InvalidStatus_ReturnsErrInvalidStatus(t *testing.T) {
	repo := &sessionRepositoryStub{}
	svc := interviewservice.NewInterviewService(repo, &questionClientStub{}, time.Minute)

	err := svc.SubmitAnswer(context.Background(), "user-1", "session-1", interviewservice.SubmitAnswerInput{
		QuestionID: "q-1",
		Status:     "invalid-status",
	})

	if !errors.Is(err, interviewservice.ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got: %v", err)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no repository save for invalid status")
	}
	if repo.getCalls != 0 {
		t.Fatalf("expected no repository get for invalid status")
	}
}

func TestGetResult_BeforeFinish_ReturnsErrSessionNotFinished(t *testing.T) {
	now := time.Now().UTC()
	repo := &sessionRepositoryStub{
		session: &domain.InterviewSession{
			ID:        "session-1",
			UserID:    "user-1",
			Questions: []domain.QuestionSnapshot{{ID: "q-1"}},
			Answers:   map[string]domain.SessionAnswer{},
			StartedAt: now,
		},
	}

	svc := interviewservice.NewInterviewService(repo, &questionClientStub{}, time.Minute)

	result, err := svc.GetResult(context.Background(), "user-1", "session-1")
	if !errors.Is(err, interviewservice.ErrSessionNotFinished) {
		t.Fatalf("expected ErrSessionNotFinished, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result before finish")
	}
}
