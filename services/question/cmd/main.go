package main

import (
	"context"
	"easyoffer/question/api/handlers"
	"easyoffer/question/internal/config"
	"easyoffer/question/internal/events"
	"easyoffer/question/internal/repository"
	"easyoffer/question/internal/service"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var questionHTTPRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "easyoffer",
		Subsystem: "question",
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests.",
	},
	[]string{"method", "route", "status"},
)

var questionHTTPRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "easyoffer",
		Subsystem: "question",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	},
	[]string{"method", "route", "status"},
)

func questionMetricsMiddleware() gin.HandlerFunc {
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

		questionHTTPRequestsTotal.WithLabelValues(method, route, status).Inc()
		questionHTTPRequestDuration.WithLabelValues(method, route, status).Observe(duration)
	}
}

func main() {
	cfg := config.Load()

	gin.SetMode(cfg.GinMode)

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	questionRepo := repository.NewQuestionRepository(db)
	if cfg.KafkaEnabled {
		kafkaPublisher, err := events.NewKafkaPublisher(cfg.KafkaBrokers, cfg.KafkaTopic)
		if err != nil {
			log.Fatal("failed to initialize kafka publisher:", err)
		}

		outboxStore, ok := questionRepo.(events.OutboxStore)
		if !ok {
			log.Fatal("question repository does not implement outbox store")
		}

		dispatcher := events.NewOutboxDispatcher(outboxStore, kafkaPublisher, 50, 2*time.Second)
		go dispatcher.Run(context.Background())
	}

	questionService := service.NewQuestionService(questionRepo)
	questionHandler := handlers.NewQuestionHandler(questionService)

	g := gin.New()
	g.Use(gin.Logger(), gin.Recovery())
	prometheus.MustRegister(questionHTTPRequestsTotal, questionHTTPRequestDuration)
	g.Use(questionMetricsMiddleware())
	events.RegisterMetrics(prometheus.DefaultRegisterer)
	if cfg.TrustedProxies == "" {
		if err := g.SetTrustedProxies(nil); err != nil {
			log.Fatal(err)
		}
	} else {
		proxies := strings.Split(cfg.TrustedProxies, ",")
		for i := range proxies {
			proxies[i] = strings.TrimSpace(proxies[i])
		}
		if err := g.SetTrustedProxies(proxies); err != nil {
			log.Fatal(err)
		}
	}

	g.POST("/questions", questionHandler.CreateQuestion)
	g.PATCH("/questions/:id", questionHandler.PatchQuestion)
	g.POST("/questions/:id/reviews", questionHandler.ReviewQuestion)
	g.GET("/questions/:id/review", questionHandler.GetMyQuestionReview)
	g.GET("/questions", questionHandler.ListQuestions)
	g.GET("/questions/:id", questionHandler.GetQuestion)
	g.GET("/me/reviews", questionHandler.ListMyReviews)
	g.GET("/me/questions", questionHandler.ListMyQuestions)
	g.DELETE("/questions/:id", questionHandler.DeleteQuestion)
	g.GET("/metrics", gin.WrapH(promhttp.Handler()))

	log.Printf("question service starting on :%s", cfg.Port)
	if err := g.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
