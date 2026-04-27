// Package main keeps the legacy Logos binary buildable until Phase 7 cleanup
// removes services/logos/ entirely. The real document parse pipeline now lives
// in internal/parse and runs in-process on the unified binary.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

func main() {
	router := gin.New()
	router.Use(middleware.CorrelationMiddleware("logos"))
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "deprecated",
			"service": "logos",
			"note":    "parse pipeline moved to internal/parse on the unified binary",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}
	log.Printf("logos legacy stub starting on port %s (pipeline now in unified binary)", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("logos stub: %v", err)
	}
}
