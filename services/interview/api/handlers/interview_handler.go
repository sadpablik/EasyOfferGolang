package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"easyoffer/interview/internal/service"

	"github.com/gin-gonic/gin"
)

type InterviewHandler struct {
	service service.InterviewService
}

func NewInterviewHandler(service service.InterviewService) *InterviewHandler {
	return &InterviewHandler{service: service}
}

func (h *InterviewHandler) StartInterview(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	input := StartInterviewRequest{}
	if err := json.NewDecoder(c.Request.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	session, firstQuestion, err := h.service.StartSession(c.Request.Context(), userID, service.StartSessionInput{
		Category:     input.Category,
		AnswerFormat: input.AnswerFormat,
		Language:     input.Language,
		Count:        input.Count,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, toStartInterviewResponse(session, firstQuestion))
}

func (h *InterviewHandler) NextQuestion(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	question, hasMore, err := h.service.GetNextQuestion(c.Request.Context(), userID, sessionID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, toNextQuestionResponse(question, hasMore))
}

func (h *InterviewHandler) SubmitAnswer(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	input := SubmitAnswerRequest{}
	if err := json.NewDecoder(c.Request.Body).Decode(&input); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.service.SubmitAnswer(c.Request.Context(), userID, sessionID, service.SubmitAnswerInput{
		QuestionID: input.QuestionID,
		Status:     input.Status,
		UserAnswer: input.UserAnswer,
		Note:       input.Note,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *InterviewHandler) FinishInterview(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	result, err := h.service.FinishSession(c.Request.Context(), userID, sessionID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, toResultResponse(result))
}

func (h *InterviewHandler) GetResult(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	result, err := h.service.GetResult(c.Request.Context(), userID, sessionID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, toResultResponse(result))
}
