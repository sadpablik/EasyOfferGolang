package handlers

import (
	"errors"
	"net/http"

	"easyoffer/interview/internal/service"

	"github.com/gin-gonic/gin"
)

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrMissingUserID):
		writeError(c, http.StatusUnauthorized, "missing X-User-ID header")
	case errors.Is(err, service.ErrInvalidCount):
		writeError(c, http.StatusBadRequest, "question count must be between 1 and 50")
	case errors.Is(err, service.ErrInvalidStatus):
		writeError(c, http.StatusBadRequest, "invalid review status")
	case errors.Is(err, service.ErrQuestionNotInSession):
		writeError(c, http.StatusBadRequest, "question does not belong to this session")
	case errors.Is(err, service.ErrSessionNotFound):
		writeError(c, http.StatusNotFound, "session not found")
	case errors.Is(err, service.ErrSessionForbidden):
		writeError(c, http.StatusForbidden, "access denied")
	case errors.Is(err, service.ErrSessionFinished):
		writeError(c, http.StatusConflict, "session is already finished")
	case errors.Is(err, service.ErrSessionNotFinished):
		writeError(c, http.StatusConflict, "session is not finished yet")
	case errors.Is(err, service.ErrNoQuestionsAvailable):
		writeError(c, http.StatusUnprocessableEntity, "no questions available for given filters")
	case errors.Is(err, service.ErrNotImplemented):
		writeError(c, http.StatusNotImplemented, "interview service endpoint is not implemented yet")
	default:
		writeError(c, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorResponse{Error: message})
}

func writeJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}
