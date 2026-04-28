package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/ai"
)

type ChatHandler struct {
	aiService *ai.Service
}

func NewChatHandler(aiService *ai.Service) *ChatHandler {
	return &ChatHandler{aiService: aiService}
}

func (h *ChatHandler) HandleFinancialQuery(c *gin.Context) {
	var query ai.FinancialQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.aiService.AnswerFinancialQuery(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process financial query",
			"details": err.Error(),
		})
		return
	}

	response.GeneratedAt = time.Now()
	c.JSON(http.StatusOK, response)
}
