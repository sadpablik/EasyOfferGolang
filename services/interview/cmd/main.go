package main

import (
	"context"
	"easyoffer/interview/api/handlers"
	"easyoffer/interview/internal/client"
	"easyoffer/interview/internal/config"
	"easyoffer/interview/internal/repository"
	"easyoffer/interview/internal/service"
	"log"
	"net/http"
	"time"

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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /interviews/start", interviewHandler.StartInterview)
	mux.HandleFunc("GET /interviews/{id}/next", interviewHandler.NextQuestion)
	mux.HandleFunc("POST /interviews/{id}/answer", interviewHandler.SubmitAnswer)
	mux.HandleFunc("POST /interviews/{id}/finish", interviewHandler.FinishInterview)
	mux.HandleFunc("GET /interviews/{id}/result", interviewHandler.GetResult)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("interview service starting on :%s", cfg.Port)
	log.Fatal(server.ListenAndServe())
}
