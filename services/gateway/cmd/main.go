package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "easyoffer/gateway/docs"

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

	port := strings.TrimSpace(os.Getenv("GATEWAY_PORT"))
	if port == "" {
		port = "8080"
	}

	g := &gateway{
		client:      &http.Client{Timeout: 5 * time.Second},
		authURL:     authURL,
		questionURL: questionURL,
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
