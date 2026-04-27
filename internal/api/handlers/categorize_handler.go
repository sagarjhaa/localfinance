package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/ai"
)

type CategorizeHandler struct {
	aiService *ai.Service
}

func NewCategorizeHandler(aiService *ai.Service) *CategorizeHandler {
	return &CategorizeHandler{aiService: aiService}
}

func (h *CategorizeHandler) CategorizeTransaction(c *gin.Context) {
	var request ai.CategorizationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Categorize the transaction using AI
	result, err := h.aiService.CategorizeTransaction(request.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to categorize transaction",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"original_description": request.Description,
		"amount": request.Amount,
		"merchant": request.Merchant,
		"categorization": result,
	})
}

func (h *CategorizeHandler) CategorizeTransactionBatch(c *gin.Context) {
	var requests []ai.CategorizationRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(requests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No transactions provided"})
		return
	}

	if len(requests) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 100 transactions per batch"})
		return
	}

	results := make([]gin.H, len(requests))
	
	for i, request := range requests {
		// Categorize each transaction
		result, err := h.aiService.CategorizeTransaction(request.Description)
		if err != nil {
			results[i] = gin.H{
				"original_description": request.Description,
				"error": err.Error(),
				"categorization": nil,
			}
		} else {
			results[i] = gin.H{
				"original_description": request.Description,
				"amount": request.Amount,
				"merchant": request.Merchant,
				"categorization": result,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_transactions": len(requests),
		"results": results,
	})
}