package main

import (
	"easyoffer/auth/api/handlers"
	"easyoffer/auth/internal/config"
	"easyoffer/auth/internal/repository"
	"easyoffer/auth/internal/service"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var authHTTPRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "easyoffer",
		Subsystem: "auth",
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests.",
	},
	[]string{"method", "route", "status"},
)

var authHTTPRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "easyoffer",
		Subsystem: "auth",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	},
	[]string{"method", "route", "status"},
)

func authMetricsMiddleware() gin.HandlerFunc {
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

		authHTTPRequestsTotal.WithLabelValues(method, route, status).Inc()
		authHTTPRequestDuration.WithLabelValues(method, route, status).Observe(duration)
	}
}

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	gin.SetMode(cfg.GinMode)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	prometheus.MustRegister(authHTTPRequestsTotal, authHTTPRequestDuration)
	r.Use(authMetricsMiddleware())

	if cfg.TrustedProxies == "" {
		if err := r.SetTrustedProxies(nil); err != nil {
			log.Fatal(err)
		}
	} else {
		proxies := strings.Split(cfg.TrustedProxies, ",")
		for i := range proxies {
			proxies[i] = strings.TrimSpace(proxies[i])
		}
		if err := r.SetTrustedProxies(proxies); err != nil {
			log.Fatal(err)
		}
	}

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	repo := repository.NewUserRepository(db)
	authService := service.NewAuthService(repo, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(authService)

	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
