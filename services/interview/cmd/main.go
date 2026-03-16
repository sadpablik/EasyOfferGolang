package main

import (
	"context"
	"easyoffer/interview/api/handlers"
	"easyoffer/interview/internal/config"
	"easyoffer/interview/internal/consumer"
	"easyoffer/interview/internal/repository"
	"easyoffer/interview/internal/service"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "easyoffer",
		Subsystem: "interview",
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests.",
	},
	[]string{"method", "route", "status"},
)

var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "easyoffer",
		Subsystem: "interview",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	},
	[]string{"method", "route", "status"},
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
	questionRepo := repository.NewRedisQuestionRepository(redisClient)
	eventStore := repository.NewRedisEventStore(redisClient)
	var questionConsumer *consumer.QuestionConsumer

	if cfg.KafkaEnabled {
		qc, err := consumer.NewQuestionConsumer(
			cfg.KafkaBrokers,
			cfg.KafkaTopic,
			cfg.KafkaConsumerGroup,
			questionRepo,
		)
		if err != nil {
			log.Fatal("failed to initialize question consumer: ", err)
		}
		questionConsumer = qc

		go func() {
			if err := questionConsumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("question consumer stopped: %v", err)
			}
		}()
	}

	interviewService := service.NewInterviewServiceWithEventStore(sessionRepo, questionRepo, eventStore, cfg.SessionTTL)
	interviewHandler := handlers.NewInterviewHandler(interviewService)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
	consumer.RegisterMetrics(prometheus.DefaultRegisterer)
	repository.RegisterMetrics(prometheus.DefaultRegisterer)
	r.Use(metricsMiddleware())
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

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		log.Printf("interview service starting on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	select {
	case err := <-serverErrCh:
		log.Printf("interview server failed: %v", err)
		stop()
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	}

	if questionConsumer != nil {
		if err := questionConsumer.Close(); err != nil {
			log.Printf("failed to close question consumer: %v", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("failed to shutdown interview server: %v", err)
	}
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}
		if route == "/metrics" {
			return
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(method, route, status).Inc()
		httpRequestDuration.WithLabelValues(method, route, status).Observe(duration)
	}
}
