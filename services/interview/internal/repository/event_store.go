package repository

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"easyoffer/interview/internal/domain"

	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidInterviewEvent  = errors.New("invalid interview event")
	ErrInvalidInterviewStream = errors.New("invalid interview stream")
)

const (
	interviewEventStreamPrefix = "interview:events:session:"
	interviewEventSessionsKey  = "interview:events:sessions"
)

type EventStore interface {
	Append(ctx context.Context, event *domain.InterviewEvent) error
	ListBySession(ctx context.Context, sessionID string) ([]domain.InterviewEvent, error)
}

type ProjectorEventStore interface {
	EventStore
	ListSessions(ctx context.Context) ([]string, error)
	EventCount(ctx context.Context, sessionID string) (int64, error)
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
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	streamKey := interviewEventStreamKey(event.SessionID)
	txf := func(tx *redis.Tx) error {
		size, err := tx.LLen(ctx, streamKey).Result()
		if err != nil {
			return err
		}

		event.Version = int(size) + 1

		raw, err := json.Marshal(event)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.RPush(ctx, streamKey, raw)
			pipe.SAdd(ctx, interviewEventSessionsKey, strings.TrimSpace(event.SessionID))
			return nil
		})

		return err
	}

	for i := 0; i < 5; i++ {
		err := s.client.Watch(ctx, txf, streamKey)
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return err
	}

	return redis.TxFailedErr
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

func (s *redisEventStore) ListSessions(ctx context.Context) ([]string, error) {
	items, err := s.client.SMembers(ctx, interviewEventSessionsKey).Result()
	if err != nil {
		return nil, err
	}

	sessions := make([]string, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item)
		if id == "" {
			continue
		}
		sessions = append(sessions, id)
	}

	sort.Strings(sessions)
	return sessions, nil
}

func (s *redisEventStore) EventCount(ctx context.Context, sessionID string) (int64, error) {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return 0, ErrInvalidInterviewStream
	}

	return s.client.LLen(ctx, interviewEventStreamKey(id)).Result()
}

func interviewEventStreamKey(sessionID string) string {
	return interviewEventStreamPrefix + strings.TrimSpace(sessionID)
}

