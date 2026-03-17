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
	ProjectorEnabled   bool
	ProjectorInterval  time.Duration
	KafkaEnabled       bool
	KafkaBrokers       string
	KafkaTopic         string
	KafkaConsumerGroup string
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
	projectorIntervalMS := parseIntEnv("INTERVIEW_PROJECTOR_INTERVAL_MS", 1000)
	if projectorIntervalMS <= 0 {
		projectorIntervalMS = 1000
	}

	return Config{
		Port:               port,
		QuestionServiceURL: questionServiceURL,
		RedisAddr:          redisAddr,
		RedisPassword:      strings.TrimSpace(os.Getenv("REDIS_PASSWORD")),
		RedisDB:            parseIntEnv("REDIS_DB", 0),
		SessionTTL:         time.Duration(sessionTTLSeconds) * time.Second,
		ProjectorEnabled:   getEnvBool("INTERVIEW_PROJECTOR_ENABLED", true),
		ProjectorInterval:  time.Duration(projectorIntervalMS) * time.Millisecond,
		KafkaEnabled:       getEnvBool("KAFKA_ENABLED", false),
		KafkaBrokers:       getEnv("KAFKA_BROKERS", "kafka:29092"),
		KafkaTopic:         getEnv("KAFKA_TOPIC_QUESTIONS", "questions.events"),
		KafkaConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "interview-service"),
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

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
