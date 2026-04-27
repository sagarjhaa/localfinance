package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/logos/config"
	"github.com/sagarjhaa/localfinance/services/logos/models"
	"github.com/sagarjhaa/localfinance/services/logos/processors"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

// supportedFormats is what Logos can hand off to Sophia. Excel binary
// parsing was dropped along with the regex processors — users should
// convert .xlsx to .csv before uploading.
var supportedFormats = []string{"csv", "pdf", "txt"}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	router := gin.New()
	router.Use(middleware.CorrelationMiddleware("logos"))
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":            "healthy",
			"service":           "logos",
			"version":           "1.0.0",
			"supported_formats": supportedFormats,
		})
	})

	// Process endpoint — receives document from Thesaurus, extracts text,
	// hands off to Sophia for AI parsing, sends transactions back.
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

		go processDocument(req, cfg)
	})

	// Supported formats info
	router.GET("/api/v1/info/formats", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"supported_formats": supportedFormats,
			"max_file_size":     cfg.Processing.MaxFileSize,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}

	log.Printf("📜 Logos service starting on port %s", port)
	log.Printf("📄 Supported formats: %v", supportedFormats)
	log.Printf("🔗 Thesaurus URL: %s", cfg.Thesaurus.BaseURL)
	log.Printf("🔗 Sophia URL: %s (AI parse)", cfg.Sophia.BaseURL)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Logos service: %v", err)
	}
}

func processDocument(req models.ProcessRequest, cfg *config.Config) {
	start := time.Now()
	log.Printf("[%s] Processing document: %s (type: %s)", req.DocumentID, req.FilePath, req.FileType)

	// File must exist and be readable
	stat, err := os.Stat(req.FilePath)
	if err != nil {
		log.Printf("[%s] Error opening file: %v", req.DocumentID, err)
		updateDocumentStatus(cfg.Thesaurus.BaseURL, req.DocumentID, "error", fmt.Sprintf("Cannot open file: %v", err), "")
		return
	}

	ext := strings.ToLower(filepath.Ext(req.FilePath))

	// Extract raw text from the file. PDF: pdftotext + Go fallback. CSV/TXT:
	// read bytes as UTF-8. XLSX: refused — user must convert to CSV.
	rawText, err := extractTextForAI(req.FilePath, ext)
	if err != nil {
		log.Printf("[%s] text extraction failed: %v", req.DocumentID, err)
		updateDocumentStatus(cfg.Thesaurus.BaseURL, req.DocumentID, "error",
			fmt.Sprintf("Text extraction failed: %v", err), "")
		return
	}

	// Send extracted text to Sophia for AI parsing.
	transactions, modelUsed, err := parseViaSophia(cfg.Sophia.BaseURL, req.UserID.String(), rawText, req.FilePath)
	if err != nil {
		log.Printf("[%s] Sophia parse failed: %v", req.DocumentID, err)
		updateDocumentStatus(cfg.Thesaurus.BaseURL, req.DocumentID, "error",
			fmt.Sprintf("AI parse failed: %v", err), "")
		return
	}

	if len(transactions) == 0 {
		log.Printf("[%s] Sophia returned 0 transactions", req.DocumentID)
		updateDocumentStatus(cfg.Thesaurus.BaseURL, req.DocumentID, "error",
			"AI returned no transactions from this file", rawText)
		return
	}

	log.Printf("[%s] Sophia parsed %d transactions using model %s", req.DocumentID, len(transactions), modelUsed)

	result := processors.ProcessResult{
		Transactions:      transactions,
		TransactionsFound: len(transactions),
		ProcessingTimeMs:  time.Since(start).Milliseconds(),
		FileSize:          stat.Size(),
	}

	// Normalize transaction type from amount sign
	for i := range result.Transactions {
		if result.Transactions[i].Amount >= 0 {
			result.Transactions[i].Type = "credit"
		} else {
			result.Transactions[i].Type = "debit"
		}
		if result.Transactions[i].FileSource == "" {
			result.Transactions[i].FileSource = req.FilePath
		}
	}

	// Step 1: Apply user's custom category rules (if any)
	applyUserRules(result.Transactions, req.UserID)

	// Step 2: AI categorize remaining uncategorized transactions via Ollama
	aiCategorize(result.Transactions)

	// Detect statement metadata from extracted text
	meta := processors.DetectStatementInfo(rawText)
	log.Printf("[%s] Detected: type=%s institution=%s account=%s", req.DocumentID, meta.AccountType, meta.Institution, meta.AccountNumber)

	// Send transactions + metadata to Thesaurus
	if result.TransactionsFound > 0 && req.AccountID.String() != "00000000-0000-0000-0000-000000000000" {
		err = sendTransactionsWithMetadata(cfg.Thesaurus.BaseURL, req.AccountID, req.DocumentID, result.Transactions, meta)
		if err != nil {
			log.Printf("[%s] Error sending to Thesaurus: %v", req.DocumentID, err)
			updateDocumentStatus(cfg.Thesaurus.BaseURL, req.DocumentID, "error", fmt.Sprintf("Failed to save transactions: %v", err), "")
			return
		}

		// Fire-and-forget month-review generation. Period is inferred from the
		// first transaction's date (statements typically cover one month); falls
		// back to "now" if dates are missing. Failures are logged but do not
		// affect the upload's success status.
		period := inferPeriod(result.Transactions)
		go triggerMonthReview(req.UserID.String(), period, req.DocumentID)
	}

	// Update document status to processed (include extracted text for AI comparison)
	updateDocumentStatus(cfg.Thesaurus.BaseURL, req.DocumentID, "processed", "", rawText)
	log.Printf("[%s] Processing complete — %d transactions saved", req.DocumentID, result.TransactionsFound)
}

