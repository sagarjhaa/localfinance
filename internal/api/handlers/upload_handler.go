package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/internal/data/models"
	"github.com/sagarjhaa/localfinance/internal/parse"
	"gorm.io/gorm"
)

type UploadHandler struct {
	db       *gorm.DB
	pipeline *parse.Pipeline
}

func NewUploadHandler(db *gorm.DB) *UploadHandler {
	return &UploadHandler{db: db}
}

// SetPipeline wires the parse pipeline. Optional: when nil, uploads will
// still be persisted but no async processing fires (useful in tests).
func (h *UploadHandler) SetPipeline(p *parse.Pipeline) {
	h.pipeline = p
}

// UploadDocument handles file upload, stores document record, fires to Logos
func (h *UploadHandler) UploadDocument(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Validate file type
	ext := filepath.Ext(file.Filename)
	allowed := map[string]bool{".csv": true, ".pdf": true, ".xlsx": true, ".xls": true, ".txt": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed"})
		return
	}
	accountID, err := h.ensureAccount(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user account"})
		return
	}

	// Save file to disk
	uploadDir := "/tmp/localfinance/uploads"
	os.MkdirAll(uploadDir, 0755)
	documentID := fmt.Sprintf("doc_%s", uuid.New().String()[:16])
	filePath := filepath.Join(uploadDir, fmt.Sprintf("%s_%s", documentID, file.Filename))

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Create document record
	doc := models.Document{
		ID:               documentID,
		UserID:           uid,
		Filename:         file.Filename,
		OriginalFilename: file.Filename,
		FileSize:         file.Size,
		Status:           "processing",
	}

	if err := h.db.Create(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create document record"})
		return
	}

	// Fire the in-process parse pipeline (async). Logos no longer exists as a
	// separate service — extract -> AI parse -> categorize -> persist all run
	// here on the unified binary.
	if h.pipeline != nil {
		go h.pipeline.ProcessDocument(context.Background(), parse.Request{
			DocumentID: documentID,
			FilePath:   filePath,
			FileType:   ext,
			UserID:     uid,
			AccountID:  accountID,
		})
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "File uploaded, processing started",
		"document_id": documentID,
		"filename":    file.Filename,
		"status":      "processing",
	})
}

func (h *UploadHandler) ensureAccount(userID uuid.UUID) (uuid.UUID, error) {
	var account models.Account
	err := h.db.Where("user_id = ?", userID).First(&account).Error
	if err == nil {
		return account.ID, nil
	}

	// Create default account
	account = models.Account{
		UserID:   userID,
		Name:     "Primary Account",
		Type:     "checking",
		Currency: "INR",
		IsActive: true,
	}
	if err := h.db.Create(&account).Error; err != nil {
		return uuid.Nil, err
	}
	return account.ID, nil
}

// GetProcessingStatus returns document processing status
func (h *UploadHandler) GetProcessingStatus(c *gin.Context) {
	documentID := c.Param("id")

	var doc models.Document
	if err := h.db.Where("id = ?", documentID).First(&doc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"document_id": doc.ID,
		"status":      doc.Status,
		"filename":    doc.Filename,
		"file_size":   doc.FileSize,
	})
}
