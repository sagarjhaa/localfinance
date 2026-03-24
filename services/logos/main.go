package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	pdf "github.com/ledongthuc/pdf"
	"github.com/sagarjhaa/localfinance/services/logos/config"
	"github.com/sagarjhaa/localfinance/services/logos/models"
	"github.com/sagarjhaa/localfinance/services/logos/processors"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize processor manager
	processorManager := processors.NewManager()

	router := gin.New()
	router.Use(middleware.CorrelationMiddleware("logos"))
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":            "healthy",
			"service":           "logos",
			"version":           "1.0.0",
			"supported_formats": processorManager.GetSupportedTypes(),
		})
	})

	// Process endpoint — receives document from Thesaurus, parses, sends transactions back
	router.POST("/api/v1/process", func(c *gin.Context) {
		var req models.ProcessRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Acknowledge immediately, process in background
		c.JSON(202, gin.H{
			"message":     "Processing started",
			"document_id": req.DocumentID,
		})

		// Process asynchronously
		go processDocument(req, processorManager, cfg)
	})

	// Supported formats info
	router.GET("/api/v1/info/formats", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"supported_formats": processorManager.GetSupportedTypes(),
			"max_file_size":     cfg.Processing.MaxFileSize,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}

	log.Printf("📜 Logos service starting on port %s", port)
	log.Printf("📄 Supported formats: %v", processorManager.GetSupportedTypes())
	log.Printf("🔗 Thesaurus URL: %s", cfg.Thesaurus.BaseURL)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Logos service: %v", err)
	}
}

func processDocument(req models.ProcessRequest, pm *processors.Manager, cfg *config.Config) {
	start := time.Now()
	log.Printf("[%s] Processing document: %s (type: %s)", req.DocumentID, req.FilePath, req.FileType)

	// Open the file
	file, err := os.Open(req.FilePath)
	if err != nil {
		log.Printf("[%s] Error opening file: %v", req.DocumentID, err)
		updateDocumentStatus(cfg.Thesaurus.BaseURL, req.DocumentID, "error", fmt.Sprintf("Cannot open file: %v", err))
		return
	}
	defer file.Close()

	// Get file info for size
	stat, _ := file.Stat()

	// Process the document
	result, err := pm.ProcessDocument(file, req.FilePath)
	if err != nil {
		log.Printf("[%s] Error processing: %v", req.DocumentID, err)
		updateDocumentStatus(cfg.Thesaurus.BaseURL, req.DocumentID, "error", fmt.Sprintf("Processing failed: %v", err))
		return
	}

	result.ProcessingTimeMs = time.Since(start).Milliseconds()
	if stat != nil {
		result.FileSize = stat.Size()
	}

	log.Printf("[%s] Parsed %d transactions in %dms", req.DocumentID, result.TransactionsFound, result.ProcessingTimeMs)

	// Auto-categorize transactions
	for i := range result.Transactions {
		if result.Transactions[i].Category == "" {
			result.Transactions[i].Category = categorize(result.Transactions[i].Description)
		}
		if result.Transactions[i].Amount >= 0 {
			result.Transactions[i].Type = "credit"
		} else {
			result.Transactions[i].Type = "debit"
		}
	}

	// Detect statement metadata from file content
	var meta processors.StatementMetadata
	fileBytes, readErr := os.ReadFile(req.FilePath)
	if readErr == nil {
		rawText := string(fileBytes) // works for CSV; for PDF we need extracted text
		if strings.HasSuffix(strings.ToLower(req.FilePath), ".pdf") {
			// Re-extract PDF text for detection
			if pdfText, err := extractPDFTextForDetection(req.FilePath); err == nil {
				rawText = pdfText
			}
		}
		meta = processors.DetectStatementInfo(rawText)
		log.Printf("[%s] Detected: type=%s institution=%s account=%s", req.DocumentID, meta.AccountType, meta.Institution, meta.AccountNumber)
	}

	// Send transactions + metadata to Thesaurus
	if result.TransactionsFound > 0 && req.AccountID.String() != "00000000-0000-0000-0000-000000000000" {
		err = sendTransactionsWithMetadata(cfg.Thesaurus.BaseURL, req.AccountID, req.DocumentID, result.Transactions, meta)
		if err != nil {
			log.Printf("[%s] Error sending to Thesaurus: %v", req.DocumentID, err)
			updateDocumentStatus(cfg.Thesaurus.BaseURL, req.DocumentID, "error", fmt.Sprintf("Failed to save transactions: %v", err))
			return
		}
	}

	// Update document status to processed
	updateDocumentStatus(cfg.Thesaurus.BaseURL, req.DocumentID, "processed", "")
	log.Printf("[%s] Processing complete — %d transactions saved", req.DocumentID, result.TransactionsFound)
}

