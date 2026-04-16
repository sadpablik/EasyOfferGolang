package domain

import "time"

type ReviewStatus string

const (
	StatusKnow     ReviewStatus = "know"
	StatusDontKnow ReviewStatus = "dont_know"
	StatusRepeat   ReviewStatus = "repeat"
)

type QuestionSnapshot struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	Category     string `json:"category"`
	AnswerFormat string `json:"answer_format"`
	Language     string `json:"language,omitempty"`
	StarterCode  string `json:"starter_code,omitempty"`
	AuthorID     string `json:"author_id"`
	CreatedAt    string `json:"created_at"`
}

type SessionAnswer struct {
	QuestionID string       `json:"question_id"`
	Status     ReviewStatus `json:"status"`
	UserAnswer string       `json:"user_answer,omitempty"`
	Note       string       `json:"note,omitempty"`
	AnsweredAt time.Time    `json:"answered_at"`
}

type InterviewSession struct {
	ID           string                   `json:"id"`
	UserID       string                   `json:"user_id"`
	Questions    []QuestionSnapshot       `json:"questions"`
	CurrentIndex int                      `json:"current_index"`
	Answers      map[string]SessionAnswer `json:"answers"`
	StartedAt    time.Time                `json:"started_at"`
	FinishedAt   *time.Time               `json:"finished_at,omitempty"`
}

type InterviewResult struct {
	SessionID  string    `json:"session_id"`
	Total      int       `json:"total"`
	Answered   int       `json:"answered"`
	Know       int       `json:"know"`
	DontKnow   int       `json:"dont_know"`
	Repeat     int       `json:"repeat"`
	FinishedAt time.Time `json:"finished_at"`
}
