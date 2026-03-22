package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/sophia/ai"
	"github.com/sagarjhaa/localfinance/services/sophia/models"
)

type ChatHandler struct {
	aiService *ai.Service
}

func NewChatHandler(aiService *ai.Service) *ChatHandler {
	return &ChatHandler{aiService: aiService}
}

func (h *ChatHandler) HandleFinancialQuery(c *gin.Context) {
	var query models.FinancialQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate AI response
	response, err := h.aiService.AnswerFinancialQuery(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process financial query",
			"details": err.Error(),
		})
		return
	}

	// Set generation timestamp
	response.GeneratedAt = time.Now()

	c.JSON(http.StatusOK, response)
}

func (h *ChatHandler) GetChatHistory(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// For now, return empty history - in production, this would query a chat history store
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"messages": []models.ChatMessage{},
		"message": "Chat history feature coming soon",
	})
}

// saveChatMessage would save the chat to a database in production
func (h *ChatHandler) saveChatMessage(userID, question, answer string) error {
	// TODO: Implement chat history persistence
	// This would save to a chat_messages table or similar
	message := models.ChatMessage{
		ID:        uuid.New(),
		UserID:    userID,
		Message:   question,
		Response:  answer,
		Timestamp: time.Now(),
	}
	
	// In production, save to database
	_ = message
	
	return nil
}