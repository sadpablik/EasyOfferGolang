package client

type ListQuestionsParams struct {
	UserID       string
	Category     string
	AnswerFormat string
	Language     string
	Limit        int
}