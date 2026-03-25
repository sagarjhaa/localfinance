package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/sophia/ai"
	"github.com/sagarjhaa/localfinance/services/sophia/api"
	"github.com/sagarjhaa/localfinance/services/sophia/config"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize AI service
	aiService, err := ai.NewService(cfg.AI)
	if err != nil {
		log.Fatalf("Failed to initialize AI service: %v", err)
	}

	router := gin.New()
	router.Use(middleware.CorrelationMiddleware("sophia"))
	router.Use(gin.Recovery())

	// Setup full routes
	api.SetupRoutes(router, aiService, cfg.Thesaurus)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8002"
	}

	log.Printf("Sophia service starting on port %s", port)
	log.Printf("Correlation ID tracking enabled")
	log.Printf("AI model: %s", cfg.AI.ModelName)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Sophia service: %v", err)
	}
}
