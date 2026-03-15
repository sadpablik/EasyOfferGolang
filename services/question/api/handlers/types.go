package handlers

type CreateQuestionRequest struct {
	Title        string `json:"title" binding:"required"`
	Content      string `json:"content" binding:"required"`
	Category     string `json:"category" binding:"required"`
	AnswerFormat string `json:"answer_format" binding:"required"`
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

type ReviewQuestionRequest struct {
	Status     string `json:"status" binding:"required"`
	UserAnswer string `json:"user_answer"`
	Note       string `json:"note"`
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

type ErrorResponse struct {
	Error string `json:"error"`
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
	Total     int64                  `json:"total"`
	Limit int 		`json:"limit"`
	Offset int 		`json:"offset"`
}



type ReviewsListResponse struct {
	Reviews []QuestionReviewResponse `json:"reviews"`
	Total   int                      `json:"total"`
}

type QuestionsListResponse struct {
	Questions []QuestionResponse `json:"questions"`
	Total     int64                `json:"total"`
	Limit int 		`json:"limit"`
	Offset int 		`json:"offset"`
}