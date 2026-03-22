package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/hermes/api"
	"github.com/sagarjhaa/localfinance/services/hermes/config"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Set up Gin router
	router := gin.New()

	// Add correlation middleware as the first middleware
	router.Use(middleware.CorrelationMiddleware("hermes"))

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Setup routes
	api.SetupRoutes(router, cfg)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🎭 Hermes service starting on port %s", port)
	log.Printf("📊 Correlation ID tracking enabled")
	log.Printf("🔗 Proxying to:")
	log.Printf("   🏛️ Thesaurus: %s", cfg.Services.Thesaurus)
	log.Printf("   🦉 Sophia: %s", cfg.Services.Sophia)
	log.Printf("   📜 Logos: %s", cfg.Services.Logos)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Hermes service: %v", err)
	}
}