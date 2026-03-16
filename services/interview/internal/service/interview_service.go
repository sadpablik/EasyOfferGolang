package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"easyoffer/interview/internal/client"
	"easyoffer/interview/internal/domain"
	"easyoffer/interview/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrMissingUserID        = errors.New("missing user id")
	ErrInvalidCount         = errors.New("question count must be between 1 and 50")
	ErrInvalidStatus        = errors.New("invalid review status")
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionForbidden     = errors.New("session belongs to another user")
	ErrSessionFinished      = errors.New("session is already finished")
	ErrSessionNotFinished   = errors.New("session is not finished yet")
	ErrQuestionNotInSession = errors.New("question does not belong to this session")
	ErrNoQuestionsAvailable = errors.New("no questions available")
	ErrNotImplemented       = errors.New("interview service is not implemented")
)

type StartSessionInput struct {
	Category     string
	AnswerFormat string
	Language     string
	Count        int
}

type SubmitAnswerInput struct {
	QuestionID string
	Status     string
	UserAnswer string
	Note       string
}

type InterviewService interface {
	StartSession(ctx context.Context, userID string, input StartSessionInput) (*domain.InterviewSession, *domain.QuestionSnapshot, error)
	GetNextQuestion(ctx context.Context, userID, sessionID string) (*domain.QuestionSnapshot, bool, error)
	SubmitAnswer(ctx context.Context, userID, sessionID string, input SubmitAnswerInput) error
	FinishSession(ctx context.Context, userID, sessionID string) (*domain.InterviewResult, error)
	GetResult(ctx context.Context, userID, sessionID string) (*domain.InterviewResult, error)
}

type interviewService struct {
	repo       repository.SessionRepository
	client     client.QuestionClient
	sessionTTL time.Duration
}

func NewInterviewService(
	repo repository.SessionRepository,
	qClient client.QuestionClient,
	sessionTTL time.Duration,
) InterviewService {
	return &interviewService{
		repo:       repo,
		client:     qClient,
		sessionTTL: sessionTTL,
	}
}

func (s *interviewService) StartSession(ctx context.Context, userID string, input StartSessionInput) (*domain.InterviewSession, *domain.QuestionSnapshot, error) {
	if userID == "" {
		return nil, nil, ErrMissingUserID
	}
	count := input.Count
	if count <= 0 {
		count = 10
	}
	if count > 50 {
		return nil, nil, ErrInvalidCount
	}
	questions, err := s.client.ListQuestions(ctx, client.ListQuestionsParams{
		UserID:       userID,
		Category:     input.Category,
		AnswerFormat: input.AnswerFormat,
		Language:     input.Language,
		Limit:        count,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch questions: %w", err)
	}
	if len(questions) == 0 {
		return nil, nil, ErrNoQuestionsAvailable
	}

	session := &domain.InterviewSession{
		ID:           uuid.NewString(),
		UserID:       userID,
		Questions:    questions,
		CurrentIndex: 0,
		Answers:      make(map[string]domain.SessionAnswer),
		StartedAt:    time.Now().UTC(),
	}

	if err := s.repo.Save(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("failed to save session: %w", err)
	}

	firstQuestion := &session.Questions[0]
	return session, firstQuestion, nil
}

func (s *interviewService) GetNextQuestion(ctx context.Context, userID, sessionID string) (*domain.QuestionSnapshot, bool, error) {
	session, err := s.getOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return nil, false, err
	}

	if session.FinishedAt != nil {
		return nil, false, ErrSessionFinished
	}

	for i := session.CurrentIndex; i < len(session.Questions); i++ {
		if _, answered := session.Answers[session.Questions[i].ID]; answered {
			continue
		}

		session.CurrentIndex = i
		if err := s.repo.Save(ctx, session); err != nil {
			return nil, false, fmt.Errorf("failed to update session index: %w", err)
		}
		return &session.Questions[i], true, nil
	}

	return nil, false, nil
}

func (s *interviewService) SubmitAnswer(ctx context.Context, userID, sessionID string, input SubmitAnswerInput) error {
	status := domain.ReviewStatus(input.Status)
	if status != domain.StatusKnow && status != domain.StatusDontKnow && status != domain.StatusRepeat {
		return ErrInvalidStatus
	}
	session, err := s.getOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if session.FinishedAt != nil {
		return ErrSessionFinished
	}
	belongs := false
	for _, q := range session.Questions {
		if q.ID == input.QuestionID {
			belongs = true
			break
		}
	}
	if !belongs {
		return ErrQuestionNotInSession
	}

	session.Answers[input.QuestionID] = domain.SessionAnswer{
		QuestionID: input.QuestionID,
		Status:     status,
		UserAnswer: input.UserAnswer,
		Note:       input.Note,
		AnsweredAt: time.Now().UTC(),
	}

	if err := s.repo.Save(ctx, session); err != nil {
		return fmt.Errorf("failed to save answer: %w", err)
	}
	return nil
}

func (s *interviewService) FinishSession(ctx context.Context, userID, sessionID string) (*domain.InterviewResult, error) {
	session, err := s.getOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if session.FinishedAt != nil {
		return nil, ErrSessionFinished
	}

	now := time.Now().UTC()
	session.FinishedAt = &now

	if err := s.repo.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to finish session: %w", err)
	}

	result := buildResult(session)
	return result, nil
}

func (s *interviewService) GetResult(ctx context.Context, userID, sessionID string) (*domain.InterviewResult, error) {
	session, err := s.getOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if session.FinishedAt == nil {
		return nil, ErrSessionNotFinished
	}

	result := buildResult(session)
	return result, nil
}
