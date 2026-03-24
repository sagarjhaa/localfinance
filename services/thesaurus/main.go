package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/thesaurus/api"
	"github.com/sagarjhaa/localfinance/services/thesaurus/config"
	"github.com/sagarjhaa/localfinance/services/thesaurus/database"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Set up Gin router
	router := gin.New()
	router.Use(middleware.CorrelationMiddleware("thesaurus"))
	router.Use(gin.Recovery())

	// Register all routes
	api.SetupRoutes(router, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	log.Printf("🏛️ Thesaurus service starting on port %s", port)
	log.Printf("📊 Database: %s@%s/%s", cfg.Database.User, cfg.Database.Host, cfg.Database.Name)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Thesaurus service: %v", err)
	}
}
