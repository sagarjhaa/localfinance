package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/logos/config"
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
	router.Use(middleware.CorrelationMiddleware("logos"))

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Setup routes (for now, use simplified setup)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"service": "logos",
			"version": "1.0.0",
		})
	})
	
	// TODO: Initialize processors and storage when ready
	// api.SetupRoutes(router, processorManager, storageClient, cfg)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}

	log.Printf("📜 Logos service starting on port %s", port)
	log.Printf("📊 Correlation ID tracking enabled")
	log.Printf("📁 Processing directory: %s", cfg.Processing.TempDir)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Logos service: %v", err)
	}
}