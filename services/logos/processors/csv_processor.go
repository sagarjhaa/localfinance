package processors

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/sagarjhaa/localfinance/services/logos/models"
)

// CSVProcessor handles CSV file processing
type CSVProcessor struct{}

// CanProcess returns true if this processor can handle the file type
func (p *CSVProcessor) CanProcess(fileType string) bool {
	return fileType == "csv"
}

// GetSupportedTypes returns the file types this processor supports
func (p *CSVProcessor) GetSupportedTypes() []string {
	return []string{"csv"}
}

// ValidateFormat validates that the file is a proper CSV
func (p *CSVProcessor) ValidateFormat(file io.Reader) error {
	reader := csv.NewReader(file)
	
	// Try to read the first line to validate CSV format
	_, err := reader.Read()
	if err != nil && err != io.EOF {
		return fmt.Errorf("invalid CSV format: %w", err)
	}
	
	return nil
}

// ProcessDocument processes a CSV file and extracts transactions
func (p *CSVProcessor) ProcessDocument(file io.Reader, filename string) ([]models.Transaction, error) {
	reader := csv.NewReader(file)
	
	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}
	
	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV file")
	}
	
	// Auto-detect column mapping
	columnMap, err := p.detectColumns(records[0])
	if err != nil {
		return nil, fmt.Errorf("failed to detect CSV columns: %w", err)
	}
	
	var transactions []models.Transaction
	
	// Process each record (skip header)
	for i, record := range records[1:] {
		if len(record) < len(columnMap) {
			continue // Skip incomplete records
		}
		
		transaction, err := p.parseTransaction(record, columnMap, filename)
		if err != nil {
			// Log error but continue processing other transactions
			fmt.Printf("Error parsing transaction at line %d: %v\n", i+2, err)
			continue
		}
		
		transactions = append(transactions, transaction)
	}
	
	return transactions, nil
}

// detectColumns attempts to auto-detect CSV column mapping
func (p *CSVProcessor) detectColumns(header []string) (map[string]int, error) {
	columnMap := make(map[string]int)
	
	for i, col := range header {
		colLower := strings.ToLower(strings.TrimSpace(col))
		
		// Date column detection
		if p.isDateColumn(colLower) && columnMap["date"] == 0 {
			columnMap["date"] = i
		}
		
		// Description column detection
		if p.isDescriptionColumn(colLower) && columnMap["description"] == 0 {
			columnMap["description"] = i
		}
		
		// Amount column detection
		if p.isAmountColumn(colLower) && columnMap["amount"] == 0 {
			columnMap["amount"] = i
		}
		
		// Category column detection (optional)
		if p.isCategoryColumn(colLower) {
			columnMap["category"] = i
		}
	}
	
	// Verify required columns are found
	if _, exists := columnMap["date"]; !exists {
		return nil, fmt.Errorf("date column not found")
	}
	if _, exists := columnMap["description"]; !exists {
		return nil, fmt.Errorf("description column not found")
	}
	if _, exists := columnMap["amount"]; !exists {
		return nil, fmt.Errorf("amount column not found")
	}
	
	return columnMap, nil
}

// isDateColumn checks if a column name indicates it contains dates
func (p *CSVProcessor) isDateColumn(col string) bool {
	dateKeywords := []string{"date", "transaction date", "posted date", "effective date", "time"}
	for _, keyword := range dateKeywords {
		if strings.Contains(col, keyword) {
			return true
		}
	}
	return false
}

// isDescriptionColumn checks if a column name indicates it contains descriptions
func (p *CSVProcessor) isDescriptionColumn(col string) bool {
	descKeywords := []string{"description", "memo", "payee", "merchant", "details", "transaction details"}
	for _, keyword := range descKeywords {
		if strings.Contains(col, keyword) {
			return true
		}
	}
	return false
}

// isAmountColumn checks if a column name indicates it contains amounts
func (p *CSVProcessor) isAmountColumn(col string) bool {
	amountKeywords := []string{"amount", "debit", "credit", "transaction amount", "balance", "withdrawal", "deposit"}
	for _, keyword := range amountKeywords {
		if strings.Contains(col, keyword) {
			return true
		}
	}
	return false
}

// isCategoryColumn checks if a column name indicates it contains categories
func (p *CSVProcessor) isCategoryColumn(col string) bool {
	categoryKeywords := []string{"category", "type", "classification"}
	for _, keyword := range categoryKeywords {
		if strings.Contains(col, keyword) {
			return true
		}
	}
	return false
}

// parseTransaction parses a CSV record into a transaction
func (p *CSVProcessor) parseTransaction(record []string, columnMap map[string]int, filename string) (models.Transaction, error) {
	var transaction models.Transaction
	
	// Parse date
	dateStr := strings.TrimSpace(record[columnMap["date"]])
	date, err := p.parseDate(dateStr)
	if err != nil {
		return transaction, fmt.Errorf("invalid date format: %s", dateStr)
	}
	transaction.Date = date
	
	// Parse description
	transaction.Description = strings.TrimSpace(record[columnMap["description"]])
	if transaction.Description == "" {
		return transaction, fmt.Errorf("empty description")
	}
	
	// Parse amount
	amountStr := strings.TrimSpace(record[columnMap["amount"]])
	amount, err := p.parseAmount(amountStr)
	if err != nil {
		return transaction, fmt.Errorf("invalid amount format: %s", amountStr)
	}
	transaction.Amount = amount
	
	// Parse category (optional)
	if categoryIndex, exists := columnMap["category"]; exists && categoryIndex < len(record) {
		transaction.Category = strings.TrimSpace(record[categoryIndex])
	}
	
	// Set file source
	transaction.FileSource = filename
	transaction.CreatedAt = time.Now()
	
	return transaction, nil
}

// parseDate attempts to parse various date formats
func (p *CSVProcessor) parseDate(dateStr string) (time.Time, error) {
	// Common date formats
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"01-02-2006",
		"2006/01/02",
		"Jan 2, 2006",
		"January 2, 2006",
		"2-Jan-2006",
		"02-Jan-2006",
	}
	
	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// parseAmount parses amount strings, handling various formats
func (p *CSVProcessor) parseAmount(amountStr string) (float64, error) {
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