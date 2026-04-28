// Package parse owns document text extraction, AI parsing, categorization,
// and persistence. It replaces the HTTP-driven Logos service: the same
// pipeline now runs in-process on the unified binary, calling internal/ai
// directly and writing through the shared *gorm.DB.
package parse

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/internal/ai"
	dmodels "github.com/sagarjhaa/localfinance/internal/data/models"
	"gorm.io/gorm"
)

// maxPDFPagesForVision caps the number of PDF pages we render for the
// vision-parse path. Statements over this size fall back to text extraction.
const maxPDFPagesForVision = 25

// Request is what the upload handler hands the pipeline.
type Request struct {
	DocumentID string
	FilePath   string
	FileType   string
	UserID     uuid.UUID
	AccountID  uuid.UUID
}

// Transaction is the in-flight shape between AI parse and DB write. Mirrors
// the old services/logos/models.Transaction so the conversion code can be
// reused verbatim.
type Transaction struct {
	Date        time.Time
	Description string
	Amount      float64
	Category    string
	Type        string
	Reference   string
	FileSource  string
	CreatedAt   time.Time
}

// MonthReviewTrigger is the minimal interface the pipeline needs to kick a
// month-review generation. Implemented by *monthreview.Service.
type MonthReviewTrigger interface {
	Generate(ctx context.Context, userID string, period any) error
}

// Pipeline orchestrates extract -> AI parse -> categorize -> persist.
type Pipeline struct {
	DB        *gorm.DB
	AI        *ai.Service
	ModelName string

	// OnPersisted, if non-nil, is called after a successful bulk insert. Used
	// to fire-and-forget month-review generation. Best-effort; never blocks.
	OnPersisted func(userID, period, documentID string)
}

// New builds a pipeline with the given deps.
func New(db *gorm.DB, aiSvc *ai.Service, modelName string) *Pipeline {
	return &Pipeline{DB: db, AI: aiSvc, ModelName: modelName}
}

// ProcessDocument is the top-level entry point. Mirrors the old Logos
// processDocument main but with HTTP calls replaced by in-process equivalents.
func (p *Pipeline) ProcessDocument(ctx context.Context, req Request) {
	start := time.Now()
	log.Printf("[%s] Processing document: %s (type: %s)", req.DocumentID, req.FilePath, req.FileType)

	stat, err := os.Stat(req.FilePath)
	if err != nil {
		log.Printf("[%s] Error opening file: %v", req.DocumentID, err)
		p.updateDocumentStatus(req.DocumentID, "error", fmt.Sprintf("Cannot open file: %v", err), "")
		return
	}

	ext := strings.ToLower(filepath.Ext(req.FilePath))
	var transactions []Transaction
	var modelUsed string
	var rawText string
	var parseErr error

	// Kick off docling extraction in parallel for comparison. Best-effort:
	// if docling isn't installed or fails, this is a no-op. Output lands in
	// ~/Library/Application Support/LocalFinance/parse-debug/.
	if ext == ".pdf" {
		dumpDoclingComparison(req.FilePath, req.DocumentID)
	}

	// Text-first dispatch. Most bank statements are digital PDFs with
	// embedded text, and the text path is 3-5x faster than vision on the
	// same model. Fall back to vision (rendered page images) only when
	// text extraction yields nothing usable — typically scanned/image PDFs.
	rawText, textErr := extractTextForAI(req.FilePath, ext)
	if textErr == nil && strings.TrimSpace(rawText) != "" {
		transactions, modelUsed, parseErr = p.parseText(req.UserID.String(), rawText, req.FilePath)
	} else if ext == ".pdf" {
		log.Printf("[%s] text extraction empty/failed (%v); trying vision fallback", req.DocumentID, textErr)
		images, rErr := renderPDFToPNGs(req.FilePath)
		if rErr != nil {
			log.Printf("[%s] PDF render failed: %v", req.DocumentID, rErr)
			p.updateDocumentStatus(req.DocumentID, "error", fmt.Sprintf("Couldn't read this file: %v", rErr), "")
			return
		}
		log.Printf("[%s] rendered %d page(s) for vision parse", req.DocumentID, len(images))
		transactions, modelUsed, parseErr = p.parseImages(req.UserID.String(), images, req.FilePath)
	} else {
		log.Printf("[%s] text extraction failed: %v", req.DocumentID, textErr)
		p.updateDocumentStatus(req.DocumentID, "error", fmt.Sprintf("Text extraction failed: %v", textErr), "")
		return
	}

	if parseErr != nil {
		log.Printf("[%s] AI parse failed: %v", req.DocumentID, parseErr)
		p.updateDocumentStatus(req.DocumentID, "error", fmt.Sprintf("AI parse failed: %v", parseErr), rawText)
		return
	}

	if len(transactions) == 0 {
		log.Printf("[%s] AI returned 0 transactions", req.DocumentID)
		if rawText == "" {
			if t, _ := extractTextForAI(req.FilePath, ext); t != "" {
				rawText = t
			}
		}
		p.updateDocumentStatus(req.DocumentID, "error", "AI returned no transactions from this file", rawText)
		return
	}

	log.Printf("[%s] AI parsed %d transactions using model %s", req.DocumentID, len(transactions), modelUsed)

	if rawText == "" {
		if t, _ := extractTextForAI(req.FilePath, ext); t != "" {
			rawText = t
		}
	}

	for i := range transactions {
		if transactions[i].Amount >= 0 {
			transactions[i].Type = "credit"
		} else {
			transactions[i].Type = "debit"
		}
		if transactions[i].FileSource == "" {
			transactions[i].FileSource = req.FilePath
		}
	}

	p.applyUserRules(transactions, req.UserID)
	p.aiCategorize(transactions)

	meta := DetectStatementInfo(rawText)
	log.Printf("[%s] Detected: type=%s institution=%s account=%s",
		req.DocumentID, meta.AccountType, meta.Institution, meta.AccountNumber)

	if req.AccountID != uuid.Nil {
		if err := p.bulkInsertTransactions(req.AccountID, req.DocumentID, transactions, meta); err != nil {
			log.Printf("[%s] bulk insert failed: %v", req.DocumentID, err)
			p.updateDocumentStatus(req.DocumentID, "error",
				fmt.Sprintf("Failed to save transactions: %v", err), "")
			return
		}

		period := inferPeriod(transactions)
		if p.OnPersisted != nil {
			go p.OnPersisted(req.UserID.String(), period, req.DocumentID)
		}
	}

	p.updateDocumentStatus(req.DocumentID, "processed", "", rawText)
	log.Printf("[%s] Processing complete (%d txns) in %s",
		req.DocumentID, len(transactions), time.Since(start))
	_ = stat
}

