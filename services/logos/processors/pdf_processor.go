package processors

import (
	"fmt"
	"io"
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

// extractPDFText extracts all text content from a PDF file
func extractPDFText(filePath string) (string, error) {
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

// Date patterns for lines with proper spacing
var datePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^(\d{1,2}[/\-]\d{1,2}[/\-]\d{2,4})`),
	regexp.MustCompile(`^(\d{1,2}\s+[A-Za-z]{3}\s+\d{2,4})`),
	regexp.MustCompile(`^(\d{4}-\d{1,2}-\d{1,2})`),
	regexp.MustCompile(`^([A-Za-z]{3}\s+\d{1,2},?\s+\d{4})`),
}

// Inline date pattern for concatenated text (e.g., Amex: "01/05/21DESCRIPTION$10.87")
var inlineDateAmountRegex = regexp.MustCompile(`(\d{2}/\d{2}/\d{2,4})([A-Za-z*\s].+?)[-]?\$(\d[\d,]*\.\d{2})`)

// Amount pattern
var amountRegex = regexp.MustCompile(`[-]?\s*[₹$€£]?\s*[\d,]+\.\d{2}`)

// ParseStatementText parses extracted PDF text into transactions.
// Handles both well-formatted PDFs and concatenated text (like Amex statements).
func ParseStatementText(text, filename string) []models.Transaction {
	// First try line-by-line parsing (for well-formatted PDFs)
	transactions := parseLineByLine(text, filename)

	// If that didn't find much, try inline pattern matching (for concatenated text like Amex)
	if len(transactions) < 2 {
		inlineTxns := parseInlineTransactions(text, filename)
		if len(inlineTxns) > len(transactions) {
			transactions = inlineTxns
		}
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
	lines := strings.Split(text, "\n")
	var transactions []models.Transaction

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if len(line) < 5 {
			continue
		}

		var dateStr string
		var dateEndIdx int
		for _, pat := range datePatterns {
			m := pat.FindStringIndex(line)
			if m != nil {
				dateStr = line[m[0]:m[1]]
				dateEndIdx = m[1]
				break
			}
		}
		if dateStr == "" {
			continue
		}

		rest := strings.TrimSpace(line[dateEndIdx:])

		extraDesc := ""
		if i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			hasDate := false
			for _, pat := range datePatterns {
				if pat.MatchString(nextLine) { hasDate = true; break }
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

		parsedDate := normalizeDate(dateStr)
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
