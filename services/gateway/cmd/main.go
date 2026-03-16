package main

import (
	"bytes"
	_ "easyoffer/gateway/docs"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func main() {
	authURL := strings.TrimRight(strings.TrimSpace(os.Getenv("AUTH_SERVICE_URL")), "/")

	if authURL == "" {
		log.Fatal("AUTH_SERVICE_URL is required")
	}

	questionURL := strings.TrimRight(strings.TrimSpace(os.Getenv("QUESTION_SERVICE_URL")), "/")
	if questionURL == "" {
		log.Fatal("QUESTION_SERVICE_URL is required")
	}

	interviewURL := strings.TrimRight(strings.TrimSpace(os.Getenv("INTERVIEW_SERVICE_URL")), "/")
	if interviewURL == "" {
		log.Fatal("INTERVIEW_SERVICE_URL is required")
	}

	port := strings.TrimSpace(os.Getenv("GATEWAY_PORT"))
	if port == "" {
		port = "8080"
	}

	g := &gateway{
		client:       &http.Client{Timeout: 5 * time.Second},
		authURL:      authURL,
		questionURL:  questionURL,
		interviewURL: interviewURL,
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.POST("/api/v1/auth/register", g.registerHandler)
	r.POST("/api/v1/auth/login", g.loginHandler)

	protected := r.Group("/api/v1")
	protected.Use(JWTAuthMiddleware(jwtSecret))
	protected.POST("/questions", g.createQuestionHandler)
	protected.PATCH("/questions/:id", g.patchQuestionHandler)
	protected.POST("/questions/:id/reviews", g.reviewQuestionHandler)
	protected.GET("/questions/:id/review", g.getMyQuestionReviewHandler)
	protected.GET("/questions", g.listQuestionsHandler)
	protected.GET("/questions/:id", g.getQuestionHandler)
	protected.GET("/me/reviews", g.listMyReviewsHandler)
	protected.GET("/me/questions", g.listMyQuestionsHandler)
	protected.DELETE("/questions/:id", g.deleteQuestionHandler)
	protected.POST("/interviews/start", g.startInterviewHandler)
	protected.GET("/interviews/:id/next", g.nextInterviewQuestionHandler)
	protected.POST("/interviews/:id/answer", g.answerInterviewQuestionHandler)
	protected.POST("/interviews/:id/finish", g.finishInterviewHandler)
	protected.GET("/interviews/:id/result", g.interviewResultHandler)
	r.GET("/health", healthHandler)
	r.GET("/swagger/*any", gin.WrapH(httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json"))))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("gateway starting on :%s", port)
	log.Fatal(server.ListenAndServe())
}

// registerHandler proxies registration requests to Auth Service.
// @Summary Register user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "User registration"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/register [post]
func (g *gateway) registerHandler(c *gin.Context) {
	g.proxyPost(c, g.authURL+"/register")
}

// @Summary Get my questions
// @Tags questions
// @Produce json
// @Param status query string false "Filter by status: know, dont_know, repeat"
// @Param category query string false "Filter by category: resume, theory, practice"
// @Param limit query int false "Page size (default 20, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {object} MyQuestionsListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/me/questions [get]
func (g *gateway) listMyQuestionsHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	target := g.questionURL + "/me/questions"

	query := url.Values{}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query.Set("status", status)
	}
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query.Set("category", category)
	}
	if limit := strings.TrimSpace(c.Query("limit")); limit != "" {
		query.Set("limit", limit)
	}
	if offset := strings.TrimSpace(c.Query("offset")); offset != "" {
		query.Set("offset", offset)
	}

	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}

	g.proxyGet(c, target, userID)
}

// @Summary Get my review for question
// @Tags questions
// @Produce json
// @Param id path string true "Question ID"
// @Success 200 {object} QuestionReviewResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/questions/{id}/review [get]
func (g *gateway) getMyQuestionReviewHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	target := g.questionURL + "/questions/" + c.Param("id") + "/review"
	g.proxyGet(c, target, userID)
}

// loginHandler proxies login requests to Auth Service.
// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "User login"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/login [post]
func (g *gateway) loginHandler(c *gin.Context) {
	g.proxyPost(c, g.authURL+"/login")
}

// createQuestionHandler proxies question creation requests to Question Service.
// @Summary Create question
// @Tags questions
// @Accept json
// @Produce json
// @Param request body CreateQuestionRequest true "Create question"
// @Success 201 {object} QuestionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/questions [post]
func (g *gateway) createQuestionHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}
	g.proxyPost(c, g.questionURL+"/questions", userID)
}