// parseTimeout returns the wall-clock budget for a parse call. Text parses
// default to 30s; vision (much slower) defaults to 60s. Both can be
// overridden via env so power users with bigger models get headroom.
func parseTimeout(kind string) time.Duration {
	envKey := "PARSE_TIMEOUT_SEC"
	def := 120
	if kind == "vision" {
		envKey = "VISION_PARSE_TIMEOUT_SEC"
		def = 180
	}
	if v := os.Getenv(envKey); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return time.Duration(def) * time.Second
}

// withParseTimeout runs fn under a context with the configured deadline.
// When the deadline fires, ctx is canceled — the underlying http.Client
// aborts the in-flight Ollama request, which causes Ollama to drop the
// generation server-side and free the GPU. No zombie goroutines.
func withParseTimeout[T any](kind string, fn func(ctx context.Context) (T, error)) (T, error) {
	timeout := parseTimeout(kind)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	v, err := fn(ctx)
	if err != nil && ctx.Err() == context.DeadlineExceeded {
		var zero T
		return zero, fmt.Errorf("parse timed out after %s — try a smaller statement, switch to a faster model in Profile, or set PARSE_TIMEOUT_SEC", timeout)
	}
	return v, err
}

func (p *Pipeline) parseText(userID, text, fileSource string) ([]Transaction, string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, "", fmt.Errorf("empty text — nothing to parse")
	}
	userModel := p.ModelName
	if userID != "" {
		userModel = p.AI.GetUserModelPreference(userID)
	}
	parsed, err := withParseTimeout("text", func(ctx context.Context) ([]map[string]interface{}, error) {
		return p.AI.ParseTransactions(ctx, text, userModel)
	})
	if err != nil {
		return nil, "", err
	}
	return convertParsedTransactions(parsed, fileSource), userModel, nil
}

func (p *Pipeline) parseImages(userID string, images []string, fileSource string) ([]Transaction, string, error) {
	userModel := p.ModelName
	if userID != "" {
		userModel = p.AI.GetUserModelPreference(userID)
	}
	parsed, err := withParseTimeout("vision", func(ctx context.Context) ([]map[string]interface{}, error) {
		return p.AI.ParseTransactionsFromImages(ctx, images, userModel)
	})
	if err != nil {
		return nil, "", err
	}
	return convertParsedTransactions(parsed, fileSource), userModel, nil
}

