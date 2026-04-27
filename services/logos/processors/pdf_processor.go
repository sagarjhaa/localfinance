package processors

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	pdflib "github.com/ledongthuc/pdf"
)

// ExtractPDFText extracts all text content from a PDF file.
// Tries the Go library first, falls back to pdftotext (poppler) for better extraction.
//
// Logos no longer parses transactions itself — extracted text is forwarded to
// Sophia's /api/v1/parse endpoint for AI-driven extraction.
func ExtractPDFText(filePath string) (string, error) {
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

// normalizeDate parses various date formats into time.Time.
// Retained because detector.go uses it for statement period detection.
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
