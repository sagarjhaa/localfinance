package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/sophia/ai"
	"github.com/sagarjhaa/localfinance/services/sophia/api/handlers"
	"github.com/sagarjhaa/localfinance/services/sophia/config"
)

func SetupRoutes(router *gin.Engine, aiService *ai.Service, thesaurusConfig config.ThesaurusConfig, modelName string) {
	// Set Thesaurus URL for AI service
	aiService.SetThesaurusURL(thesaurusConfig.BaseURL)

	// Initialize handlers
	chatHandler := handlers.NewChatHandler(aiService)
	insightsHandler := handlers.NewInsightsHandler(aiService)
	categorizeHandler := handlers.NewCategorizeHandler(aiService)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "sophia",
			"ai_model": modelName,
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Chat routes - Natural language financial queries
		chat := v1.Group("/chat")
		{
			chat.POST("/", chatHandler.HandleFinancialQuery)
			chat.GET("/history/:userId", chatHandler.GetChatHistory)
		}

		// Insights routes - AI-generated financial insights
		insights := v1.Group("/insights")
		{
			insights.POST("/", insightsHandler.GenerateInsights)
			insights.GET("/:userId", insightsHandler.GetUserInsights)
			insights.POST("/analyze", insightsHandler.AnalyzeSpendingPatterns)
		}

		// Categorization routes - Transaction categorization
		categorize := v1.Group("/categorize")
		{
			categorize.POST("/", categorizeHandler.CategorizeTransaction)
			categorize.POST("/batch", categorizeHandler.CategorizeTransactionBatch)
		}

		// AI status and configuration
		status := v1.Group("/status")
		{
			status.GET("/", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"ai_service": "operational",
					"model":      modelName,
					"endpoints": []string{
						"/api/v1/chat",
						"/api/v1/insights",
						"/api/v1/categorize",
					},
				})
			})
		}
	}
}