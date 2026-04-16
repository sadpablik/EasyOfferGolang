package consumer

import (
	"time"

	"easyoffer/interview/internal/domain"
)

const (
	EventQuestionCreated = "question.created"
	EventQuestionUpdated = "question.updated"
	EventQuestionDeleted = "question.deleted"
)

type QuestionEvent struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Version    int             `json:"version"`
	Payload    QuestionPayload `json:"payload"`
}

type QuestionPayload struct {
	QuestionID   string    `json:"question_id"`
	AuthorID     string    `json:"author_id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	Category     string    `json:"category"`
	AnswerFormat string    `json:"answer_format"`
	Language     string    `json:"language,omitempty"`
	StarterCode  string    `json:"starter_code,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func (p QuestionPayload) ToSnapshot() *domain.QuestionSnapshot {
	createdAt := ""
	if !p.CreatedAt.IsZero() {
		createdAt = p.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return &domain.QuestionSnapshot{
		ID:           p.QuestionID,
		Title:        p.Title,
		Content:      p.Content,
		Category:     p.Category,
		AnswerFormat: p.AnswerFormat,
		Language:     p.Language,
		StarterCode:  p.StarterCode,
		AuthorID:     p.AuthorID,
		CreatedAt:    createdAt,
	}
}
