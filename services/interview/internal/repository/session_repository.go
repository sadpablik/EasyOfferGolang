package repository

import (
	"context"
	"easyoffer/interview/internal/domain"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

var (
	ErrNotImplemented  = errors.New("session repository is not implemented")
	ErrSessionNotFound = errors.New("session not found")
)

var redisOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "easyoffer",
		Subsystem: "interview",
		Name:      "redis_operations_total",
		Help:      "Total number of Redis operations by type and status.",
	},
	[]string{"operation", "status"},
)
var registerRedisMetricsOnce sync.Once

func RegisterMetrics(reg prometheus.Registerer) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	registerRedisMetricsOnce.Do(func() {
		reg.MustRegister(redisOperationsTotal)
	})
}

func observeRedis(operation, outcome string) {
	redisOperationsTotal.WithLabelValues(operation, outcome).Inc()
}

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
		observeRedis("session_save", "error")
		return err
	}

	if err := r.client.Set(ctx, sessionKey(session.ID), payload, r.ttl).Err(); err != nil {
		observeRedis("session_save", "error")
		return err
	}

	observeRedis("session_save", "success")
	return nil
}

func (r *redisSessionRepository) Get(ctx context.Context, sessionID string) (*domain.InterviewSession, error) {
	payload, err := r.client.Get(ctx, sessionKey(sessionID)).Bytes()
	if errors.Is(err, redis.Nil) {
		observeRedis("session_get", "miss")
		return nil, ErrSessionNotFound
	}
	if err != nil {
		observeRedis("session_get", "error")
		return nil, err
	}

	var session domain.InterviewSession
	if err := json.Unmarshal(payload, &session); err != nil {
		observeRedis("session_get", "error")
		return nil, err
	}
	if session.Answers == nil {
		session.Answers = make(map[string]domain.SessionAnswer)
	}

	observeRedis("session_get", "hit")
	return &session, nil
}

func (r *redisSessionRepository) Delete(ctx context.Context, sessionID string) error {
	if err := r.client.Del(ctx, sessionKey(sessionID)).Err(); err != nil {
		observeRedis("session_delete", "error")
		return err
	}

	observeRedis("session_delete", "success")
	return nil
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
