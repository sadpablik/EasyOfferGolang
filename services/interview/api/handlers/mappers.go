package handlers

import (
	"time"

	"easyoffer/interview/internal/domain"
)

func toQuestionResponse(question *domain.QuestionSnapshot) *InterviewQuestionResponse {
	if question == nil {
		return nil
	}

	return &InterviewQuestionResponse{
		ID:           question.ID,
		Title:        question.Title,
		Content:      question.Content,
		Category:     question.Category,
		AnswerFormat: question.AnswerFormat,
		Language:     question.Language,
		StarterCode:  question.StarterCode,
		AuthorID:     question.AuthorID,
		CreatedAt:    question.CreatedAt,
	}
}

func toStartInterviewResponse(session *domain.InterviewSession, firstQuestion *domain.QuestionSnapshot) StartInterviewResponse {
	return StartInterviewResponse{
		SessionID:     session.ID,
		Total:         len(session.Questions),
		FirstQuestion: toQuestionResponse(firstQuestion),
	}
}

func toNextQuestionResponse(question *domain.QuestionSnapshot, hasMore bool) NextQuestionResponse {
	return NextQuestionResponse{
		Done:     !hasMore,
		Question: toQuestionResponse(question),
	}
}

func toResultResponse(result *domain.InterviewResult) InterviewResultResponse {
	if result == nil {
		return InterviewResultResponse{}
	}

	return InterviewResultResponse{
		SessionID:  result.SessionID,
		Total:      result.Total,
		Answered:   result.Answered,
		Know:       result.Know,
		DontKnow:   result.DontKnow,
		Repeat:     result.Repeat,
		FinishedAt: result.FinishedAt.Format(time.RFC3339),
	}
}
