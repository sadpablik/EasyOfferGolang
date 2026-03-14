package handlers

import (
	"easyoffer/question/internal/service"
	"net/http"
	"time"
	"strings"
	"github.com/gin-gonic/gin"
)

type CreateQuestionRequest struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
}

type QuestionResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	AuthorID  string `json:"author_id"`
	CreatedAt string `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type QuestionHandler struct {
	questionService service.QuestionService
}

func NewQuestionHandler(questionService service.QuestionService) *QuestionHandler {
	return &QuestionHandler{questionService: questionService}
}

func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
	var req CreateQuestionRequest
	authorID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if authorID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing X-User-ID header"})
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	q,err := h.questionService.CreateQuestion(req.Title, req.Content, authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create question"})
		return
	}
	
	c.JSON(http.StatusCreated, QuestionResponse{
		ID:        q.ID,
		Title:     q.Title,
		Content:   q.Content,
		AuthorID:  q.AuthorID,
		CreatedAt: q.CreatedAt.Format(time.RFC3339),
	})
}
