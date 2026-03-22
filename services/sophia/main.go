package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/sophia/config"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set up Gin router
	router := gin.New()

	// Add correlation middleware as the first middleware
	router.Use(middleware.CorrelationMiddleware("sophia"))

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Setup basic health route
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"service": "sophia",
			"version": "1.0.0",
		})
	})
	
	// TODO: Setup full routes when AI service is ready
	// api.SetupRoutes(router, aiService, cfg.Thesaurus)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8002"
	}

	log.Printf("🦉 Sophia service starting on port %s", port)
	log.Printf("📊 Correlation ID tracking enabled")
	log.Printf("🤖 AI model: %s", cfg.AI.ModelName)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Sophia service: %v", err)
	}
}