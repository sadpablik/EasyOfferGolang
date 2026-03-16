package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"easyoffer/interview/internal/domain"

	"github.com/redis/go-redis/v9"
)

var (
	ErrNotImplemented  = errors.New("session repository is not implemented")
	ErrSessionNotFound = errors.New("session not found")
)

type SessionRepository interface {
	Save(ctx context.Context, session *domain.InterviewSession) error
	Get(ctx context.Context, sessionID string) (*domain.InterviewSession, error)
	Delete(ctx context.Context, sessionID string) error
}

const sessionKeyPrefix = "interview:sessions:"

type redisSessionRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisSessionRepository(client *redis.Client, ttl time.Duration) SessionRepository {
	if ttl <= 0 {
		ttl = 2 * time.Hour
	}

	return &redisSessionRepository{
		client: client,
		ttl:    ttl,
	}
}

func (r *redisSessionRepository) Save(ctx context.Context, session *domain.InterviewSession) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, sessionKey(session.ID), payload, r.ttl).Err()
}

func (r *redisSessionRepository) Get(ctx context.Context, sessionID string) (*domain.InterviewSession, error) {
	payload, err := r.client.Get(ctx, sessionKey(sessionID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	var session domain.InterviewSession
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, err
	}
	if session.Answers == nil {
		session.Answers = make(map[string]domain.SessionAnswer)
	}

	return &session, nil
}

func (r *redisSessionRepository) Delete(ctx context.Context, sessionID string) error {
	return r.client.Del(ctx, sessionKey(sessionID)).Err()
}

func sessionKey(sessionID string) string {
	return sessionKeyPrefix + sessionID
}

type NoopSessionRepository struct{}

func NewNoopSessionRepository() SessionRepository {
	return &NoopSessionRepository{}
}

func (r *NoopSessionRepository) Save(_ context.Context, _ *domain.InterviewSession) error {
	return ErrNotImplemented
}

func (r *NoopSessionRepository) Get(_ context.Context, _ string) (*domain.InterviewSession, error) {
	return nil, ErrNotImplemented
}

func (r *NoopSessionRepository) Delete(_ context.Context, _ string) error {
	return ErrNotImplemented
}
