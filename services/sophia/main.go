package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/sophia/api"
	"github.com/sagarjhaa/localfinance/services/sophia/config"
	"github.com/sagarjhaa/localfinance/services/sophia/ai"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize AI service
	aiService, err := ai.NewService(cfg.AI)
	if err != nil {
		log.Fatal("Failed to initialize AI service:", err)
	}

	// Setup routes
	router := gin.Default()
	api.SetupRoutes(router, aiService, cfg.Thesaurus)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8002"
	}

	log.Printf("🦉 Sophia AI service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}