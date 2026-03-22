package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/thesaurus/config"
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
	router.Use(middleware.CorrelationMiddleware("thesaurus"))

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Setup basic health route
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"service": "thesaurus",
			"version": "1.0.0",
		})
	})



	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	log.Printf("🏛️ Thesaurus service starting on port %s", port)
	log.Printf("📊 Correlation ID tracking enabled")
	log.Printf("🗄️ Connected to database: %s", cfg.Database.Host)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Thesaurus service: %v", err)
	}
}