// patchQuestionHandler proxies question update requests to Question Service.
// @Summary Update question
// @Tags questions
// @Accept json
// @Produce json
// @Param id path string true "Question ID"
// @Param request body UpdateQuestionRequest true "Update question"
// @Success 200 {object} QuestionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/questions/{id} [patch]
func (g *gateway) patchQuestionHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	target := g.questionURL + "/questions/" + c.Param("id")
	g.proxyPatch(c, target, userID)
}

// reviewQuestionHandler proxies review requests to Question Service.
// @Summary Review a question (know / dont_know / repeat)
// @Tags questions
// @Accept json
// @Produce json
// @Param id path string true "Question ID"
// @Param request body ReviewQuestionRequest true "Review"
// @Success 200 {object} QuestionReviewResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/questions/{id}/reviews [post]
func (g *gateway) reviewQuestionHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}
	questionID := c.Param("id")
	g.proxyPost(c, g.questionURL+"/questions/"+questionID+"/reviews", userID)
}

// listQuestionsHandler proxies question list requests to Question Service.
// @Summary List questions
// @Tags questions
// @Produce json
// @Param category query string false "Filter by category: resume, theory, practice"
// @Param status query string false "Filter by my review status: know, dont_know, repeat"
// @Param answer_format query string false "Filter by answer format: text, code"
// @Param language query string false "Filter by language (case-insensitive exact match)"
// @Param q query string false "Search in title/content (case-insensitive)"
// @Param unreviewed query bool false "Only questions without my review"
// @Param limit query int false "Page size (default 20, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Param sort_by query string false "Sort field: created_at, title"
// @Param order query string false "Sort order: asc, desc"
// @Success 200 {object} QuestionsListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/questions [get]
func (g *gateway) listQuestionsHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	target := g.questionURL + "/questions"

	query := url.Values{}
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query.Set("category", category)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query.Set("status", status)
	}
	if answerFormat := strings.TrimSpace(c.Query("answer_format")); answerFormat != "" {
		query.Set("answer_format", answerFormat)
	}
	if language := strings.TrimSpace(c.Query("language")); language != "" {
		query.Set("language", language)
	}
	if searchQuery := strings.TrimSpace(c.Query("q")); searchQuery != "" {
		query.Set("q", searchQuery)
	}
	if unreviewed := strings.TrimSpace(c.Query("unreviewed")); unreviewed != "" {
		query.Set("unreviewed", unreviewed)
	}
	if limit := strings.TrimSpace(c.Query("limit")); limit != "" {
		query.Set("limit", limit)
	}
	if offset := strings.TrimSpace(c.Query("offset")); offset != "" {
		query.Set("offset", offset)
	}
	if sortBy := strings.TrimSpace(c.Query("sort_by")); sortBy != "" {
		query.Set("sort_by", sortBy)
	}
	if order := strings.TrimSpace(c.Query("order")); order != "" {
		query.Set("order", order)
	}

	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}

	g.proxyGet(c, target, userID)
}

// getQuestionHandler proxies single question fetch to Question Service.
// @Summary Get question by ID
// @Tags questions
// @Produce json
// @Param id path string true "Question ID"
// @Success 200 {object} QuestionResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/questions/{id} [get]
func (g *gateway) getQuestionHandler(c *gin.Context) {
	g.proxyGet(c, g.questionURL+"/questions/"+c.Param("id"))
}

// @Summary Get my question reviews
// @Tags questions
// @Produce json
// @Param status query string false "Filter by status: know, dont_know, repeat"
// @Success 200 {object} ReviewsListResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/me/reviews [get]
func (g *gateway) listMyReviewsHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	target := g.questionURL + "/me/reviews"

	query := url.Values{}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query.Set("status", status)
	}

	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}

	g.proxyGet(c, target, userID)
}

// @Summary Delete question
// @Tags questions
// @Produce json
// @Param id path string true "Question ID"
// @Success 204 {string} string "No Content"
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/questions/{id} [delete]
func (g *gateway) deleteQuestionHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	target := g.questionURL + "/questions/" + c.Param("id")
	g.proxyDelete(c, target, userID)
}

// @Summary Start interview session
// @Tags interviews
// @Accept json
// @Produce json
// @Param request body StartInterviewRequest false "Interview filters"
// @Success 201 {object} StartInterviewResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/interviews/start [post]
func (g *gateway) startInterviewHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	g.proxyPost(c, g.interviewURL+"/interviews/start", userID)
}

