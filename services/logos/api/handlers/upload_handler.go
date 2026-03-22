package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/logos/config"
	"github.com/sagarjhaa/localfinance/services/logos/models"
	"github.com/sagarjhaa/localfinance/services/logos/processors"
	"github.com/sagarjhaa/localfinance/services/logos/storage"
)

type UploadHandler struct {
	processorManager *processors.Manager
	storageClient    *storage.MinIOClient
	config          *config.Config
}

func NewUploadHandler(processorManager *processors.Manager, storageClient *storage.MinIOClient, cfg *config.Config) *UploadHandler {
	return &UploadHandler{
		processorManager: processorManager,
		storageClient:   storageClient,
		config:          cfg,
	}
}

func (h *UploadHandler) UploadDocument(c *gin.Context) {
	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Check file size
	if file.Size > h.config.Processing.MaxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File too large. Max size: %d bytes", h.config.Processing.MaxFileSize),
		})
		return
	}

	// Check if file type is supported
	if !h.processorManager.CanProcessFile(file.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unsupported file format",
			"supported_formats": h.processorManager.GetSupportedTypes(),
		})
		return
	}

	// Open the uploaded file
	fileReader, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read uploaded file"})
		return
	}
	defer fileReader.Close()

	// Generate unique filename for storage
	ext := filepath.Ext(file.Filename)
	storageFilename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Upload to storage
	storageURL, err := h.storageClient.UploadFile(storageFilename, fileReader, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store file",
			"details": err.Error(),
		})
		return
	}

	// Reopen file for processing
	fileReader, err = file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file for processing"})
		return
	}
	defer fileReader.Close()

	// Process the document
	startTime := time.Now()
	processingResult, err := h.processorManager.ProcessDocument(fileReader, file.Filename)
	processingTime := time.Since(startTime)

	if err != nil {
		// Processing failed, but file is stored
		result := models.ProcessingResult{
			ID:               uuid.New(),
			Filename:         file.Filename,
			FileType:         h.getFileType(file.Filename),
			FileSize:         file.Size,
			Status:           "error",
			ErrorMessage:     err.Error(),
			ProcessedAt:      time.Now(),
			ProcessingTimeMs: processingTime.Milliseconds(),
		}

		c.JSON(http.StatusOK, gin.H{
			"processing_result": result,
			"storage_url": storageURL,
			"warning": "File uploaded but processing failed",
		})
		return
	}

	// Processing successful
	processingResult.ID = uuid.New()
	processingResult.FileSize = file.Size
	processingResult.ProcessedAt = time.Now()
	processingResult.ProcessingTimeMs = processingTime.Milliseconds()

	// TODO: Send transactions to Thesaurus service
	// TODO: Send transactions to Sophia for categorization

	c.JSON(http.StatusOK, gin.H{
		"processing_result": processingResult,
		"storage_url": storageURL,
		"message": fmt.Sprintf("Successfully processed %d transactions", processingResult.TransactionsFound),
	})
}

func (h *UploadHandler) UploadDocumentBatch(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files uploaded"})
		return
	}

	if len(files) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 10 files per batch"})
		return
	}

	results := make([]gin.H, len(files))

	for i, file := range files {
		// Process each file individually
		fileReader, err := file.Open()
		if err != nil {
			results[i] = gin.H{
				"filename": file.Filename,
				"error": "Failed to read file",
			}
			continue
		}

		// Check file size and type
		if file.Size > h.config.Processing.MaxFileSize {
			results[i] = gin.H{
				"filename": file.Filename,
				"error": "File too large",
			}
			fileReader.Close()
			continue
		}

		if !h.processorManager.CanProcessFile(file.Filename) {
			results[i] = gin.H{
				"filename": file.Filename,
				"error": "Unsupported file format",
			}
			fileReader.Close()
			continue
		}

		// Process the file
		processingResult, err := h.processorManager.ProcessDocument(fileReader, file.Filename)
		fileReader.Close()

		if err != nil {
			results[i] = gin.H{
				"filename": file.Filename,
				"error": err.Error(),
			}
		} else {
			results[i] = gin.H{
				"filename": file.Filename,
				"transactions_found": processingResult.TransactionsFound,
				"status": "success",
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_files": len(files),
		"results": results,
	})
}

func (h *UploadHandler) getFileType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".csv":
		return "csv"
	case ".pdf":
		return "pdf"
	case ".xlsx", ".xls":
		return "excel"
	default:
		return "unknown"
	}
}