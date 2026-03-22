package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"github.com/sagarjhaa/localfinance/shared"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

type UploadHandler struct {
	db          *gorm.DB
	logosClient *shared.HTTPClient
}

func NewUploadHandler(db *gorm.DB) *UploadHandler {
	return &UploadHandler{
		db:          db,
		logosClient: shared.NewHTTPClient("thesaurus", "http://logos:8003"),
	}
}

// UploadDocument handles file upload with correlation ID threading
func (h *UploadHandler) UploadDocument(c *gin.Context) {
	correlationID := middleware.GetCorrelationID(c)
	
	// Log the start of upload processing
	shared.LogBusinessLogic(correlationID, "thesaurus", "Starting document upload processing")

	// Get user ID from auth middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":          "User not authenticated",
			"correlation_id": correlationID,
		})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "No file provided",
			"correlation_id": correlationID,
		})
		return
	}

	// Validate file
	if !h.isAllowedFileType(file.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "File type not allowed",
			"correlation_id": correlationID,
		})
		return
	}

	// Create document record
	documentID := fmt.Sprintf("doc_%s", uuid.New().String()[:16])
	document := models.Document{
		ID:               documentID,
		UserID:          userID.(string),
		Filename:        file.Filename,
		OriginalFilename: file.Filename,
		FileSize:        file.Size,
		Status:          "uploaded",
		CorrelationID:   correlationID, // Store correlation ID with document
	}

	// Save to database
	if err := h.db.Create(&document).Error; err != nil {
		shared.LogDatabaseError(correlationID, "thesaurus", "create document", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":          "Failed to save document",
			"correlation_id": correlationID,
		})
		return
	}

	// Save file to disk (this would normally be done more securely)
	filePath := fmt.Sprintf("/tmp/%s_%s", documentID, file.Filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		shared.LogEntry{
			CorrelationID: correlationID,
			ServiceName:   "thesaurus",
			Type:          "file_error",
			Message:       fmt.Sprintf("Failed to save file: %v", err),
			Path:          c.Request.URL.Path,
			Timestamp:     time.Now().Format(time.RFC3339Nano),
		}.Log("")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":          "Failed to save file",
			"correlation_id": correlationID,
		})
		return
	}

	// Send to Logos service for processing with correlation ID
	processingRequest := map[string]interface{}{
		"document_id":    documentID,
		"file_path":      filePath,
		"file_type":      getFileExtension(file.Filename),
		"user_id":        userID,
		"correlation_id": correlationID,
	}

	ctx := context.Background()
	opts := &shared.RequestOptions{
		CorrelationID: correlationID,
		Headers: map[string]string{
			"X-Request-Source": "thesaurus-upload",
		},
	}

	// Call Logos service asynchronously
	go func() {
		shared.LogServiceCall(correlationID, "thesaurus", "logos", "/api/process/document")
		
		resp, err := h.logosClient.Post(ctx, "/api/process/document", processingRequest, opts)
		if err != nil {
			shared.LogServiceError(correlationID, "thesaurus", "logos", err)

			// Update document status to error
			h.db.Model(&document).Updates(map[string]interface{}{
				"status":        "error",
				"error_message": fmt.Sprintf("Processing service unavailable: %v", err),
			})
			return
		}
		resp.Body.Close()

		shared.LogBusinessLogic(correlationID, "thesaurus", "Successfully sent document to Logos for processing")
	}()

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"message":        "File uploaded successfully",
		"document_id":    documentID,
		"filename":       file.Filename,
		"status":         "processing",
		"correlation_id": correlationID,
	})

	shared.LogBusinessLogic(correlationID, "thesaurus", "Document upload processing completed successfully")
}

// GetProcessingStatus returns the processing status of a document
func (h *UploadHandler) GetProcessingStatus(c *gin.Context) {
	correlationID := middleware.GetCorrelationID(c)
	documentID := c.Param("id")

	shared.LogBusinessLogic(correlationID, "thesaurus", fmt.Sprintf("Checking processing status for document: %s", documentID))

	var document models.Document
	if err := h.db.Where("id = ?", documentID).First(&document).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":          "Document not found",
			"correlation_id": correlationID,
		})
		return
	}

	// Get status from Logos service if still processing
	if document.Status == "processing" {
		ctx := context.Background()
		opts := &shared.RequestOptions{
			CorrelationID: correlationID,
		}

		var statusResp map[string]interface{}
		err := h.logosClient.GetJSON(ctx, fmt.Sprintf("/api/process/status/%s", documentID), &statusResp, opts)
		if err == nil && statusResp != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":         statusResp["status"],
				"progress":       statusResp["progress"],
				"message":        statusResp["message"],
				"correlation_id": correlationID,
			})
			return
		}
	}

	// Return status from database
	c.JSON(http.StatusOK, gin.H{
		"status":         document.Status,
		"progress":       100,
		"message":        "Processing completed",
		"correlation_id": correlationID,
	})
}

// Helper functions
func (h *UploadHandler) isAllowedFileType(filename string) bool {
	allowedExtensions := []string{".pdf", ".csv", ".xlsx", ".xls"}
	ext := getFileExtension(filename)
	
	for _, allowed := range allowedExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}