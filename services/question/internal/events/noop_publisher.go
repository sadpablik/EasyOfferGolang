package events

import (
	"context"

	"easyoffer/question/internal/domain"
)

type NoopPublisher struct{}

func NewNoopPublisher() Publisher {
	return &NoopPublisher{}
}

func (p *NoopPublisher) PublishQuestionCreated(ctx context.Context, question *domain.Question) error {
	return nil
}

func (p *NoopPublisher) PublishQuestionUpdated(ctx context.Context, question *domain.Question) error {
	return nil
}

func (p *NoopPublisher) PublishQuestionDeleted(ctx context.Context, questionID string) error {
	return nil
}

func (p *NoopPublisher) Close() error {
	return nil
}
