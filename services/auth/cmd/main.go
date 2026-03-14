package main

import (
	"easyoffer/auth/api/handlers"
	"easyoffer/auth/internal/domain"
	"easyoffer/auth/internal/repository"
	"easyoffer/auth/internal/service"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"strings"
)

func main() {
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(mode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	proxiesRaw := os.Getenv("GIN_TRUSTED_PROXIES")
	if proxiesRaw == "" {
		// Локальный запуск без reverse proxy: безопасно не доверять прокси
		if err := r.SetTrustedProxies(nil); err != nil {
			log.Fatal(err)
		}
	} else {
		proxies := strings.Split(proxiesRaw, ",")
		for i := range proxies {
			proxies[i] = strings.TrimSpace(proxies[i])
		}
		if err := r.SetTrustedProxies(proxies); err != nil {
			log.Fatal(err)
		}
	}
	godotenv.Load()
	dsn := "host=" + os.Getenv("DB_HOST") + " user=" + os.Getenv("DB_USER") + " password=" + os.Getenv("DB_PASSWORD") + " dbname=" + os.Getenv("DB_NAME") + " port=" + os.Getenv("DB_PORT") + " sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	// golang-migrate in prodaction, for development we can use AutoMigrate
	db.AutoMigrate(&domain.User{})

	repo := repository.NewUserRepository(db)
	authService := service.NewAuthService(repo, os.Getenv("JWT_SECRET"))
	authHandler := handlers.NewAuthHandler(authService)

	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
	r.GET("/openapi/swagger.json", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/openapi/swagger.json")))
	r.Run(":8081")
}
