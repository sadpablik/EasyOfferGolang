package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	GinMode        string
	TrustedProxies string
	Port           string

	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	KafkaEnabled bool
	KafkaBrokers string
	KafkaTopic   string
}

func Load() Config {
	return Config{
		GinMode:        getEnv("GIN_MODE", "release"),
		TrustedProxies: getEnv("GIN_TRUSTED_PROXIES", ""),
		Port:           getEnv("QUESTION_SERVICE_PORT", "8082"),

		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "user"),
		DBPassword:   getEnv("DB_PASSWORD", "password"),
		DBName:       getEnv("DB_NAME", "easyoffer"),
		KafkaEnabled: getEnvBool("KAFKA_ENABLED", false),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "kafka:29092"),
		KafkaTopic:   getEnv("KAFKA_TOPIC_QUESTIONS", "questions.events"),
	}
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		c.DBHost,
		c.DBUser,
		c.DBPassword,
		c.DBName,
		c.DBPort,
	)
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
