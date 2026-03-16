package handlers

type StartInterviewRequest struct {
	Category     string `json:"category"`
	AnswerFormat string `json:"answer_format"`
	Language     string `json:"language"`
	Count        int    `json:"count"`
}

type SubmitAnswerRequest struct {
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

type NextQuestionResponse struct {
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

type ErrorResponse struct {
	Error string `json:"error"`
}
