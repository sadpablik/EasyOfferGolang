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
	ErrInvalidQuestionSnapshot = errors.New("invalid question snapshot")
	ErrInvalidQuestionID       = errors.New("invalid question id")
	ErrInvalidEventID          = errors.New("invalid event id")
)

type QuestionFilter struct {
	Category     string
	AnswerFormat string
	Language     string
	Limit        int
}

type QuestionRepository interface {
	Upsert(ctx context.Context, question *domain.QuestionSnapshot) error
	DeleteQuestion(ctx context.Context, questionID string) error
	List(ctx context.Context, filter QuestionFilter) ([]domain.QuestionSnapshot, error)
}

type EventDedupStore interface {
	MarkEventProcessed(ctx context.Context, eventID string, ttl time.Duration) (bool, error)
}

const (
	questionKeyPrefix       = "interview:questions:"
	questionIndexKey        = "interview:questions:index"
	processedEventKeyPrefix = "interview:questions:events:processed:"
)

type redisQuestionRepository struct {
	client *redis.Client
}

func NewRedisQuestionRepository(client *redis.Client) QuestionRepository {
	return &redisQuestionRepository{client: client}
}

func (r *redisQuestionRepository) Upsert(ctx context.Context, question *domain.QuestionSnapshot) error {
	if question == nil || strings.TrimSpace(question.ID) == "" {
		return ErrInvalidQuestionSnapshot
	}

	payload, err := json.Marshal(question)
	if err != nil {
		return err
	}

	pipe := r.client.TxPipeline()
	pipe.Set(ctx, questionKey(question.ID), payload, 0)
	pipe.SAdd(ctx, questionIndexKey, question.ID)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *redisQuestionRepository) DeleteQuestion(ctx context.Context, questionID string) error {
	id := strings.TrimSpace(questionID)
	if id == "" {
		return ErrInvalidQuestionID
	}

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, questionKey(id))
	pipe.SRem(ctx, questionIndexKey, id)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisQuestionRepository) MarkEventProcessed(ctx context.Context, eventID string, ttl time.Duration) (bool, error) {
	id := strings.TrimSpace(eventID)
	if id == "" {
		return false, ErrInvalidEventID
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	created, err := r.client.SetNX(ctx, processedEventKey(id), "1", ttl).Result()
	if err != nil {
		return false, err
	}

	return created, nil
}

func (r *redisQuestionRepository) List(ctx context.Context, filter QuestionFilter) ([]domain.QuestionSnapshot, error) {
	ids, err := r.client.SMembers(ctx, questionIndexKey).Result()
	if err != nil {
		return nil, err
	}

	questions := make([]domain.QuestionSnapshot, 0, len(ids))
	for _, id := range ids {
		payload, err := r.client.Get(ctx, questionKey(id)).Bytes()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return nil, err
		}

		var question domain.QuestionSnapshot
		if err := json.Unmarshal(payload, &question); err != nil {
			return nil, err
		}

		if !matchesQuestionFilter(question, filter) {
			continue
		}

		questions = append(questions, question)
	}

	sort.Slice(questions, func(i, j int) bool {
		return questions[i].CreatedAt > questions[j].CreatedAt
	})

	if filter.Limit > 0 && len(questions) > filter.Limit {
		questions = questions[:filter.Limit]
	}

	return questions, nil
}

func questionKey(questionID string) string {
	return questionKeyPrefix + questionID
}

func processedEventKey(eventID string) string {
	return processedEventKeyPrefix + eventID
}

func matchesQuestionFilter(question domain.QuestionSnapshot, filter QuestionFilter) bool {
	category := strings.TrimSpace(filter.Category)
	if category != "" && question.Category != category {
		return false
	}

	answerFormat := strings.TrimSpace(filter.AnswerFormat)
	if answerFormat != "" && question.AnswerFormat != answerFormat {
		return false
	}

	language := strings.TrimSpace(filter.Language)
	if language != "" && question.Language != language {
		return false
	}

	return true
}