func extractPDFTextForDetection(filePath string) (string, error) {
	pdfFile, reader, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer pdfFile.Close()

	var text strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		text.WriteString(content)
		text.WriteString("\n")
	}
	return text.String(), nil
}

func sendTransactionsWithMetadata(baseURL string, accountID fmt.Stringer, documentID string, transactions []models.Transaction, meta processors.StatementMetadata) error {
	type thesaurusTxn struct {
		Date        time.Time `json:"date"`
		Description string    `json:"description"`
		Amount      float64   `json:"amount"`
		Category    string    `json:"category"`
		Type        string    `json:"type"`
		Reference   string    `json:"reference"`
		DocumentID  string    `json:"document_id"`
	}

	txns := make([]thesaurusTxn, len(transactions))
	for i, t := range transactions {
		txns[i] = thesaurusTxn{
			Date: t.Date, Description: t.Description, Amount: t.Amount,
			Category: t.Category, Type: t.Type, Reference: t.Reference,
			DocumentID: documentID,
		}
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"account_id":   accountID.String(),
		"transactions": txns,
		"metadata":     meta,
	})

	resp, err := http.Post(baseURL+"/api/v1/transactions/bulk", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Thesaurus returned status %d", resp.StatusCode)
	}
	return nil
}

func sendTransactionsToThesaurus(baseURL string, accountID fmt.Stringer, documentID string, transactions []models.Transaction) error {
	// Convert to Thesaurus format
	type thesaurusTxn struct {
		Date        time.Time `json:"date"`
		Description string    `json:"description"`
		Amount      float64   `json:"amount"`
		Category    string    `json:"category"`
		Type        string    `json:"type"`
		Reference   string    `json:"reference"`
		DocumentID  string    `json:"document_id"`
	}

	txns := make([]thesaurusTxn, len(transactions))
	for i, t := range transactions {
		txns[i] = thesaurusTxn{
			Date:        t.Date,
			Description: t.Description,
			Amount:      t.Amount,
			Category:    t.Category,
			Type:        t.Type,
			Reference:   t.Reference,
			DocumentID:  documentID,
		}
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"account_id":   accountID.String(),
		"transactions": txns,
	})

	resp, err := http.Post(baseURL+"/api/v1/transactions/bulk", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Thesaurus returned status %d", resp.StatusCode)
	}
	return nil
}

func updateDocumentStatus(baseURL, documentID, status, errMsg string) {
	payload, _ := json.Marshal(map[string]string{
		"status":        status,
		"error_message": errMsg,
	})

	req, _ := http.NewRequest("PATCH", baseURL+"/api/v1/documents/"+documentID+"/status", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[%s] Failed to update status: %v", documentID, err)
		return
	}
	resp.Body.Close()
}

func categorize(desc string) string {
	d := strings.ToLower(desc)
	switch {
	case containsAny(d, "grocery", "supermarket", "food", "restaurant", "cafe", "coffee", "swiggy", "zomato", "blinkit", "bigbasket"):
		return "Food"
	case containsAny(d, "fuel", "petrol", "uber", "ola", "metro", "bus", "train", "transport", "parking", "irctc"):
		return "Transport"
	case containsAny(d, "netflix", "spotify", "amazon prime", "disney", "youtube", "entertainment", "movie", "hotstar"):
		return "Entertainment"
	case containsAny(d, "electric", "water", "gas bill", "internet", "wifi", "phone", "mobile", "utility", "broadband", "jio", "airtel"):
		return "Utilities"
	case containsAny(d, "rent", "mortgage", "housing", "property", "maintenance", "society"):
		return "Housing"
	case containsAny(d, "salary", "wages", "income", "deposit", "refund", "cashback", "interest"):
		return "Income"
	case containsAny(d, "transfer", "upi", "neft", "imps", "rtgs", "nach"):
		return "Transfer"
	case containsAny(d, "insurance", "medical", "hospital", "doctor", "pharmacy", "health", "apollo"):
		return "Health"
	case containsAny(d, "amazon", "flipkart", "myntra", "shop", "store", "purchase", "buy"):
		return "Shopping"
	case containsAny(d, "atm", "withdrawal", "cash"):
		return "Cash"
	case containsAny(d, "emi", "loan", "repay"):
		return "EMI"
	default:
		return "Other"
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
