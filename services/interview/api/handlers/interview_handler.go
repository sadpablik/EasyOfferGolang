package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"easyoffer/interview/internal/service"
)

type InterviewHandler struct {
	service service.InterviewService
}

func NewInterviewHandler(service service.InterviewService) *InterviewHandler {
	return &InterviewHandler{service: service}
}

func (h *InterviewHandler) StartInterview(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	input := StartInterviewRequest{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, firstQuestion, err := h.service.StartSession(r.Context(), userID, service.StartSessionInput{
		Category:     input.Category,
		AnswerFormat: input.AnswerFormat,
		Language:     input.Language,
		Count:        input.Count,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toStartInterviewResponse(session, firstQuestion))
}

func (h *InterviewHandler) NextQuestion(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	sessionID := strings.TrimSpace(r.PathValue("id"))
	question, hasMore, err := h.service.GetNextQuestion(r.Context(), userID, sessionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toNextQuestionResponse(question, hasMore))
}

func (h *InterviewHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	sessionID := strings.TrimSpace(r.PathValue("id"))
	input := SubmitAnswerRequest{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.service.SubmitAnswer(r.Context(), userID, sessionID, service.SubmitAnswerInput{
		QuestionID: input.QuestionID,
		Status:     input.Status,
		UserAnswer: input.UserAnswer,
		Note:       input.Note,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *InterviewHandler) FinishInterview(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	sessionID := strings.TrimSpace(r.PathValue("id"))
	result, err := h.service.FinishSession(r.Context(), userID, sessionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toResultResponse(result))
}

func (h *InterviewHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	sessionID := strings.TrimSpace(r.PathValue("id"))
	result, err := h.service.GetResult(r.Context(), userID, sessionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toResultResponse(result))
}
