package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

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
	ReplaySession(ctx context.Context, userID, sessionID string) (*domain.InterviewSession, error)
}

type interviewService struct {
	repo       repository.SessionRepository
	questions  repository.QuestionRepository
	events     repository.EventStore
	sessionTTL time.Duration
}

func NewInterviewService(
	repo repository.SessionRepository,
	questions repository.QuestionRepository,
	events repository.EventStore,
	sessionTTL time.Duration,
) InterviewService {
	return &interviewService{
		repo:       repo,
		questions:  questions,
		events:     events,
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
	questions, err := s.questions.List(ctx, repository.QuestionFilter{
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
	// Shuffle so each new session gets questions in random order.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(questions), func(i, j int) { questions[i], questions[j] = questions[j], questions[i] })
	startedAt := time.Now().UTC()

	session := &domain.InterviewSession{
		ID:           uuid.NewString(),
		UserID:       userID,
		Questions:    questions,
		CurrentIndex: 0,
		Answers:      make(map[string]domain.SessionAnswer),
		StartedAt:    startedAt,
	}

	event, err := s.appendEvent(ctx, session.ID, userID, domain.EventSessionStarted, domain.SessionStartedPayload{
		Category:       input.Category,
		AnswerFormat:   input.AnswerFormat,
		Language:       input.Language,
		RequestedCount: count,
		Questions:      session.Questions,
		StartedAt:      startedAt,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to append session started event: %w", err)
	}

	projected, err := s.projectEvent(ctx, *event)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to project session started event: %w", err)
	}
	if len(projected.Questions) == 0 {
		return nil, nil, ErrNoQuestionsAvailable
	}

	firstQuestion := &projected.Questions[0]
	return projected, firstQuestion, nil
}

func (s *interviewService) GetNextQuestion(ctx context.Context, userID, sessionID string) (*domain.QuestionSnapshot, bool, error) {
	session, err := s.getOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return nil, false, err
	}

	if session.FinishedAt != nil {
		return nil, false, ErrSessionFinished
	}

	for i := 0; i < len(session.Questions); i++ {
		if _, answered := session.Answers[session.Questions[i].ID]; answered {
			continue
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
	answeredAt := time.Now().UTC()

	event, err := s.appendEvent(ctx, session.ID, session.UserID, domain.EventAnswerSubmitted, domain.AnswerSubmittedPayload{
		QuestionID: input.QuestionID,
		Status:     status,
		UserAnswer: input.UserAnswer,
		Note:       input.Note,
		AnsweredAt: answeredAt,
	})
	if err != nil {
		return fmt.Errorf("failed to append answer submitted event: %w", err)
	}

	if _, err := s.projectEvent(ctx, *event); err != nil {
		return fmt.Errorf("failed to project answer to session: %w", err)
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
	_, err = s.appendEvent(ctx, session.ID, session.UserID, domain.EventSessionFinished, domain.SessionFinishedPayload{
		FinishedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to append session finished event: %w", err)
	}

	finishedSession := *session
	finishedSession.FinishedAt = &now
	if err := s.repo.Save(ctx, &finishedSession); err != nil {
		return nil, fmt.Errorf("failed to persist finished session: %w", err)
	}

	result := buildResult(&finishedSession)
	return result, nil
}

func (s *interviewService) appendEvent(ctx context.Context, sessionID, userID string, eventType domain.InterviewEventType, payload interface{}) (*domain.InterviewEvent, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	event := &domain.InterviewEvent{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		UserID:     userID,
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		Version:    1,
		Payload:    rawPayload,
	}

	if err := s.events.Append(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
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

func (s *interviewService) ReplaySession(ctx context.Context, userID, sessionID string) (*domain.InterviewSession, error) {
	if userID == "" {
		return nil, ErrMissingUserID
	}

	session, err := s.rebuildSessionFromEvents(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.UserID != userID {
		return nil, ErrSessionForbidden
	}

	if err := s.repo.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to persist replayed session: %w", err)
	}

	return session, nil
}
