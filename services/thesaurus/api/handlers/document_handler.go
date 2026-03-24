package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/gorm"
)

type DocumentHandler struct {
	db *gorm.DB
}

func NewDocumentHandler(db *gorm.DB) *DocumentHandler {
	return &DocumentHandler{db: db}
}

// GetDocument returns a document by ID
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	id := c.Param("id")

	var doc models.Document
	if err := h.db.Where("id = ?", id).First(&doc).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// UpdateDocumentStatus updates a document's processing status (called by Logos)
func (h *DocumentHandler) UpdateDocumentStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"status": req.Status,
	}
	if req.ErrorMessage != "" {
		updates["error_message"] = req.ErrorMessage
	}

	result := h.db.Model(&models.Document{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated", "status": req.Status})
}

// GetTransactionsByDocument returns transactions for a specific document
func (h *DocumentHandler) GetTransactionsByDocument(c *gin.Context) {
	documentID := c.Query("document_id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document_id query parameter required"})
		return
	}

	var transactions []models.Transaction
	if err := h.db.Where("document_id = ?", documentID).Order("date ASC").Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
		"document_id":  documentID,
	})
}
