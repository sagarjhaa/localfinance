package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/sophia/api"
	"github.com/sagarjhaa/localfinance/services/sophia/config"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Set up Gin router
	router := gin.New()

	// Add correlation middleware as the first middleware
	router.Use(middleware.CorrelationMiddleware("sophia"))

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Setup routes
	api.SetupRoutes(router, cfg)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8002"
	}

	log.Printf("🦉 Sophia service starting on port %s", port)
	log.Printf("📊 Correlation ID tracking enabled")
	log.Printf("🤖 AI model endpoint: %s", cfg.AI.ModelEndpoint)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Sophia service: %v", err)
	}
}