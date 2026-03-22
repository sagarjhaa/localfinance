package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/sagarjhaa/localfinance/services/thesaurus/api"
	"github.com/sagarjhaa/localfinance/services/thesaurus/config"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate database schema
	err = db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.Transaction{},
		&models.Account{},
		&models.Budget{},
		&models.Category{},
		&models.Document{},
		&models.UserSetting{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Set up Gin router
	router := gin.New()

	// Add correlation middleware as the first middleware
	router.Use(middleware.CorrelationMiddleware("thesaurus"))

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Setup routes
	api.SetupRoutes(router, db)

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