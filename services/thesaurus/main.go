package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/thesaurus/api"
	"github.com/sagarjhaa/localfinance/services/thesaurus/config"
	"github.com/sagarjhaa/localfinance/services/thesaurus/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize database
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Setup routes
	router := gin.Default()
	api.SetupRoutes(router, db)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	log.Printf("🏛️ Thesaurus service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}