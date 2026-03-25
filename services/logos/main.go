package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	// Set transaction type
	for i := range result.Transactions {
		if result.Transactions[i].Amount >= 0 {
			result.Transactions[i].Type = "credit"
		} else {
			result.Transactions[i].Type = "debit"
		}
	}

	// Step 1: Apply user's custom category rules (if any)
	applyUserRules(result.Transactions, req.UserID)

	// Step 2: AI categorize remaining uncategorized transactions via Ollama
	aiCategorize(result.Transactions)

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

// aiCategorize uses Ollama to categorize transactions in a single batch prompt
func aiCategorize(transactions []models.Transaction) {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://127.0.0.1:11434"
	}

	if len(transactions) == 0 {
		return
	}

	// Build a numbered list of descriptions
	var sb strings.Builder
	for i, tx := range transactions {
		desc := tx.Description
		if len(desc) > 60 {
			desc = desc[:60]
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, desc))
	}

	prompt := fmt.Sprintf(`Categorize each transaction into exactly one category.

Categories: Food, Transport, Shopping, Entertainment, Utilities, Housing, Income, Transfer, Health, Cash, EMI, Education, Other

Rules:
- Restaurants, grocery stores, food delivery = Food
- Gas stations, rideshare, flights, trains = Transport
- Online shopping, retail stores = Shopping
- Streaming services, movies = Entertainment
- Only use "Other" if nothing else fits

Respond with ONLY numbered lines in format: NUMBER. CATEGORY
No explanations.

Transactions:
%s`, sb.String())

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":       "llama3.2:1b",
		"prompt":      prompt,
		"stream":      false,
		"temperature": 0.1,
	})

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(ollamaHost+"/api/generate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("AI categorization failed (Ollama unreachable): %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("AI categorization failed: status %d", resp.StatusCode)
		return
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		log.Printf("AI categorization: failed to decode response: %v", err)
		return
	}

	// Parse response — expect lines like "1. Food\n2. Transport\n..."
	validCategories := map[string]bool{
		"Food": true, "Transport": true, "Shopping": true, "Entertainment": true,
		"Utilities": true, "Housing": true, "Income": true, "Transfer": true,
		"Health": true, "Cash": true, "EMI": true, "Education": true, "Other": true,
	}

	lines := strings.Split(ollamaResp.Response, "\n")
	updated := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Parse "1. Food" or "1: Food"
		parts := strings.SplitN(line, ".", 2)
		if len(parts) != 2 {
			parts = strings.SplitN(line, ":", 2)
		}
		if len(parts) != 2 {
			continue
		}

		numStr := strings.TrimSpace(parts[0])
		cat := strings.TrimSpace(parts[1])
		// Clean category — remove any extra text after the category name
		cat = strings.TrimSpace(strings.Split(cat, " ")[0])
		cat = strings.TrimSpace(strings.Split(cat, "-")[0])
		cat = strings.TrimRight(cat, ".,;:")

		idx := 0
		if _, err := fmt.Sscanf(numStr, "%d", &idx); err != nil || idx < 1 || idx > len(transactions) {
			continue
		}

		if validCategories[cat] {
			transactions[idx-1].Category = cat
			updated++
		}
	}

	log.Printf("AI categorized %d/%d transactions", updated, len(transactions))
}

// applyUserRules checks user's custom merchant→category rules from Thesaurus
func applyUserRules(transactions []models.Transaction, userID uuid.UUID) {
	thesaurusURL := os.Getenv("THESAURUS_URL")
	if thesaurusURL == "" {
		thesaurusURL = "http://localhost:8001"
	}

	if userID == uuid.Nil || len(transactions) == 0 {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	matched := 0

	for i := range transactions {
		if transactions[i].Category != "" {
			continue // already categorized
		}
		// Query Thesaurus internal match endpoint
		u := fmt.Sprintf("%s/api/v1/internal/category-rules/match?user_id=%s&description=%s",
			thesaurusURL, userID.String(), url.QueryEscape(transactions[i].Description))
		resp, err := client.Get(u)
		if err != nil {
			continue
		}
		var result struct {
			Matched  bool   `json:"matched"`
			Category string `json:"category"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if result.Matched && result.Category != "" {
			transactions[i].Category = result.Category
			matched++
		}
	}

	if matched > 0 {
		log.Printf("User rules matched %d/%d transactions", matched, len(transactions))
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
