package processors

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sagarjhaa/localfinance/services/logos/models"
)

// PDFProcessor handles PDF file processing
type PDFProcessor struct{}

// CanProcess returns true if this processor can handle the file type
func (p *PDFProcessor) CanProcess(fileType string) bool {
	return fileType == "pdf"
}

// GetSupportedTypes returns the file types this processor supports
func (p *PDFProcessor) GetSupportedTypes() []string {
	return []string{"pdf"}
}

// ValidateFormat validates that the file is a proper PDF
func (p *PDFProcessor) ValidateFormat(file io.Reader) error {
	// Read first few bytes to check PDF header
	buffer := make([]byte, 5)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file header: %w", err)
	}
	
	if n < 4 || string(buffer[:4]) != "%PDF" {
		return fmt.Errorf("not a valid PDF file")
	}
	
	return nil
}

// ProcessDocument processes a PDF file and extracts transactions
func (p *PDFProcessor) ProcessDocument(file io.Reader, filename string) ([]models.Transaction, error) {
	// For now, implement basic text extraction and parsing
	// In production, this would use a PDF library like pdfcpu or unipdf
	
	var transactions []models.Transaction
	
	// TODO: Implement actual PDF text extraction
	// For now, return a placeholder transaction to show the structure works
	placeholder := models.Transaction{
		Date:        time.Now(),
		Description: "PDF processing placeholder - implement with PDF library",
		Amount:      0.00,
		Category:    "Other",
		FileSource:  filename,
		CreatedAt:   time.Now(),
	}
	
	transactions = append(transactions, placeholder)
	
	return transactions, nil
}

// extractTextFromPDF would extract text from PDF using a PDF library
func (p *PDFProcessor) extractTextFromPDF(file io.Reader) (string, error) {
	// TODO: Implement with actual PDF library
	// This is a placeholder that reads the file as text (which won't work for real PDFs)
	
	scanner := bufio.NewScanner(file)
	var text strings.Builder
	
	for scanner.Scan() {
		text.WriteString(scanner.Text())
		text.WriteString("\n")
	}
	
	if err := scanner.Err(); err != nil {
		return "", err
	}
	
	return text.String(), nil
}

// parseTransactionsFromText extracts transactions from PDF text
func (p *PDFProcessor) parseTransactionsFromText(text, filename string) []models.Transaction {
	var transactions []models.Transaction
	
	// Common patterns for bank statements
	patterns := []string{
		// Pattern 1: Date Amount Description
		`(\d{2}/\d{2}/\d{4})\s+([-+]?\$?\d+\.\d{2})\s+(.+)`,
		// Pattern 2: Date Description Amount
		`(\d{2}/\d{2}/\d{4})\s+(.+?)\s+([-+]?\$?\d+\.\d{2})`,
		// Add more patterns as needed
	}
	
	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		
		matches := regex.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				transaction := p.parseTransactionMatch(match, filename)
				if transaction != nil {
					transactions = append(transactions, *transaction)
				}
			}
		}
	}
	
	return transactions
}

// parseTransactionMatch parses a regex match into a transaction
func (p *PDFProcessor) parseTransactionMatch(match []string, filename string) *models.Transaction {
	if len(match) < 4 {
		return nil
	}
	
	// Parse date
	dateStr := strings.TrimSpace(match[1])
	date, err := time.Parse("01/02/2006", dateStr)
	if err != nil {
		return nil
	}
	
	// Parse amount
	amountStr := strings.TrimSpace(match[2])
	amount, err := p.parseAmount(amountStr)
	if err != nil {
		return nil
	}
	
	// Get description
	description := strings.TrimSpace(match[3])
	if description == "" {
		return nil
	}
	
	return &models.Transaction{
		Date:        date,
		Description: description,
		Amount:      amount,
		FileSource:  filename,
		CreatedAt:   time.Now(),
	}
}

// parseAmount parses amount strings from PDF text
func (p *PDFProcessor) parseAmount(amountStr string) (float64, error) {
	// Clean the amount string
	cleaned := strings.TrimSpace(amountStr)
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.ReplaceAll(cleaned, "$", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "-")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.TrimSpace(cleaned)
	
	if cleaned == "" {
		return 0, fmt.Errorf("empty amount")
	}
	
	amount, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number format: %s", cleaned)
	}
	
	return amount, nil
}