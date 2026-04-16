package config

import (
	"fmt"
	"os"
)

type Config struct {
	GinMode        string
	TrustedProxies string
	Port           string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret string
}

func Load() Config {
	return Config{
		GinMode:        getEnv("GIN_MODE", "release"),
		TrustedProxies: getEnv("GIN_TRUSTED_PROXIES", ""),
		Port:           getEnv("AUTH_SERVICE_PORT", "8081"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "user"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "easyoffer"),

		JWTSecret: getEnv("JWT_SECRET", ""),
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
