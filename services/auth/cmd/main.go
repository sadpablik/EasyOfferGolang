package main

import (
	"easyoffer/auth/api/handlers"
	"easyoffer/auth/internal/config"
	"easyoffer/auth/internal/domain"
	"easyoffer/auth/internal/repository"
	"easyoffer/auth/internal/service"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Загружаем .env, если файл есть (для docker/env не критично)
	_ = godotenv.Load()

	cfg := config.Load()

	gin.SetMode(cfg.GinMode)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	if cfg.TrustedProxies == "" {
		// Локальный запуск без reverse proxy: не доверяем прокси
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

	// В prod лучше миграции через golang-migrate, тут оставляем для локальной разработки
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	repo := repository.NewUserRepository(db)
	authService := service.NewAuthService(repo, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(authService)

	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
