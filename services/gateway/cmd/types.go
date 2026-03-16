package main

import "net/http"

type gateway struct {
	client       *http.Client
	authURL      string
	questionURL  string
	interviewURL string
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	Role      string `json:"role"`
}

type RegisterResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type CreateQuestionRequest struct {
	Title        string `json:"title"`
	Content      string `json:"content"`
	Category     string `json:"category"`
	AnswerFormat string `json:"answer_format"`
	Language     string `json:"language"`
	StarterCode  string `json:"starter_code"`
}

type UpdateQuestionRequest struct {
	Title        *string `json:"title,omitempty"`
	Content      *string `json:"content,omitempty"`
	Category     *string `json:"category,omitempty"`
	AnswerFormat *string `json:"answer_format,omitempty"`
	Language     *string `json:"language,omitempty"`
	StarterCode  *string `json:"starter_code,omitempty"`
}

type QuestionResponse struct {
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

type ReviewQuestionRequest struct {
	Status     string `json:"status"`
	UserAnswer string `json:"user_answer"`
	Note       string `json:"note"`
}

type QuestionReviewResponse struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	QuestionID string `json:"question_id"`
	Status     string `json:"status"`
	UserAnswer string `json:"user_answer,omitempty"`
	Note       string `json:"note,omitempty"`
	ReviewedAt string `json:"reviewed_at"`
}

type MyQuestionResponse struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	Category     string `json:"category"`
	AnswerFormat string `json:"answer_format"`
	Language     string `json:"language,omitempty"`
	StarterCode  string `json:"starter_code,omitempty"`
	AuthorID     string `json:"author_id"`
	CreatedAt    string `json:"created_at"`
	ReviewStatus string `json:"review_status"`
	ReviewedAt   string `json:"reviewed_at"`
}

type MyQuestionsListResponse struct {
	Questions []MyQuestionResponse `json:"questions"`
	Total     int64                `json:"total"`
	Limit     int                  `json:"limit"`
	Offset    int                  `json:"offset"`
}

type QuestionsListResponse struct {
	Questions []QuestionResponse `json:"questions"`
	Total     int64              `json:"total"`
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
}

type ReviewsListResponse struct {
	Reviews []QuestionReviewResponse `json:"reviews"`
	Total   int                      `json:"total"`
}

type StartInterviewRequest struct {
	Category     string `json:"category"`
	AnswerFormat string `json:"answer_format"`
	Language     string `json:"language"`
	Count        int    `json:"count"`
}

type SubmitInterviewAnswerRequest struct {
	QuestionID string `json:"question_id"`
	Status     string `json:"status"`
	UserAnswer string `json:"user_answer"`
	Note       string `json:"note"`
}

type InterviewQuestionResponse struct {
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

type StartInterviewResponse struct {
	SessionID     string                     `json:"session_id"`
	Total         int                        `json:"total"`
	FirstQuestion *InterviewQuestionResponse `json:"first_question,omitempty"`
}

type NextInterviewQuestionResponse struct {
	Done     bool                       `json:"done"`
	Question *InterviewQuestionResponse `json:"question,omitempty"`
}

type InterviewResultResponse struct {
	SessionID  string `json:"session_id"`
	Total      int    `json:"total"`
	Answered   int    `json:"answered"`
	Know       int    `json:"know"`
	DontKnow   int    `json:"dont_know"`
	Repeat     int    `json:"repeat"`
	FinishedAt string `json:"finished_at"`
}
