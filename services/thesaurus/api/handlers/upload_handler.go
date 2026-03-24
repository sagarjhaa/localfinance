package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/gorm"
)

type UploadHandler struct {
	db       *gorm.DB
	logosURL string
}

func NewUploadHandler(db *gorm.DB) *UploadHandler {
	logosURL := os.Getenv("LOGOS_URL")
	if logosURL == "" {
		logosURL = "http://localhost:8003"
	}
	return &UploadHandler{db: db, logosURL: logosURL}
}

// UploadDocument handles file upload, stores document record, fires to Logos
func (h *UploadHandler) UploadDocument(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
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

	// Ensure user has a default account
	uid := userID.(uuid.UUID)
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

	// Fire to Logos for processing (async)
	go h.sendToLogos(documentID, filePath, ext, uid, accountID)

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

func (h *UploadHandler) sendToLogos(documentID, filePath, fileType string, userID, accountID uuid.UUID) {
	payload, _ := json.Marshal(map[string]interface{}{
		"document_id": documentID,
		"file_path":   filePath,
		"file_type":   fileType,
		"user_id":     userID,
		"account_id":  accountID,
	})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(h.logosURL+"/api/v1/process", "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("[%s] Failed to send to Logos: %v", documentID, err)
		h.db.Model(&models.Document{}).Where("id = ?", documentID).Updates(map[string]interface{}{
			"status":        "error",
			"error_message": fmt.Sprintf("Logos service unavailable: %v", err),
		})
		return
	}
	defer resp.Body.Close()
	log.Printf("[%s] Sent to Logos, status: %d", documentID, resp.StatusCode)
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
