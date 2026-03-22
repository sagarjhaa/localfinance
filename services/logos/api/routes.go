package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/logos/api/handlers"
	"github.com/sagarjhaa/localfinance/services/logos/config"
	"github.com/sagarjhaa/localfinance/services/logos/processors"
	"github.com/sagarjhaa/localfinance/services/logos/storage"
)

func SetupRoutes(router *gin.Engine, processorManager *processors.Manager, storageClient *storage.MinIOClient, cfg *config.Config) {
	// Initialize handlers
	uploadHandler := handlers.NewUploadHandler(processorManager, storageClient, cfg)
	processingHandler := handlers.NewProcessingHandler(processorManager, cfg)
	statusHandler := handlers.NewStatusHandler()

	// Configure file upload limits
	router.MaxMultipartMemory = 50 << 20 // 50 MB

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "logos",
			"supported_formats": processorManager.GetSupportedTypes(),
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// File upload routes
		upload := v1.Group("/upload")
		{
			upload.POST("/", uploadHandler.UploadDocument)
			upload.POST("/batch", uploadHandler.UploadDocumentBatch)
		}

		// Document processing routes
		documents := v1.Group("/documents")
		{
			documents.GET("/", processingHandler.ListDocuments)
			documents.GET("/:id", processingHandler.GetDocument)
			documents.DELETE("/:id", processingHandler.DeleteDocument)
			documents.POST("/:id/reprocess", processingHandler.ReprocessDocument)
		}

		// Processing status routes
		processing := v1.Group("/processing")
		{
			processing.GET("/status/:jobId", statusHandler.GetProcessingStatus)
			processing.GET("/stats", statusHandler.GetProcessingStats)
			processing.GET("/queue", statusHandler.GetProcessingQueue)
		}

		// Configuration and info routes
		info := v1.Group("/info")
		{
			info.GET("/formats", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"supported_formats": processorManager.GetSupportedTypes(),
					"max_file_size": cfg.Processing.MaxFileSize,
				})
			})
		}
	}
}