// convertParsedTransactions converts the AI service's []map[string]interface{}
// payload into typed []Transaction. Shared between text and image paths.
func convertParsedTransactions(raw []map[string]interface{}, fileSource string) []Transaction {
	txns := make([]Transaction, 0, len(raw))
	for _, r := range raw {
		t := Transaction{FileSource: fileSource, CreatedAt: time.Now()}
		if d, ok := r["date"].(string); ok {
			t.Date = parseDate(d)
		}
		if d, ok := r["description"].(string); ok {
			t.Description = strings.TrimSpace(d)
		}
		switch v := r["amount"].(type) {
		case float64:
			t.Amount = v
		case int:
			t.Amount = float64(v)
		case string:
			clean := strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(v), ",", ""), "$")
			fmt.Sscanf(clean, "%f", &t.Amount)
		}
		if c, ok := r["category"].(string); ok {
			t.Category = strings.TrimSpace(c)
		}
		if c, ok := r["type"].(string); ok {
			t.Type = strings.TrimSpace(c)
		}
		if c, ok := r["reference"].(string); ok {
			t.Reference = strings.TrimSpace(c)
		}
		if t.Date.IsZero() {
			t.Date = time.Now()
		}
		if strings.TrimSpace(t.Description) == "" && t.Amount == 0 {
			continue
		}
		txns = append(txns, t)
	}
	return txns
}

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	formats := []string{
		"2006-01-02", time.RFC3339, "01/02/2006", "02/01/2006",
		"01-02-2006", "2006/01/02", "Jan 2, 2006", "Jan 02, 2006", "2 Jan 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// bulkInsertTransactions writes transactions and (if metadata is present)
// updates the account and creates a StatementPeriod. Mirrors the existing
// /api/v1/transactions/bulk handler logic so the new in-process path is
// behavior-compatible.
func (p *Pipeline) bulkInsertTransactions(accountID uuid.UUID, documentID string, txns []Transaction, meta StatementMetadata) error {
	if len(txns) == 0 {
		return nil
	}

	dbTxns := make([]dmodels.Transaction, len(txns))
	for i, t := range txns {
		dbTxns[i] = dmodels.Transaction{
			AccountID:   accountID,
			Date:        t.Date,
			Description: t.Description,
			Amount:      t.Amount,
			Category:    t.Category,
			Type:        t.Type,
			Reference:   t.Reference,
			DocumentID:  documentID,
		}
	}

	if err := p.DB.Create(&dbTxns).Error; err != nil {
		return fmt.Errorf("create txns: %w", err)
	}

	// Account metadata updates
	updates := map[string]interface{}{}
	if meta.AccountType != "" {
		updates["type"] = meta.AccountType
	}
	if meta.Institution != "" {
		updates["institution"] = meta.Institution
	}
	if meta.AccountNumber != "" {
		updates["account_number"] = meta.AccountNumber
	}
	if len(updates) > 0 {
		p.DB.Model(&dmodels.Account{}).Where("id = ?", accountID).Updates(updates)
	}

	// StatementPeriod aggregate
	var totalCredits, totalDebits float64
	for _, t := range txns {
		if t.Amount > 0 {
			totalCredits += t.Amount
		} else {
			totalDebits += t.Amount
		}
	}
	period := dmodels.StatementPeriod{
		AccountID:        accountID,
		DocumentID:       documentID,
		TotalCredits:     totalCredits,
		TotalDebits:      totalDebits,
		TransactionCount: len(txns),
	}
	if meta.PeriodStart != nil {
		period.PeriodStart = *meta.PeriodStart
	}
	if meta.PeriodEnd != nil {
		period.PeriodEnd = *meta.PeriodEnd
	}
	p.DB.Create(&period)
	return nil
}

func (p *Pipeline) updateDocumentStatus(documentID, status, errMsg, extractedText string) {
	updates := map[string]interface{}{"status": status}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	if extractedText != "" {
		updates["extracted_text"] = extractedText
	}
	if err := p.DB.Model(&dmodels.Document{}).Where("id = ?", documentID).Updates(updates).Error; err != nil {
		log.Printf("[%s] Failed to update status: %v", documentID, err)
	}
}

// applyUserRules looks up user category rules from the local DB and applies
// the first matching rule per uncategorized transaction.
func (p *Pipeline) applyUserRules(txns []Transaction, userID uuid.UUID) {
	if userID == uuid.Nil || len(txns) == 0 {
		return
	}
	var rules []dmodels.CategoryRule
	if err := p.DB.Where("user_id = ?", userID).Find(&rules).Error; err != nil {
		return
	}
	if len(rules) == 0 {
		return
	}
	matched := 0
	for i := range txns {
		if txns[i].Category != "" {
			continue
		}
		desc := strings.ToLower(txns[i].Description)
		for _, r := range rules {
			pat := strings.ToLower(r.Pattern)
			if pat != "" && strings.Contains(desc, pat) {
				txns[i].Category = r.Category
				matched++
				break
			}
		}
	}
	if matched > 0 {
		log.Printf("User rules matched %d/%d transactions", matched, len(txns))
	}
}

// aiCategorize fills in any blank Category fields by asking Ollama to bulk-
// classify the descriptions in a single prompt. Best-effort; failures leave
// the transactions uncategorized.
func (p *Pipeline) aiCategorize(txns []Transaction) {
	if len(txns) == 0 {
		return
	}
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://127.0.0.1:11434"
	}

	var sb strings.Builder
	for i, tx := range txns {
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
		"model":       p.ModelName,
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
		log.Printf("AI categorization: decode response: %v", err)
		return
	}

	valid := map[string]bool{
		"Food": true, "Transport": true, "Shopping": true, "Entertainment": true,
		"Utilities": true, "Housing": true, "Income": true, "Transfer": true,
		"Health": true, "Cash": true, "EMI": true, "Education": true, "Other": true,
	}

	updated := 0
	for _, line := range strings.Split(ollamaResp.Response, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ".", 2)
		if len(parts) != 2 {
			parts = strings.SplitN(line, ":", 2)
		}
		if len(parts) != 2 {
			continue
		}
		numStr := strings.TrimSpace(parts[0])
		cat := strings.TrimSpace(strings.Split(strings.TrimSpace(parts[1]), " ")[0])
		cat = strings.TrimRight(strings.Split(cat, "-")[0], ".,;:")
		idx := 0
		if _, err := fmt.Sscanf(numStr, "%d", &idx); err != nil || idx < 1 || idx > len(txns) {
			continue
		}
		if valid[cat] {
			txns[idx-1].Category = cat
			updated++
		}
	}
	log.Printf("AI categorized %d/%d transactions", updated, len(txns))
}