// @Summary Get next interview question
// @Tags interviews
// @Produce json
// @Param id path string true "Interview session ID"
// @Success 200 {object} NextInterviewQuestionResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/interviews/{id}/next [get]
func (g *gateway) nextInterviewQuestionHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	target := g.interviewURL + "/interviews/" + c.Param("id") + "/next"
	g.proxyGet(c, target, userID)
}

// @Summary Submit answer for interview question
// @Tags interviews
// @Accept json
// @Produce json
// @Param id path string true "Interview session ID"
// @Param request body SubmitInterviewAnswerRequest true "Interview answer"
// @Success 204 {string} string "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/interviews/{id}/answer [post]
func (g *gateway) answerInterviewQuestionHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	target := g.interviewURL + "/interviews/" + c.Param("id") + "/answer"
	g.proxyPost(c, target, userID)
}

// @Summary Finish interview session
// @Tags interviews
// @Produce json
// @Param id path string true "Interview session ID"
// @Success 200 {object} InterviewResultResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/interviews/{id}/finish [post]
func (g *gateway) finishInterviewHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	target := g.interviewURL + "/interviews/" + c.Param("id") + "/finish"
	g.proxyPost(c, target, userID)
}

// @Summary Get interview result
// @Tags interviews
// @Produce json
// @Param id path string true "Interview session ID"
// @Success 200 {object} InterviewResultResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/interviews/{id}/result [get]
func (g *gateway) interviewResultHandler(c *gin.Context) {
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user ID"})
		return
	}

	target := g.interviewURL + "/interviews/" + c.Param("id") + "/result"
	g.proxyGet(c, target, userID)
}

// healthHandler returns service liveness status.
// @Summary Gateway health check
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

func (g *gateway) proxyPost(c *gin.Context, target string, userID ...string) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	defer c.Request.Body.Close()

	upstreamReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to build upstream request"})
		return
	}

	if ct := strings.TrimSpace(c.GetHeader("Content-Type")); ct != "" {
		upstreamReq.Header.Set("Content-Type", ct)
	} else {
		upstreamReq.Header.Set("Content-Type", "application/json")
	}
	if len(userID) > 0 && strings.TrimSpace(userID[0]) != "" {
		upstreamReq.Header.Set("X-User-ID", strings.TrimSpace(userID[0]))
	}
	resp, err := g.client.Do(upstreamReq)
	if err != nil {
		if errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadGateway, ErrorResponse{Error: "upstream closed connection"})
			return
		}
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "upstream service unavailable"})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "failed to read upstream response"})
		return
	}

	if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
		c.Header("Content-Type", ct)
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

func (g *gateway) proxyPatch(c *gin.Context, target string, userID ...string) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	defer c.Request.Body.Close()

	upstreamReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPatch, target, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to build upstream request"})
		return
	}

	if ct := strings.TrimSpace(c.GetHeader("Content-Type")); ct != "" {
		upstreamReq.Header.Set("Content-Type", ct)
	} else {
		upstreamReq.Header.Set("Content-Type", "application/json")
	}
	if len(userID) > 0 && strings.TrimSpace(userID[0]) != "" {
		upstreamReq.Header.Set("X-User-ID", strings.TrimSpace(userID[0]))
	}

	resp, err := g.client.Do(upstreamReq)
	if err != nil {
		if errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadGateway, ErrorResponse{Error: "upstream closed connection"})
			return
		}
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "upstream service unavailable"})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "failed to read upstream response"})
		return
	}

	if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
		c.Header("Content-Type", ct)
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

func (g *gateway) proxyGet(c *gin.Context, target string, userID ...string) {
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, target, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to build upstream request"})
		return
	}
	if len(userID) > 0 && strings.TrimSpace(userID[0]) != "" {
		req.Header.Set("X-User-ID", strings.TrimSpace(userID[0]))
	}

	resp, err := g.client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "upstream service unavailable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "failed to read upstream response"})
		return
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

func (g *gateway) proxyDelete(c *gin.Context, target string, userID ...string) {
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodDelete, target, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to build upstream request"})
		return
	}
	if len(userID) > 0 && strings.TrimSpace(userID[0]) != "" {
		req.Header.Set("X-User-ID", strings.TrimSpace(userID[0]))
	}

	resp, err := g.client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "upstream service unavailable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "failed to read upstream response"})
		return
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}
