package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"easyoffer/interview/internal/domain"

	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidInterviewEvent  = errors.New("invalid interview event")
	ErrInvalidInterviewStream = errors.New("invalid interview stream")
)

const interviewEventStreamPrefix = "interview:events:session:"

type EventStore interface {
	Append(ctx context.Context, event *domain.InterviewEvent) error
	ListBySession(ctx context.Context, sessionID string) ([]domain.InterviewEvent, error)
}

type redisEventStore struct {
	client *redis.Client
}

func NewRedisEventStore(client *redis.Client) EventStore {
	return &redisEventStore{client: client}
}

func (s *redisEventStore) Append(ctx context.Context, event *domain.InterviewEvent) error {
	if event == nil || strings.TrimSpace(event.SessionID) == "" || strings.TrimSpace(event.ID) == "" || strings.TrimSpace(string(event.Type)) == "" {
		return ErrInvalidInterviewEvent
	}

	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return s.client.RPush(ctx, interviewEventStreamKey(event.SessionID), raw).Err()
}

func (s *redisEventStore) ListBySession(ctx context.Context, sessionID string) ([]domain.InterviewEvent, error) {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return nil, ErrInvalidInterviewStream
	}

	items, err := s.client.LRange(ctx, interviewEventStreamKey(id), 0, -1).Result()
	if err != nil {
		return nil, err
	}

	events := make([]domain.InterviewEvent, 0, len(items))
	for _, item := range items {
		var evt domain.InterviewEvent
		if err := json.Unmarshal([]byte(item), &evt); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, nil
}

func interviewEventStreamKey(sessionID string) string {
	return interviewEventStreamPrefix + strings.TrimSpace(sessionID)
}

type NoopEventStore struct{}

func NewNoopEventStore() EventStore {
	return &NoopEventStore{}
}

func (s *NoopEventStore) Append(_ context.Context, _ *domain.InterviewEvent) error {
	return nil
}

func (s *NoopEventStore) ListBySession(_ context.Context, _ string) ([]domain.InterviewEvent, error) {
	return []domain.InterviewEvent{}, nil
}
