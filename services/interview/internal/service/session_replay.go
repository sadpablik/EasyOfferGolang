package service

import (
	"context"
	"easyoffer/interview/internal/domain"
	"easyoffer/interview/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (s *interviewService) getOwnedSession(ctx context.Context, userID, sessionID string) (*domain.InterviewSession, error) {
	if userID == "" {
		return nil, ErrMissingUserID
	}

	session, err := s.repo.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			replayed, replayErr := s.rebuildSessionFromEvents(ctx, sessionID)
			if replayErr != nil {
				return nil, replayErr
			}
			session = replayed
		} else {
			return nil, fmt.Errorf("failed to load session: %w", err)
		}
	}

	if session.UserID != userID {
		return nil, ErrSessionForbidden
	}

	return session, nil
}

func (s *interviewService) rebuildSessionFromEvents(ctx context.Context, sessionID string) (*domain.InterviewSession, error) {
	events, err := s.events.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session events: %w", err)
	}
	if len(events) == 0 {
		return nil, ErrSessionNotFound
	}

	session, err := replaySessionFromEvents(events)
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild session from events: %w", err)
	}

	return session, nil
}

func (s *interviewService) projectEvent(ctx context.Context, event domain.InterviewEvent) (*domain.InterviewSession, error) {
	switch event.Type {
	case domain.EventSessionStarted:
		var payload domain.SessionStartedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("invalid session.started payload: %w", err)
		}

		startedAt := payload.StartedAt
		if startedAt.IsZero() {
			startedAt = event.OccurredAt
		}

		session := &domain.InterviewSession{
			ID:           event.SessionID,
			UserID:       event.UserID,
			Questions:    payload.Questions,
			CurrentIndex: 0,
			Answers:      make(map[string]domain.SessionAnswer),
			StartedAt:    startedAt,
		}

		if err := s.repo.Save(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to save projected session: %w", err)
		}

		return session, nil

	case domain.EventAnswerSubmitted:
		session, err := s.loadSessionForProjection(ctx, event.SessionID)
		if err != nil {
			return nil, err
		}

		var payload domain.AnswerSubmittedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("invalid answer.submitted payload: %w", err)
		}

		answeredAt := payload.AnsweredAt
		if answeredAt.IsZero() {
			answeredAt = event.OccurredAt
		}

		if session.Answers == nil {
			session.Answers = make(map[string]domain.SessionAnswer)
		}

		session.Answers[payload.QuestionID] = domain.SessionAnswer{
			QuestionID: payload.QuestionID,
			Status:     payload.Status,
			UserAnswer: payload.UserAnswer,
			Note:       payload.Note,
			AnsweredAt: answeredAt,
		}

		if idx := questionIndex(session.Questions, payload.QuestionID); idx >= 0 && idx >= session.CurrentIndex {
			session.CurrentIndex = idx
		}

		if err := s.repo.Save(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to save projected answer: %w", err)
		}

		return session, nil

	case domain.EventSessionFinished:
		session, err := s.loadSessionForProjection(ctx, event.SessionID)
		if err != nil {
			return nil, err
		}

		var payload domain.SessionFinishedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("invalid session.finished payload: %w", err)
		}

		finishedAt := payload.FinishedAt
		if finishedAt.IsZero() {
			finishedAt = event.OccurredAt
		}
		session.FinishedAt = &finishedAt

		if err := s.repo.Save(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to save projected finish: %w", err)
		}

		return session, nil

	default:
		return nil, fmt.Errorf("unsupported event type %q", event.Type)
	}
}

func (s *interviewService) loadSessionForProjection(ctx context.Context, sessionID string) (*domain.InterviewSession, error) {
	session, err := s.repo.Get(ctx, sessionID)
	if err == nil {
		return session, nil
	}

	if errors.Is(err, repository.ErrSessionNotFound) {
		replayed, replayErr := s.rebuildSessionFromEvents(ctx, sessionID)
		if replayErr != nil {
			return nil, replayErr
		}
		return replayed, nil
	}

	return nil, fmt.Errorf("failed to load session for projection: %w", err)
}

func replaySessionFromEvents(events []domain.InterviewEvent) (*domain.InterviewSession, error) {
	var session *domain.InterviewSession

	for i := range events {
		event := events[i]
		switch event.Type {
		case domain.EventSessionStarted:
			var payload domain.SessionStartedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("invalid session.started payload: %w", err)
			}

			startedAt := payload.StartedAt
			if startedAt.IsZero() {
				startedAt = event.OccurredAt
			}

			session = &domain.InterviewSession{
				ID:           event.SessionID,
				UserID:       event.UserID,
				Questions:    payload.Questions,
				CurrentIndex: 0,
				Answers:      make(map[string]domain.SessionAnswer),
				StartedAt:    startedAt,
			}

		case domain.EventAnswerSubmitted:
			if session == nil {
				return nil, errors.New("answer event before session start")
			}

			var payload domain.AnswerSubmittedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("invalid answer.submitted payload: %w", err)
			}

			answeredAt := payload.AnsweredAt
			if answeredAt.IsZero() {
				answeredAt = event.OccurredAt
			}

			session.Answers[payload.QuestionID] = domain.SessionAnswer{
				QuestionID: payload.QuestionID,
				Status:     payload.Status,
				UserAnswer: payload.UserAnswer,
				Note:       payload.Note,
				AnsweredAt: answeredAt,
			}

			if idx := questionIndex(session.Questions, payload.QuestionID); idx >= 0 && idx >= session.CurrentIndex {
				session.CurrentIndex = idx
			}

		case domain.EventSessionFinished:
			if session == nil {
				return nil, errors.New("finish event before session start")
			}

			var payload domain.SessionFinishedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("invalid session.finished payload: %w", err)
			}

			finishedAt := payload.FinishedAt
			if finishedAt.IsZero() {
				finishedAt = event.OccurredAt
			}
			session.FinishedAt = &finishedAt
		}
	}

	if session == nil {
		return nil, ErrSessionNotFound
	}

	if session.Answers == nil {
		session.Answers = make(map[string]domain.SessionAnswer)
	}

	if session.StartedAt.IsZero() {
		session.StartedAt = time.Now().UTC()
	}

	return session, nil
}

func questionIndex(questions []domain.QuestionSnapshot, questionID string) int {
	for i := range questions {
		if questions[i].ID == questionID {
			return i
		}
	}
	return -1
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
