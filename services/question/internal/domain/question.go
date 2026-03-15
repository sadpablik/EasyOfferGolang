package domain

import "time"

type Question struct {
	ID           string           `gorm:"type:uuid;primaryKey" json:"id"`
	Title        string           `gorm:"type:varchar(255)" json:"title"`
	Content      string           `gorm:"type:text;not null" json:"content"`
	Category     QuestionCategory `gorm:"type:varchar(255);index:idx_questions_category" json:"category"`
	AnswerFormat AnswerFormat     `gorm:"type:varchar(255);index:idx_questions_answer_format" json:"answer_format"`
	Language     string           `gorm:"type:varchar(255);index:idx_questions_language" json:"language"`
	StarterCode  string           `gorm:"type:text" json:"starter_code"`
	AuthorID     string           `gorm:"type:varchar(255); not null;index" json:"author_id"`
	CreatedAt    time.Time        `gorm:"type:timestamp;index:idx_questions_created_at" json:"created_at"`
}

type QuestionReview struct {
	ID         string       `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     string       `gorm:"type:varchar(255);not null;uniqueIndex:idx_user_question;index:idx_user_status_question,priority:1" json:"user_id"`
	QuestionID string       `gorm:"type:uuid;not null;uniqueIndex:idx_user_question;index:idx_user_status_question,priority:3" json:"question_id"`
	Status     ReviewStatus `gorm:"type:varchar(255);index:idx_user_status_question,priority:2" json:"status"`
	UserAnswer string       `gorm:"type:text" json:"user_answer"`
	Note       string       `gorm:"type:text" json:"note"`
	ReviewedAt time.Time    `gorm:"type:timestamp" json:"reviewed_at"`
}

type QuestionWithReview struct {
	ID           string           `gorm:"type:uuid;primaryKey" json:"id"`
	Title        string           `gorm:"type:varchar(255)" json:"title"`
	Content      string           `gorm:"type:text;not null" json:"content"`
	Category     QuestionCategory `gorm:"type:varchar(255)" json:"category"`
	AnswerFormat AnswerFormat     `gorm:"type:varchar(255)" json:"answer_format"`
	Language     string           `gorm:"type:varchar(255)" json:"language"`
	StarterCode  string           `gorm:"type:text" json:"starter_code"`
	AuthorID     string           `gorm:"type:varchar(255); not null;index" json:"author_id"`
	CreatedAt    time.Time        `gorm:"type:timestamp" json:"created_at"`

	ReviewStatus ReviewStatus `gorm:"type:varchar(255)" json:"review_status,omitempty"`
	ReviewedAt   time.Time    `gorm:"type:timestamp" json:"reviewed_at,omitempty"`
}

type QuestionCategory string

const (
	CategoryResume   QuestionCategory = "resume"
	CategoryTheory   QuestionCategory = "theory"
	CategoryPractice QuestionCategory = "practice"
)

type AnswerFormat string

const (
	AnswerFormatText AnswerFormat = "text"
	AnswerFormatCode AnswerFormat = "code"
)

type ReviewStatus string

const (
	StatusKnow     ReviewStatus = "know"
	StatusDontKnow ReviewStatus = "dont_know"
	StatusRepeat   ReviewStatus = "repeat"
)
