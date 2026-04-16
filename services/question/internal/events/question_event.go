package events

import (
	"time"

	"easyoffer/question/internal/domain"
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
	QuestionID   string                  `json:"question_id"`
	AuthorID     string                  `json:"author_id"`
	Title        string                  `json:"title"`
	Content      string                  `json:"content"`
	Category     domain.QuestionCategory `json:"category"`
	AnswerFormat domain.AnswerFormat     `json:"answer_format"`
	Language     string                  `json:"language,omitempty"`
	StarterCode  string                  `json:"starter_code,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
}

func payloadFromQuestion(q *domain.Question) QuestionPayload {
	if q == nil {
		return QuestionPayload{}
	}
	return QuestionPayload{
		QuestionID:   q.ID,
		AuthorID:     q.AuthorID,
		Title:        q.Title,
		Content:      q.Content,
		Category:     q.Category,
		AnswerFormat: q.AnswerFormat,
		Language:     q.Language,
		StarterCode:  q.StarterCode,
		CreatedAt:    q.CreatedAt,
	}
}
