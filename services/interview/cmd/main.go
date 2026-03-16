package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"easyoffer/interview/api/handlers"
	"easyoffer/interview/internal/client"
	"easyoffer/interview/internal/config"
	"easyoffer/interview/internal/repository"
	"easyoffer/interview/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		log.Fatal("failed to connect to redis: ", err)
	}

	sessionRepo := repository.NewRedisSessionRepository(redisClient, cfg.SessionTTL)
	questionClient := client.NewHTTPQuestionClient(cfg.QuestionServiceURL, 5*time.Second)
	interviewService := service.NewInterviewService(sessionRepo, questionClient, cfg.SessionTTL)
	interviewHandler := handlers.NewInterviewHandler(interviewService)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/interviews/start", interviewHandler.StartInterview)
	r.GET("/interviews/:id/next", interviewHandler.NextQuestion)
	r.POST("/interviews/:id/answer", interviewHandler.SubmitAnswer)
	r.POST("/interviews/:id/finish", interviewHandler.FinishInterview)
	r.GET("/interviews/:id/result", interviewHandler.GetResult)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("interview service starting on :%s", cfg.Port)
	log.Fatal(server.ListenAndServe())
}
