package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	QuestionServiceURL string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	SessionTTL         time.Duration
}

func Load() Config {
	port := strings.TrimSpace(os.Getenv("INTERVIEW_SERVICE_PORT"))
	if port == "" {
		port = "8083"
	}

	questionServiceURL := strings.TrimRight(strings.TrimSpace(os.Getenv("QUESTION_SERVICE_URL")), "/")
	if questionServiceURL == "" {
		questionServiceURL = "http://question-service:8082"
	}

	redisAddr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	sessionTTLSeconds := parseIntEnv("INTERVIEW_SESSION_TTL_SECONDS", 7200)

	return Config{
		Port:               port,
		QuestionServiceURL: questionServiceURL,
		RedisAddr:          redisAddr,
		RedisPassword:      strings.TrimSpace(os.Getenv("REDIS_PASSWORD")),
		RedisDB:            parseIntEnv("REDIS_DB", 0),
		SessionTTL:         time.Duration(sessionTTLSeconds) * time.Second,
	}
}

func parseIntEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}
