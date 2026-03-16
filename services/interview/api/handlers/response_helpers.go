package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"easyoffer/interview/internal/service"
)

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrMissingUserID):
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
	case errors.Is(err, service.ErrInvalidCount):
		writeError(w, http.StatusBadRequest, "question count must be between 1 and 50")
	case errors.Is(err, service.ErrInvalidStatus):
		writeError(w, http.StatusBadRequest, "invalid review status")
	case errors.Is(err, service.ErrQuestionNotInSession):
		writeError(w, http.StatusBadRequest, "question does not belong to this session")
	case errors.Is(err, service.ErrSessionNotFound):
		writeError(w, http.StatusNotFound, "session not found")
	case errors.Is(err, service.ErrSessionForbidden):
		writeError(w, http.StatusForbidden, "access denied")
	case errors.Is(err, service.ErrSessionFinished):
		writeError(w, http.StatusConflict, "session is already finished")
	case errors.Is(err, service.ErrSessionNotFinished):
		writeError(w, http.StatusConflict, "session is not finished yet")
	case errors.Is(err, service.ErrNoQuestionsAvailable):
		writeError(w, http.StatusUnprocessableEntity, "no questions available for given filters")
	case errors.Is(err, service.ErrNotImplemented):
		writeError(w, http.StatusNotImplemented, "interview service endpoint is not implemented yet")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
