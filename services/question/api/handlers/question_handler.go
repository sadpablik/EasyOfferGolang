package handlers

import (
	"easyoffer/question/internal/service"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type QuestionHandler struct {
	questionService service.QuestionService
}

func NewQuestionHandler(questionService service.QuestionService) *QuestionHandler {
	return &QuestionHandler{questionService: questionService}
}

func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
	authorID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if authorID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing X-User-ID header"})
		return
	}

	var req CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	q, err := h.questionService.CreateQuestion(req.Title, req.Content, authorID, req.Category, req.AnswerFormat, req.Language, req.StarterCode)
	if err != nil {
		switch err {
		case service.ErrQuestionAlreadyExists:
			c.JSON(http.StatusConflict, ErrorResponse{Error: "question already exists"})
		case service.ErrInvalidCategory:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid category, must be: resume, theory, practice"})
		case service.ErrInvalidAnswerFormat:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid answer_format, must be: text, code"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create question"})
		}
		return
	}

	c.JSON(http.StatusCreated, toQuestionResponse(q))
}

func (h *QuestionHandler) PatchQuestion(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing X-User-ID header"})
		return
	}

	questionID := strings.TrimSpace(c.Param("id"))
	if questionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "missing question id"})
		return
	}

	var req UpdateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	q, err := h.questionService.PatchQuestion(
		questionID,
		userID,
		req.Title,
		req.Content,
		req.Category,
		req.AnswerFormat,
		req.Language,
		req.StarterCode,
	)
	if err != nil {
		switch err {
		case service.ErrQuestionNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "question not found"})
		case service.ErrForbiddenQuestionUpdate:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "forbidden to update this question"})
		case service.ErrInvalidCategory:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid category, must be: resume, theory, practice"})
		case service.ErrInvalidAnswerFormat:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid answer_format, must be: text, code"})
		case service.ErrInvalidQuestionPayload:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload: title/content must not be empty"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update question"})
		}
		return
	}

	c.JSON(http.StatusOK, toQuestionResponse(q))
}

func (h *QuestionHandler) ReviewQuestion(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing X-User-ID header"})
		return
	}

	questionID := strings.TrimSpace(c.Param("id"))
	if questionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "missing question id"})
		return
	}

	var req ReviewQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	review, err := h.questionService.ReviewQuestion(userID, questionID, req.Status, req.UserAnswer, req.Note)
	if err != nil {
		switch err {
		case service.ErrInvalidReviewStatus:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid status, must be: know, dont_know, repeat"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save review"})
		}
		return
	}

	c.JSON(http.StatusOK, toQuestionReviewResponse(review))
}

func (h *QuestionHandler) ListQuestions(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	category := strings.TrimSpace(c.Query("category"))
	status := strings.TrimSpace(c.Query("status"))
	answerFormat := strings.TrimSpace(c.Query("answer_format"))
	language := strings.TrimSpace(c.Query("language"))
	searchQuery := strings.TrimSpace(c.Query("q"))

	unreviewed := false
	if raw := strings.TrimSpace(c.Query("unreviewed")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid unreviewed, must be true or false"})
			return
		}
		unreviewed = parsed
	}

	sortBy := strings.TrimSpace(c.Query("sort_by"))
	order := strings.TrimSpace(c.Query("order"))

	if (status != "" || unreviewed) && userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing X-User-ID header"})
		return
	}

	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid limit"})
			return
		}
		limit = n
	}

	offset := 0
	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid offset"})
			return
		}
		offset = n
	}

	questions, total, err := h.questionService.GetQuestions(userID, category, status, answerFormat, language, searchQuery, unreviewed, limit, offset, sortBy, order)
	if err != nil {
		switch err {
		case service.ErrMissingUserID:
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing X-User-ID header"})
		case service.ErrInvalidCategory:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid category, must be: resume, theory, practice"})
		case service.ErrInvalidAnswerFormat:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid answer_format, must be: text, code"})
		case service.ErrInvalidReviewStatus:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid status, must be: know, dont_know, repeat"})
		case service.ErrInvalidQuestionFilter:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid filters: status and unreviewed cannot be used together"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch questions"})
		}
		return
	}

	c.JSON(http.StatusOK, QuestionsListResponse{
		Questions: toQuestionResponses(questions),
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	})
}

func (h *QuestionHandler) GetQuestion(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "missing question id"})
		return
	}

	q, err := h.questionService.GetQuestionByID(id)
	if err != nil {
		switch err {
		case service.ErrQuestionNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "question not found"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch question"})
		}
		return
	}

	c.JSON(http.StatusOK, toQuestionResponse(q))
}

func (h *QuestionHandler) ListMyReviews(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing X-User-ID header"})
		return
	}

	status := strings.TrimSpace(c.Query("status"))

	reviews, err := h.questionService.GetMyReviews(userID, status)
	if err != nil {
		switch err {
		case service.ErrInvalidReviewStatus:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid status, must be: know, dont_know, repeat"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch reviews"})
		}
		return
	}

	response := toQuestionReviewResponses(reviews)

	c.JSON(http.StatusOK, ReviewsListResponse{
		Reviews: response,
		Total:   len(response),
	})
}

func (h *QuestionHandler) GetMyQuestionReview(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing X-User-ID header"})
		return
	}

	questionID := strings.TrimSpace(c.Param("id"))
	if questionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "missing question id"})
		return
	}

	review, err := h.questionService.GetMyQuestionReview(userID, questionID)
	if err != nil {
		if errors.Is(err, service.ErrReviewNotFound) {
			// 200 with empty review so the client does not get 404 for "not reviewed yet"
			c.JSON(http.StatusOK, QuestionReviewResponse{
				QuestionID: questionID,
				Status:     "",
				ReviewedAt: "",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch review"})
		return
	}

	c.JSON(http.StatusOK, toQuestionReviewResponse(review))
}

func (h *QuestionHandler) ListMyQuestions(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing X-User-ID header"})
		return
	}

	status := strings.TrimSpace(c.Query("status"))
	category := strings.TrimSpace(c.Query("category"))

	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid limit"})
			return
		}
		limit = n
	}

	offset := 0
	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid offset"})
			return
		}
		offset = n
	}

	questions, total, err := h.questionService.GetMyQuestions(userID, status, category, limit, offset)
	if err != nil {
		switch err {
		case service.ErrInvalidReviewStatus:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid status, must be: know, dont_know, repeat"})
		case service.ErrInvalidCategory:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid category, must be: resume, theory, practice"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch questions"})
		}
		return
	}

	c.JSON(http.StatusOK, MyQuestionsListResponse{
		Questions: toMyQuestionResponses(questions),
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	})
}

func (h *QuestionHandler) DeleteQuestion(c *gin.Context) {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing X-User-ID header"})
		return
	}

	questionID := strings.TrimSpace(c.Param("id"))
	if questionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "missing question id"})
		return
	}

	err := h.questionService.Delete(questionID, userID)
	if err != nil {
		switch err {
		case service.ErrQuestionNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "question not found"})
		case service.ErrForbiddenQuestionDelete:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "forbidden to delete this question"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete question"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
