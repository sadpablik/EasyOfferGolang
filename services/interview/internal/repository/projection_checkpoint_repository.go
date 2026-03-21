package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

const projectionCheckpointKeyPrefix = "interview:projection:checkpoint:"

type ProjectionCheckpointRepository interface {
	Get(ctx context.Context, sessionID string) (int64, error)
	Set(ctx context.Context, sessionID string, eventCount int64) error
}

type redisProjectionCheckpointRepository struct {
	client *redis.Client
}

func NewRedisProjectionCheckpointRepository(client *redis.Client) ProjectionCheckpointRepository {
	return &redisProjectionCheckpointRepository{client: client}
}

func (r *redisProjectionCheckpointRepository) Get(ctx context.Context, sessionID string) (int64, error) {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return 0, ErrInvalidInterviewStream
	}

	raw, err := r.client.Get(ctx, projectionCheckpointKey(id)).Result()
	if err == redis.Nil {
		observeRedis("projection_checkpoint_get", "miss")
		return 0, nil
	}
	if err != nil {
		observeRedis("projection_checkpoint_get", "error")
		return 0, err
	}

	value, convErr := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if convErr != nil {
		observeRedis("projection_checkpoint_get", "error")
		return 0, fmt.Errorf("invalid checkpoint value for session %s: %w", id, convErr)
	}

	observeRedis("projection_checkpoint_get", "hit")
	return value, nil
}

func (r *redisProjectionCheckpointRepository) Set(ctx context.Context, sessionID string, eventCount int64) error {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return ErrInvalidInterviewStream
	}
	if eventCount < 0 {
		eventCount = 0
	}

	if err := r.client.Set(ctx, projectionCheckpointKey(id), strconv.FormatInt(eventCount, 10), 0).Err(); err != nil {
		observeRedis("projection_checkpoint_set", "error")
		return err
	}

	observeRedis("projection_checkpoint_set", "success")
	return nil
}

func projectionCheckpointKey(sessionID string) string {
	return projectionCheckpointKeyPrefix + strings.TrimSpace(sessionID)
}

