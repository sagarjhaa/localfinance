package processors

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	pdflib "github.com/ledongthuc/pdf"
	"github.com/sagarjhaa/localfinance/services/logos/models"
)

// PDFProcessor handles PDF file processing
type PDFProcessor struct{}

func (p *PDFProcessor) CanProcess(fileType string) bool { return fileType == "pdf" }
func (p *PDFProcessor) GetSupportedTypes() []string     { return []string{"pdf"} }

func (p *PDFProcessor) ValidateFormat(file io.Reader) error {
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

// ProcessDocument extracts text from PDF and parses transactions
func (p *PDFProcessor) ProcessDocument(file io.Reader, filename string) ([]models.Transaction, error) {
	// ledongthuc/pdf needs a file path or ReaderAt+size, so write to temp if needed
	// The file parameter is actually from os.Open in main.go, so filename IS the path
	text, err := extractPDFText(filename)
	if err != nil {
		return nil, fmt.Errorf("PDF text extraction failed: %w", err)
	}

	transactions := ParseStatementText(text, filename)
	if len(transactions) == 0 {
		return nil, fmt.Errorf("no transactions found in PDF")
	}

	return transactions, nil
}

// extractPDFText extracts all text content from a PDF file.
// Tries the Go library first, falls back to pdftotext (poppler) for better extraction.
func extractPDFText(filePath string) (string, error) {
	// Try Go library first
	text, err := extractPDFTextGoLib(filePath)
	if err == nil && len(strings.TrimSpace(text)) > 20 {
		return text, nil
	}

	// Fall back to pdftotext with layout preservation (handles Chase, Capital One, etc.)
	layoutText, err := extractPDFTextPdftotext(filePath, true)
	if err == nil && len(strings.TrimSpace(layoutText)) > 20 {
		log.Printf("[PDF] Go library extraction empty/short, using pdftotext -layout (%d chars)", len(layoutText))
		return layoutText, nil
	}

	// Try pdftotext without layout (raw mode)
	rawText, err := extractPDFTextPdftotext(filePath, false)
	if err == nil && len(strings.TrimSpace(rawText)) > 20 {
		log.Printf("[PDF] Using pdftotext raw mode (%d chars)", len(rawText))
		return rawText, nil
	}

	// Return whatever we got (even if empty) with original error
	if len(strings.TrimSpace(text)) > 0 {
		return text, nil
	}
	return "", fmt.Errorf("failed to extract text from PDF (Go lib and pdftotext both failed)")
}

// extractPDFTextGoLib uses the ledongthuc/pdf Go library
func extractPDFTextGoLib(filePath string) (string, error) {
	f, r, err := pdflib.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var text strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
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

// extractPDFTextPdftotext uses the pdftotext CLI tool (poppler-utils)
func extractPDFTextPdftotext(filePath string, layout bool) (string, error) {
	args := []string{}
	if layout {
		args = append(args, "-layout")
	}
	args = append(args, filePath, "-")

	cmd := exec.Command("pdftotext", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext failed: %w", err)
	}
	return string(out), nil
}

// Date patterns for lines with proper spacing
var datePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^(\d{1,2}[/\-]\d{1,2}[/\-]\d{2,4})`),
	regexp.MustCompile(`^(\d{1,2}\s+[A-Za-z]{3}\s+\d{2,4})`),
	regexp.MustCompile(`^(\d{4}-\d{1,2}-\d{1,2})`),
	regexp.MustCompile(`^([A-Za-z]{3}\s+\d{1,2},?\s+\d{4})`),
}

// Short date pattern: MM/DD without year (Chase, Capital One)
var shortDatePattern = regexp.MustCompile(`^\s*(\d{2}/\d{2})\s*$`)
var shortDateWithDescPattern = regexp.MustCompile(`^\s*(\d{2}/\d{2})\s+(.+)`)

// Column-based line pattern: MM/DD ...spaces... description ...spaces... amount
var columnLinePattern = regexp.MustCompile(`^\s*(\d{2}/\d{2})\s+(.+?)\s{2,}(-?[\d,]+\.\d{2})\s*$`)

// Standalone amount pattern (no currency symbol required)
var bareAmountPattern = regexp.MustCompile(`^\s*(-?[\d,]+\.\d{2})\s*$`)

// Trailing amount in a concatenated line (description immediately followed by amount)
var trailingAmtPattern = regexp.MustCompile(`(.+?)(-?[\d,]+\.\d{2})\s*$`)

// Year inference: look for date ranges like "02/16/26 - 03/15/26" or "02/16/2026 - 03/15/2026"
var yearInferencePattern = regexp.MustCompile(`(\d{2}/\d{2}/(\d{2,4}))\s*[-–—]\s*(\d{2}/\d{2}/(\d{2,4}))`)

// Inline date pattern for concatenated text (e.g., Amex: "01/05/21DESCRIPTION$10.87")
var inlineDateAmountRegex = regexp.MustCompile(`(\d{2}/\d{2}/\d{2,4})([A-Za-z*\s].+?)[-]?\$(\d[\d,]*\.\d{2})`)

// Amount pattern
var amountRegex = regexp.MustCompile(`[-]?\s*[₹$€£]?\s*[\d,]+\.\d{2}`)

// ParseStatementText parses extracted PDF text into transactions.
// Handles well-formatted PDFs, column-based PDFs (Chase/Capital One), and concatenated text (Amex).
func ParseStatementText(text, filename string) []models.Transaction {
	// First try line-by-line parsing (for well-formatted PDFs with full dates)
	transactions := parseLineByLine(text, filename)

	// Try column-based parsing for Chase/Capital One (MM/DD with amounts, no $ prefix)
	if len(transactions) < 2 {
		columnTxns := parseColumnBased(text, filename)
		if len(columnTxns) > len(transactions) {
			transactions = columnTxns
		}
	}

	// Try multi-line parsing for Go PDF lib output (date, desc, amount on separate lines)
	if len(transactions) < 2 {
		multiLineTxns := parseMultiLine(text, filename)
		if len(multiLineTxns) > len(transactions) {
			transactions = multiLineTxns
		}
	}

	// If that didn't find much, try inline pattern matching (for concatenated text like Amex)
	if len(transactions) < 2 {
		inlineTxns := parseInlineTransactions(text, filename)
		if len(inlineTxns) > len(transactions) {
			transactions = inlineTxns
		}
	}

	return transactions
}

// inferYear looks for a date range in the statement text (e.g. "02/16/26 - 03/15/26")
// and extracts the year. Falls back to current year.
func inferYear(text string) int {
	matches := yearInferencePattern.FindStringSubmatch(text)
	if len(matches) >= 5 {
		// Use the closing date year (last one)
		yearStr := matches[4]
		year, err := strconv.Atoi(yearStr)
		if err == nil {
			if year < 100 {
				year += 2000
			}
			return year
		}
	}
	return time.Now().Year()
}

// parseShortDate parses MM/DD with an inferred year into time.Time
func parseShortDate(dateStr string, year int) time.Time {
	parts := strings.Split(dateStr, "/")
	if len(parts) != 2 {
		return time.Now()
	}
	month, err1 := strconv.Atoi(parts[0])
	day, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Now()
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// parseColumnBased handles column-based PDFs like Chase/Capital One where
// lines look like: "03/12                    AUTOMATIC PAYMENT - THANK YOU                -1,323.73"
func parseColumnBased(text, filename string) []models.Transaction {
	year := inferYear(text)
	lines := strings.Split(text, "\n")
	var transactions []models.Transaction

	for _, line := range lines {
		match := columnLinePattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		dateStr := match[1]
		description := strings.TrimSpace(match[2])
		amountStr := match[3]

		if len(description) < 2 {
			continue
		}

		// Clean amount
		amountStr = strings.ReplaceAll(amountStr, ",", "")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount == 0 {
			continue
		}

		parsedDate := parseShortDate(dateStr, year)

		// Determine type based on sign
		txnType := "credit"
		if amount < 0 {
			txnType = "debit"
		}

		transactions = append(transactions, models.Transaction{
			Date:        parsedDate,
			Description: description,
			Amount:      amount,
			Type:        txnType,
			FileSource:  filename,
			CreatedAt:   time.Now(),
		})
	}

	return transactions
}

// parseMultiLine handles the Go PDF library's output where Chase transactions
// appear as separate lines:
//
//	02/16
//	APPLE.COM/BILL 866-712-7753 CA
//	18.99
func parseMultiLine(text, filename string) []models.Transaction {
	year := inferYear(text)
	lines := strings.Split(text, "\n")
	var transactions []models.Transaction

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Check if this line is a short date only (MM/DD)
		if !shortDatePattern.MatchString(line) {
			continue
		}
		dateStr := strings.TrimSpace(line)

		// Look ahead for description and amount in next 1-2 lines
		description := ""
		amountStr := ""

		if i+1 >= len(lines) {
			continue
		}

		nextLine := strings.TrimSpace(lines[i+1])

		// Case 1: next line has both description and amount (concatenated)
		// e.g. "APPLE.COM/BILL 866-712-7753 CA18.99"
		// Try to split trailing amount from description
		if true {
			if m := trailingAmtPattern.FindStringSubmatch(nextLine); m != nil && len(m[1]) >= 3 {
				description = strings.TrimSpace(m[1])
				amountStr = m[2]
				i += 1
			}
		}

		// Case 2: next line is description, line after is amount
		if amountStr == "" && i+2 < len(lines) {
			descLine := strings.TrimSpace(lines[i+1])
			amtLine := strings.TrimSpace(lines[i+2])
			if bareAmountPattern.MatchString(amtLine) && len(descLine) >= 3 && !bareAmountPattern.MatchString(descLine) && !shortDatePattern.MatchString(descLine) {
				description = descLine
				amountStr = strings.TrimSpace(amtLine)
				i += 2
			}
		}

		if description == "" || amountStr == "" {
			continue
		}

		// Clean amount
		amountStr = strings.ReplaceAll(amountStr, ",", "")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount == 0 {
			continue
		}

		parsedDate := parseShortDate(dateStr, year)

		txnType := "credit"
		if amount < 0 {
			txnType = "debit"
		}

		transactions = append(transactions, models.Transaction{
			Date:        parsedDate,
			Description: description,
			Amount:      amount,
			Type:        txnType,
			FileSource:  filename,
			CreatedAt:   time.Now(),
		})
	}

	return transactions
}

// parseInlineTransactions handles concatenated PDF text like Amex statements
// where text is squished: "01/05/21AMZNMKTPUS*9T57777M3AMZN.COM/BILLWABOOKSTORES$10.87"
func parseInlineTransactions(text, filename string) []models.Transaction {
	var transactions []models.Transaction

	matches := inlineDateAmountRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		dateStr := match[1]
		description := match[2]
		amountStr := match[3]

		// Clean description
		description = strings.TrimSpace(description)
		// Try to make it more readable: split on common patterns
		description = cleanDescription(description)
		if len(description) < 3 {
			continue
		}

		// Parse amount
		amountStr = strings.ReplaceAll(amountStr, ",", "")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount == 0 {
			continue
		}

		// Check if this looks like a payment/credit (negative in statement context)
		descLower := strings.ToLower(description)
		if strings.Contains(descLower, "payment") || strings.Contains(descLower, "credit") || strings.Contains(descLower, "refund") {
			amount = -amount // payments are credits (reduce balance)
		}

		parsedDate := normalizeDate(dateStr)

		transactions = append(transactions, models.Transaction{
			Date:        parsedDate,
			Description: description,
			Amount:      amount,
			FileSource:  filename,
			CreatedAt:   time.Now(),
		})
	}

	return transactions
}

// cleanDescription attempts to make concatenated descriptions more readable
func cleanDescription(desc string) string {
	// Remove common suffixes that are store categories
	suffixes := []string{"BOOKSTORES", "MERCHANDISE", "RESTAURANTS", "GROCERY", "GAS STATIONS",
		"SUPPLIES", "SERVICES", "ELECTRONICS", "DEPARTMENT STORES"}
	upper := strings.ToUpper(desc)
	for _, suf := range suffixes {
		if idx := strings.LastIndex(upper, suf); idx > 5 {
			desc = strings.TrimSpace(desc[:idx])
			break
		}
	}

	// Remove URL-like patterns
	desc = regexp.MustCompile(`[A-Z]+\.COM/[A-Z]+`).ReplaceAllString(desc, "")

	// Add spaces before uppercase runs after lowercase (crude camelCase split)
	// Skip this for all-caps text
	desc = strings.TrimSpace(desc)
	if len(desc) > 3 {
		// Remove trailing special chars
		desc = strings.TrimRight(desc, "*#/ ")
	}

	return desc
}

// parseLineByLine handles well-formatted PDFs with proper line breaks
func parseLineByLine(text, filename string) []models.Transaction {
	year := inferYear(text)
	lines := strings.Split(text, "\n")
	var transactions []models.Transaction

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if len(line) < 5 {
			continue
		}

		var dateStr string
		var dateEndIdx int
		isShortDate := false
		for _, pat := range datePatterns {
			m := pat.FindStringIndex(line)
			if m != nil {
				dateStr = line[m[0]:m[1]]
				dateEndIdx = m[1]
				break
			}
		}
		// Try short date with description (MM/DD followed by text)
		if dateStr == "" {
			if m := shortDateWithDescPattern.FindStringSubmatchIndex(line); m != nil {
				dateStr = line[m[2]:m[3]] // group 1: the date
				dateEndIdx = m[3]
				isShortDate = true
			}
		}
		if dateStr == "" {
			continue
		}

		rest := strings.TrimSpace(line[dateEndIdx:])

		extraDesc := ""
		if i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			hasDate := shortDateWithDescPattern.MatchString(nextLine)
			if !hasDate {
				for _, pat := range datePatterns {
					if pat.MatchString(nextLine) { hasDate = true; break }
				}
			}
			if !hasDate && !amountRegex.MatchString(nextLine) && len(nextLine) > 2 {
				extraDesc = " " + nextLine
			}
		}

		amountMatches := amountRegex.FindAllString(rest, -1)
		if len(amountMatches) == 0 { continue }

		description := rest
		firstAmtIdx := strings.Index(rest, amountMatches[0])
		if firstAmtIdx > 0 { description = rest[:firstAmtIdx] }
		description = strings.TrimSpace(description + extraDesc)
		description = strings.Join(strings.Fields(description), " ")
		if len(description) < 2 { description = "Transaction on " + dateStr }

		var amounts []float64
		for _, amtStr := range amountMatches {
			cleaned := strings.Map(func(r rune) rune {
				if r == '₹' || r == '$' || r == '€' || r == '£' || r == ',' || r == ' ' { return -1 }
				return r
			}, amtStr)
			val, err := strconv.ParseFloat(cleaned, 64)
			if err == nil { amounts = append(amounts, val) }
		}
		if len(amounts) == 0 { continue }

		var amount float64
		switch len(amounts) {
		case 1:
			amount = amounts[0]
		case 2:
			if amounts[0] != 0 && amounts[1] != 0 {
				amount = -amounts[0]
			} else if amounts[0] != 0 {
				amount = -amounts[0]
			} else {
				amount = amounts[1]
			}
		default:
			if amounts[1] > 0 && amounts[0] == 0 { amount = amounts[1] } else if amounts[0] > 0 { amount = -amounts[0] } else { amount = amounts[0] }
		}
		if amount == 0 { continue }

		var parsedDate time.Time
		if isShortDate {
			parsedDate = parseShortDate(dateStr, year)
		} else {
			parsedDate = normalizeDate(dateStr)
		}
		txnType := "credit"
		if amount < 0 { txnType = "debit" }

		transactions = append(transactions, models.Transaction{
			Date: parsedDate, Description: description, Amount: amount,
			Type: txnType, FileSource: filename, CreatedAt: time.Now(),
		})
	}
	return transactions
}

// normalizeDate parses various date formats into time.Time
func normalizeDate(s string) time.Time {
	s = strings.TrimSpace(s)

	formats := []string{
		"2006-01-02",
		"02/01/2006", "2/1/2006",
		"02-01-2006", "2-1-2006",
		"02/01/06", "2/1/06",
		"02-01-06",
		"01/02/2006", // US format fallback
		"Jan 2, 2006", "Jan 02, 2006",
		"2 Jan 2006", "02 Jan 2006",
		"2 Jan 06", "02 Jan 06",
	}

	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}

	return time.Now()
}
