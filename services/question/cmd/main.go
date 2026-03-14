package main

import (
    "easyoffer/question/api/handlers"
    "easyoffer/question/internal/config"
    "easyoffer/question/internal/domain"
    "easyoffer/question/internal/service"
    "easyoffer/question/internal/repository"
    "log"
    "strings"

    "github.com/gin-gonic/gin"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    cfg := config.Load()

    gin.SetMode(cfg.GinMode)

    db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{TranslateError: true})
    if err != nil {
        log.Fatal("failed to connect to database:", err)
    }

    if err := db.AutoMigrate(&domain.Question{}); err != nil {
        log.Fatal("failed to migrate database:", err)
    }

    questionRepo := repository.NewQuestionRepository(db)
    questionService := service.NewQuestionService(questionRepo)
    questionHandler := handlers.NewQuestionHandler(questionService)

    g := gin.New()
    g.Use(gin.Logger(), gin.Recovery())

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

	log.Printf("question service starting on :%s", cfg.Port)
	if err := g.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}

}
