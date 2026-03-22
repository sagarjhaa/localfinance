package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/hermes/api"
	"github.com/sagarjhaa/localfinance/services/hermes/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Setup API Gateway and routes
	router := gin.Default()
	
	// Configure CORS for frontend
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Setup routes
	api.SetupRoutes(router, cfg)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🎭 Hermes frontend service starting on port %s", port)
	log.Printf("Frontend will be accessible at http://localhost:%s", port)
	log.Printf("API Gateway routes configured for backend services")
	
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}