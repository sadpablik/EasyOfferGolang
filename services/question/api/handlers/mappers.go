package handlers

import (
	"time"

	"easyoffer/question/internal/domain"
)

func toQuestionResponse(question *domain.Question) QuestionResponse {
	return QuestionResponse{
		ID:           question.ID,
		Title:        question.Title,
		Content:      question.Content,
		Category:     string(question.Category),
		AnswerFormat: string(question.AnswerFormat),
		Language:     question.Language,
		StarterCode:  question.StarterCode,
		AuthorID:     question.AuthorID,
		CreatedAt:    question.CreatedAt.Format(time.RFC3339),
	}
}

func toQuestionResponses(questions []*domain.Question) []QuestionResponse {
	response := make([]QuestionResponse, 0, len(questions))
	for _, question := range questions {
		response = append(response, toQuestionResponse(question))
	}
	return response
}

func toQuestionReviewResponse(review *domain.QuestionReview) QuestionReviewResponse {
	return QuestionReviewResponse{
		ID:         review.ID,
		UserID:     review.UserID,
		QuestionID: review.QuestionID,
		Status:     string(review.Status),
		UserAnswer: review.UserAnswer,
		Note:       review.Note,
		ReviewedAt: review.ReviewedAt.Format(time.RFC3339),
	}
}

func toQuestionReviewResponses(reviews []*domain.QuestionReview) []QuestionReviewResponse {
	response := make([]QuestionReviewResponse, 0, len(reviews))
	for _, review := range reviews {
		response = append(response, toQuestionReviewResponse(review))
	}
	return response
}

func toMyQuestionResponse(question *domain.QuestionWithReview) MyQuestionResponse {
	reviewedAt := ""
	if !question.ReviewedAt.IsZero() {
		reviewedAt = question.ReviewedAt.Format(time.RFC3339)
	}
	return MyQuestionResponse{
		ID:           question.ID,
		Title:        question.Title,
		Content:      question.Content,
		Category:     string(question.Category),
		AnswerFormat: string(question.AnswerFormat),
		Language:     question.Language,
		StarterCode:  question.StarterCode,
		AuthorID:     question.AuthorID,
		CreatedAt:    question.CreatedAt.Format(time.RFC3339),
		ReviewStatus: string(question.ReviewStatus),
		ReviewedAt:   reviewedAt,
	}
}

func toMyQuestionResponses(questions []*domain.QuestionWithReview) []MyQuestionResponse {
	response := make([]MyQuestionResponse, 0, len(questions))
	for _, question := range questions {
		response = append(response, toMyQuestionResponse(question))
	}
	return response
}
