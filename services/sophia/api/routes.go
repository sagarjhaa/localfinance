package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/ai"
	"github.com/sagarjhaa/localfinance/internal/api/handlers"
	"github.com/sagarjhaa/localfinance/services/sophia/config"
	"github.com/sagarjhaa/localfinance/internal/insights"
	"github.com/sagarjhaa/localfinance/internal/monthreview"
)

func SetupRoutes(router *gin.Engine, aiService *ai.Service, thesaurusConfig config.ThesaurusConfig, modelName string) {
	// Set Thesaurus URL for AI service
	aiService.SetThesaurusURL(thesaurusConfig.BaseURL)

	// Initialize handlers
	chatHandler := handlers.NewChatHandler(aiService)
	insightsHandler := handlers.NewInsightsHandler(aiService)
	categorizeHandler := handlers.NewCategorizeHandler(aiService)

	// Wire month-review service: shares the dismissal fetcher + narrator with
	// the insights handler so dismissal state stays consistent. Engine and
	// narrator are constructed fresh — they're stateless.
	thesaurusURL := os.Getenv("THESAURUS_URL")
	if thesaurusURL == "" {
		thesaurusURL = thesaurusConfig.BaseURL
	}
	mrDismiss := insights.NewThesaurusDismissalFetcher(thesaurusURL)
	mrDismiss.Client = &http.Client{Timeout: 5 * time.Second}
	var mrNarrator insights.Narrator = insights.NewTemplateNarrator()
	if os.Getenv("INSIGHTS_LLM_POLISH") == "1" {
		mrNarrator = insights.NewLLMNarrator(aiService)
	}
	mrService := monthreview.NewService(insights.NewEngine(mrDismiss), mrNarrator, aiService)
	monthReviewHandler := handlers.NewMonthReviewHandler(mrService)

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

		// Month-in-Review routes
		// Internal: called by Logos after a successful upload to pre-warm the cache.
		v1.POST("/internal/month-review/generate", monthReviewHandler.Generate)
		// Public: read by Iris for the JWT user (or via ?user_id= query for tests).
		v1.GET("/month-review/:period", monthReviewHandler.Get)
		v1.DELETE("/month-review/:period", monthReviewHandler.Delete)

		// Categorization routes - Transaction categorization
		categorize := v1.Group("/categorize")
		{
			categorize.POST("/", categorizeHandler.CategorizeTransaction)
			categorize.POST("/batch", categorizeHandler.CategorizeTransactionBatch)
		}

		// AI-powered transaction parsing
		v1.POST("/parse", func(c *gin.Context) {
			var req struct {
				Text   string `json:"text" binding:"required"`
				UserID string `json:"user_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			userModel := modelName
			if req.UserID != "" {
				userModel = aiService.GetUserModelPreference(req.UserID)
			}

			transactions, err := aiService.ParseTransactions(req.Text, userModel)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{
				"transactions": transactions,
				"count":        len(transactions),
				"model":        userModel,
			})
		})

		// AI-powered transaction parsing from PDF page images (vision model).
		// Body: { images: ["<base64 png>", ...], user_id: "<uuid>" }
		v1.POST("/parse-images", func(c *gin.Context) {
			var req struct {
				Images []string `json:"images" binding:"required"`
				UserID string   `json:"user_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			if len(req.Images) == 0 {
				c.JSON(400, gin.H{"error": "images array is empty"})
				return
			}

			userModel := modelName
			if req.UserID != "" {
				userModel = aiService.GetUserModelPreference(req.UserID)
			}

			transactions, err := aiService.ParseTransactionsFromImages(req.Images, userModel)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{
				"transactions": transactions,
				"count":        len(transactions),
				"model":        userModel,
			})
		})

		// Models — list installed Ollama models
		v1.GET("/models", func(c *gin.Context) {
			resp, err := http.Get(aiService.GetOllamaHost() + "/api/tags")
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to connect to Ollama"})
				return
			}
			defer resp.Body.Close()

			var result struct {
				Models []struct {
					Name       string `json:"name"`
					Size       int64  `json:"size"`
					ModifiedAt string `json:"modified_at"`
				} `json:"models"`
			}
			json.NewDecoder(resp.Body).Decode(&result)

			// Format sizes to human-readable
			type ModelInfo struct {
				Name string `json:"name"`
				Size string `json:"size"`
			}
			var models []ModelInfo
			for _, m := range result.Models {
				sizeGB := float64(m.Size) / 1e9
				sizeStr := fmt.Sprintf("%.1f GB", sizeGB)
				if sizeGB < 1 {
					sizeStr = fmt.Sprintf("%.0f MB", float64(m.Size)/1e6)
				}
				models = append(models, ModelInfo{Name: m.Name, Size: sizeStr})
			}
			c.JSON(200, models)
		})

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