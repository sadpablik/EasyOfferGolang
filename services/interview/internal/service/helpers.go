package service

import (
    "context"
    "errors"
    "fmt"
    "easyoffer/interview/internal/domain"
    "easyoffer/interview/internal/repository"
)

func (s *interviewService) getOwnedSession(ctx context.Context, userID, sessionID string) (*domain.InterviewSession, error) {
    if userID == "" {
        return nil, ErrMissingUserID
    }

    session, err := s.repo.Get(ctx, sessionID)
    if err != nil {
        if errors.Is(err, repository.ErrSessionNotFound) {
            return nil, ErrSessionNotFound
        }
        return nil, fmt.Errorf("failed to load session: %w", err)
    }

    if session.UserID != userID {
        return nil, ErrSessionForbidden
    }

    return session, nil
}

func buildResult(session *domain.InterviewSession) *domain.InterviewResult {
	result := &domain.InterviewResult{
		SessionID:  session.ID,
		Total:      len(session.Questions),
		Answered:   len(session.Answers),
		FinishedAt: *session.FinishedAt,
	}
	for _, answer := range session.Answers {
		switch answer.Status {
		case domain.StatusKnow:
			result.Know++
		case domain.StatusDontKnow:
			result.DontKnow++
		case domain.StatusRepeat:
			result.Repeat++
		}
	}
	return result
}

// — Noop —

type NoopInterviewService struct{}

func NewNoopInterviewService() InterviewService {
	return &NoopInterviewService{}
}

func (s *NoopInterviewService) StartSession(_ context.Context, _ string, _ StartSessionInput) (*domain.InterviewSession, *domain.QuestionSnapshot, error) {
	return nil, nil, ErrNotImplemented
}

func (s *NoopInterviewService) GetNextQuestion(_ context.Context, _ string, _ string) (*domain.QuestionSnapshot, bool, error) {
	return nil, false, ErrNotImplemented
}

func (s *NoopInterviewService) SubmitAnswer(_ context.Context, _ string, _ string, _ SubmitAnswerInput) error {
	return ErrNotImplemented
}

func (s *NoopInterviewService) FinishSession(_ context.Context, _ string, _ string) (*domain.InterviewResult, error) {
	return nil, ErrNotImplemented
}

func (s *NoopInterviewService) GetResult(_ context.Context, _ string, _ string) (*domain.InterviewResult, error) {
	return nil, ErrNotImplemented
}