// extractTextForAI returns the file's raw text suitable for forwarding to
// Sophia's /api/v1/parse endpoint.
//   - PDF: uses processors.ExtractPDFText (pdftotext with Go fallback).
//   - CSV/TXT: reads file bytes verbatim as UTF-8.
//   - XLSX/XLS: refused — Logos no longer parses Excel binary formats.
//     Users should re-export to CSV.
func extractTextForAI(filePath, ext string) (string, error) {
	switch ext {
	case ".pdf":
		return processors.ExtractPDFText(filePath)
	case ".csv", ".txt", ".tsv":
		b, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", ext, err)
		}
		return string(b), nil
	case ".xlsx", ".xls":
		return "", fmt.Errorf("excel binary format no longer supported — please convert to CSV before uploading")
	default:
		// Best-effort: try reading as text. If it's binary garbage Sophia
		// will simply return zero transactions and we'll surface a clear
		// error to the user.
		b, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("unsupported file type %q: %w", ext, err)
		}
		return string(b), nil
	}
}

// parseViaSophia POSTs the extracted text to Sophia's AI parse endpoint and
// converts the returned []map[string]interface{} into []models.Transaction.
// Uses a 5-minute timeout because LLM parsing on 8b/14b models is slow.
func parseViaSophia(sophiaURL, userID, text, fileSource string) ([]models.Transaction, string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, "", fmt.Errorf("empty text — nothing to parse")
	}

	payload, _ := json.Marshal(map[string]string{
		"text":    text,
		"user_id": userID,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		sophiaURL+"/api/v1/parse", bytes.NewReader(payload))
	if err != nil {
		return nil, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("call sophia: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("sophia returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Transactions []map[string]interface{} `json:"transactions"`
		Count        int                      `json:"count"`
		Model        string                   `json:"model"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", fmt.Errorf("decode sophia response: %w", err)
	}

	transactions := make([]models.Transaction, 0, len(parsed.Transactions))
	for _, m := range parsed.Transactions {
		t := mapToTransaction(m, fileSource)
		// Skip junk rows with neither description nor amount
		if strings.TrimSpace(t.Description) == "" && t.Amount == 0 {
			continue
		}
		transactions = append(transactions, t)
	}

	return transactions, parsed.Model, nil
}

// mapToTransaction converts a single Sophia parse-response map into a
// models.Transaction. Field names match Sophia's output:
// {date, description, amount, category, type, reference}.
func mapToTransaction(m map[string]interface{}, fileSource string) models.Transaction {
	t := models.Transaction{
		FileSource: fileSource,
		CreatedAt:  time.Now(),
	}

	if s, ok := m["description"].(string); ok {
		t.Description = strings.TrimSpace(s)
	}
	if s, ok := m["category"].(string); ok {
		t.Category = strings.TrimSpace(s)
	}
	if s, ok := m["type"].(string); ok {
		t.Type = strings.TrimSpace(s)
	}
	if s, ok := m["reference"].(string); ok {
		t.Reference = strings.TrimSpace(s)
	}

	switch v := m["amount"].(type) {
	case float64:
		t.Amount = v
	case int:
		t.Amount = float64(v)
	case string:
		// Tolerate numeric strings like "12.34" or "-12.34"
		clean := strings.ReplaceAll(strings.TrimSpace(v), ",", "")
		clean = strings.TrimPrefix(clean, "$")
		if f, err := parseFloat(clean); err == nil {
			t.Amount = f
		}
	}

	if s, ok := m["date"].(string); ok {
		t.Date = parseDate(s)
	}
	if t.Date.IsZero() {
		t.Date = time.Now()
	}

	return t
}

// parseFloat is a tiny stdlib-only float parser wrapper.
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// parseDate handles the date formats Sophia tends to emit. Returns zero
// time on failure so callers can fall back.
func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	formats := []string{
		"2006-01-02",
		time.RFC3339,
		"01/02/2006",
		"02/01/2006",
		"01-02-2006",
		"2006/01/02",
		"Jan 2, 2006",
		"Jan 02, 2006",
		"2 Jan 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
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

func updateDocumentStatus(baseURL, documentID, status, errMsg, extractedText string) {
	payload, _ := json.Marshal(map[string]string{
		"status":         status,
		"error_message":  errMsg,
		"extracted_text": extractedText,
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

// inferPeriod returns a "YYYY-MM" string derived from the first transaction
// with a non-zero date; falls back to the current month (UTC).
func inferPeriod(txns []models.Transaction) string {
	for _, t := range txns {
		if !t.Date.IsZero() {
			d := t.Date.UTC()
			return fmt.Sprintf("%04d-%02d", d.Year(), int(d.Month()))
		}
	}
	now := time.Now().UTC()
	return fmt.Sprintf("%04d-%02d", now.Year(), int(now.Month()))
}

// triggerMonthReview posts to Sophia's internal generate endpoint. Wrapped in
// a 60s timeout so a slow LLM doesn't pin the goroutine indefinitely. All
// errors are logged but never propagated — month-review is best-effort.
func triggerMonthReview(userID, period, documentID string) {
	if userID == "" || userID == "00000000-0000-0000-0000-000000000000" {
		return
	}
	sophiaURL := os.Getenv("SOPHIA_URL")
	if sophiaURL == "" {
		sophiaURL = "http://localhost:8002"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	payload, _ := json.Marshal(map[string]string{
		"user_id": userID,
		"period":  period,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		sophiaURL+"/api/v1/internal/month-review/generate",
		bytes.NewReader(payload))
	if err != nil {
		log.Printf("[%s] month-review trigger: build request failed: %v", documentID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[%s] month-review trigger failed (non-fatal): %v", documentID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("[%s] month-review trigger returned status %d", documentID, resp.StatusCode)
		return
	}
	log.Printf("[%s] month-review triggered for period %s", documentID, period)
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