// extractTextForAI returns the file's raw text suitable for AI parsing.
func extractTextForAI(filePath, ext string) (string, error) {
	switch ext {
	case ".pdf":
		return ExtractPDFText(filePath)
	case ".csv", ".txt", ".tsv":
		b, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", ext, err)
		}
		return string(b), nil
	case ".xlsx", ".xls":
		return "", fmt.Errorf("excel binary format no longer supported — please convert to CSV before uploading")
	default:
		b, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("unsupported file type %q: %w", ext, err)
		}
		return string(b), nil
	}
}

// renderPDFToPNGs shells out to pdftoppm to render each PDF page as a 150 DPI
// PNG, then base64-encodes each page. Returns up to maxPDFPagesForVision pages.
func renderPDFToPNGs(filePath string) ([]string, error) {
	tmpDir, err := os.MkdirTemp("", "lf-pdfimg-")
	if err != nil {
		return nil, fmt.Errorf("render pdf: mkdir temp: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	prefix := filepath.Join(tmpDir, "page")
	cmd := exec.Command("pdftoppm", "-r", "150", "-png", filePath, prefix)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("render pdf: pdftoppm failed: %w (output: %s)", err, string(out))
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("render pdf: read tmp dir: %w", err)
	}

	var pngFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			pngFiles = append(pngFiles, filepath.Join(tmpDir, e.Name()))
		}
	}
	if len(pngFiles) == 0 {
		return nil, fmt.Errorf("render pdf: no pages produced")
	}
	sort.Slice(pngFiles, func(i, j int) bool {
		return pageNumFromFilename(pngFiles[i]) < pageNumFromFilename(pngFiles[j])
	})
	if len(pngFiles) > maxPDFPagesForVision {
		pngFiles = pngFiles[:maxPDFPagesForVision]
	}

	images := make([]string, 0, len(pngFiles))
	for _, p := range pngFiles {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("render pdf: read page %s: %w", p, err)
		}
		images = append(images, base64.StdEncoding.EncodeToString(data))
	}
	return images, nil
}

func pageNumFromFilename(name string) int {
	base := strings.TrimSuffix(filepath.Base(name), ".png")
	idx := strings.LastIndex(base, "-")
	if idx < 0 || idx == len(base)-1 {
		return 0
	}
	var n int
	if _, err := fmt.Sscanf(base[idx+1:], "%d", &n); err != nil {
		return 0
	}
	return n
}

// inferPeriod returns a "YYYY-MM" string from the first txn with a non-zero
// date, falling back to the current month (UTC).
func inferPeriod(txns []Transaction) string {
	for _, t := range txns {
		if !t.Date.IsZero() {
			d := t.Date.UTC()
			return fmt.Sprintf("%04d-%02d", d.Year(), int(d.Month()))
		}
	}
	now := time.Now().UTC()
	return fmt.Sprintf("%04d-%02d", now.Year(), int(now.Month()))
}
