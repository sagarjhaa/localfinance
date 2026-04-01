package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/gorm"
)

type ConversationHandler struct {
	db *gorm.DB
}

func NewConversationHandler(db *gorm.DB) *ConversationHandler {
	return &ConversationHandler{db: db}
}

// --- Internal routes (no auth, called by Sophia) ---

// CreateConversation creates a new conversation for a user.
// POST /internal/conversations
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Title  string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	conversation := models.Conversation{
		UserID: userID,
		Title:  req.Title,
	}
	if conversation.Title == "" {
		conversation.Title = "New Chat"
	}

	if err := h.db.Create(&conversation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	c.JSON(http.StatusCreated, conversation)
}

// UpdateConversation updates a conversation's title.
// PATCH /internal/conversations/:id
func (h *ConversationHandler) UpdateConversation(c *gin.Context) {
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var conversation models.Conversation
	if err := h.db.First(&conversation, "id = ?", convID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find conversation"})
		return
	}

	if err := h.db.Model(&conversation).Update("title", req.Title).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update conversation"})
		return
	}

	c.JSON(http.StatusOK, conversation)
}

// AddMessage adds a message to a conversation.
// POST /internal/conversations/:id/messages
func (h *ConversationHandler) AddMessage(c *gin.Context) {
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req struct {
		Role       string   `json:"role" binding:"required"`
		Content    string   `json:"content" binding:"required"`
		Confidence *float64 `json:"confidence,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify conversation exists
	var conversation models.Conversation
	if err := h.db.First(&conversation, "id = ?", convID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find conversation"})
		return
	}

	message := models.ChatMessage{
		ConversationID: convID,
		Role:           req.Role,
		Content:        req.Content,
		Confidence:     req.Confidence,
	}

	if err := h.db.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message"})
		return
	}

	// Touch conversation updated_at
	h.db.Model(&conversation).Update("updated_at", time.Now())

	c.JSON(http.StatusCreated, message)
}

// --- Protected routes (auth required, called by Iris) ---

// ListConversations returns the user's conversations ordered by updated_at DESC.
// GET /api/v1/conversations/
func (h *ConversationHandler) ListConversations(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	var conversations []models.Conversation
	if err := h.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&conversations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list conversations"})
		return
	}

	// Build response with last assistant message preview
	type ConversationResponse struct {
		models.Conversation
		Preview string `json:"preview"`
	}

	var results []ConversationResponse
	for _, conv := range conversations {
		var lastMsg models.ChatMessage
		preview := ""
		if err := h.db.Where("conversation_id = ? AND role = ?", conv.ID, "assistant").
			Order("created_at DESC").First(&lastMsg).Error; err == nil {
			preview = lastMsg.Content
			if len(preview) > 100 {
				preview = preview[:100]
			}
		}
		results = append(results, ConversationResponse{
			Conversation: conv,
			Preview:      preview,
		})
	}

	c.JSON(http.StatusOK, results)
}

// GetMessages returns all messages for a conversation ordered by created_at ASC.
// GET /api/v1/conversations/:id/messages
func (h *ConversationHandler) GetMessages(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	// Verify conversation belongs to user
	var conversation models.Conversation
	if err := h.db.Where("id = ? AND user_id = ?", convID, userID).First(&conversation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find conversation"})
		return
	}

	var messages []models.ChatMessage
	if err := h.db.Where("conversation_id = ?", convID).Order("created_at ASC").Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// DeleteConversation soft-deletes a conversation and hard-deletes its messages.
// DELETE /api/v1/conversations/:id
func (h *ConversationHandler) DeleteConversation(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	// Verify ownership
	var conversation models.Conversation
	if err := h.db.Where("id = ? AND user_id = ?", convID, userID).First(&conversation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find conversation"})
		return
	}

	// Hard-delete messages first
	if err := h.db.Unscoped().Where("conversation_id = ?", convID).Delete(&models.ChatMessage{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete messages"})
		return
	}

	// Soft-delete conversation
	if err := h.db.Delete(&conversation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation deleted successfully"})
}
