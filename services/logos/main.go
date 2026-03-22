package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/logos/api"
	"github.com/sagarjhaa/localfinance/services/logos/config"
	"github.com/sagarjhaa/localfinance/services/logos/processors"
	"github.com/sagarjhaa/localfinance/services/logos/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize storage
	storageClient, err := storage.NewMinIOClient(cfg.Storage)
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	// Initialize document processors
	processorManager := processors.NewManager()

	// Setup routes
	router := gin.Default()
	api.SetupRoutes(router, processorManager, storageClient, cfg)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}

	log.Printf("📜 Logos document processing service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}