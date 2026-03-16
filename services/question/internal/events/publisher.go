package events

import (
	"context"

	"easyoffer/question/internal/domain"
)

type Publisher interface {
	PublishQuestionCreated(ctx context.Context, question *domain.Question) error
	PublishQuestionUpdated(ctx context.Context, question *domain.Question) error
	PublishQuestionDeleted(ctx context.Context, questionID string) error
}

