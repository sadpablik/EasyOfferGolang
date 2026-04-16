package domain

import (
	"encoding/json"
	"time"
)

type InterviewEventType string

const (
	EventSessionStarted  InterviewEventType = "session.started"
	EventAnswerSubmitted InterviewEventType = "answer.submitted"
	EventSessionFinished InterviewEventType = "session.finished"
)

type InterviewEvent struct {
	ID         string             `json:"id"`
	SessionID  string             `json:"session_id"`
	UserID     string             `json:"user_id"`
	Type       InterviewEventType `json:"type"`
	OccurredAt time.Time          `json:"occurred_at"`
	Version    int                `json:"version"`
	Payload    json.RawMessage    `json:"payload"`
}

type SessionStartedPayload struct {
	Category       string             `json:"category,omitempty"`
	AnswerFormat   string             `json:"answer_format,omitempty"`
	Language       string             `json:"language,omitempty"`
	RequestedCount int                `json:"requested_count"`
	Questions      []QuestionSnapshot `json:"questions"`
	StartedAt      time.Time          `json:"started_at"`
}

type AnswerSubmittedPayload struct {
	QuestionID string       `json:"question_id"`
	Status     ReviewStatus `json:"status"`
	UserAnswer string       `json:"user_answer,omitempty"`
	Note       string       `json:"note,omitempty"`
	AnsweredAt time.Time    `json:"answered_at"`
}

type SessionFinishedPayload struct {
	FinishedAt time.Time `json:"finished_at"`
}
