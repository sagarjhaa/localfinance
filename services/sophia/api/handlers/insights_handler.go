package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/sophia/ai"
	"github.com/sagarjhaa/localfinance/services/sophia/models"
)

type InsightsHandler struct {
	aiService *ai.Service
}

func NewInsightsHandler(aiService *ai.Service) *InsightsHandler {
	return &InsightsHandler{aiService: aiService}
}

func (h *InsightsHandler) GenerateInsights(c *gin.Context) {
	var request models.InsightsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate financial insights for the user
	insights, err := h.aiService.GenerateInsights(request.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate insights",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": request.UserID,
		"insights": insights,
		"period": request.Period,
		"generated_at": insights[0].CreatedAt,
	})
}

func (h *InsightsHandler) GetUserInsights(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Generate insights for the user
	insights, err := h.aiService.GenerateInsights(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user insights",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"insights": insights,
	})
}

func (h *InsightsHandler) AnalyzeSpendingPatterns(c *gin.Context) {
	var request models.InsightsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement comprehensive spending pattern analysis
	// For now, return basic analysis structure
	analysis := models.SpendingAnalysis{
		UserID:        request.UserID,
		Period:        "monthly",
		TotalSpending: 0.0,
		CategoryBreakdown: map[string]models.CategoryStats{
			"Food": {
				Amount:     250.0,
				Count:      15,
				Percentage: 25.0,
				Average:    16.67,
				Trend:      "stable",
			},
			"Transportation": {
				Amount:     180.0,
				Count:      8,
				Percentage: 18.0,
				Average:    22.50,
				Trend:      "increasing",
			},
		},
		Trends: []models.SpendingTrend{
			{
				Category:    "Food",
				Direction:   "stable",
				Magnitude:   2.5,
				Description: "Food spending has remained consistent over the past month",
			},
		},
		Recommendations: []string{
			"Consider setting a monthly food budget based on your average spending",
			"Transportation costs are trending upward - review your commute options",
		},
	}

	c.JSON(http.StatusOK, analysis)
}