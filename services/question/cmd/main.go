package main

import (
	"easyoffer/question/api/handlers"
	"easyoffer/question/internal/config"
	"easyoffer/question/internal/repository"
	"easyoffer/question/internal/service"
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
	g.PATCH("/questions/:id", questionHandler.PatchQuestion)
	g.POST("/questions/:id/reviews", questionHandler.ReviewQuestion)
	g.GET("/questions/:id/review", questionHandler.GetMyQuestionReview)
	g.GET("/questions", questionHandler.ListQuestions)
	g.GET("/questions/:id", questionHandler.GetQuestion)
	g.GET("/me/reviews", questionHandler.ListMyReviews)
	g.GET("/me/questions", questionHandler.ListMyQuestions)
	g.DELETE("/questions/:id", questionHandler.DeleteQuestion)

	log.Printf("question service starting on :%s", cfg.Port)
	if err := g.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
