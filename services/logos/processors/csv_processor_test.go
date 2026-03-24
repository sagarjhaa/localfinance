package processors

import (
	"strings"
	"testing"
)

func TestCSVProcessorCanProcess(t *testing.T) {
	p := &CSVProcessor{}
	if !p.CanProcess("csv") {
		t.Fatal("Should process csv")
	}
	if p.CanProcess("pdf") {
		t.Fatal("Should not process pdf")
	}
}

func TestCSVProcessorGetSupportedTypes(t *testing.T) {
	p := &CSVProcessor{}
	types := p.GetSupportedTypes()
	if len(types) != 1 || types[0] != "csv" {
		t.Fatalf("Expected [csv], got %v", types)
	}
}

func TestCSVProcessorBasicParsing(t *testing.T) {
	csv := `Date,Description,Amount,Category
2026-03-01,Salary Credit,85000.00,Income
2026-03-02,Swiggy Food,-450.00,Food
2026-03-03,Amazon Purchase,-2499.00,Shopping`

	p := &CSVProcessor{}
	txns, err := p.ProcessDocument(strings.NewReader(csv), "test.csv")
	if err != nil {
		t.Fatalf("ProcessDocument failed: %v", err)
	}
	if len(txns) != 3 {
		t.Fatalf("Expected 3 transactions, got %d", len(txns))
	}
	if txns[0].Amount != 85000 {
		t.Fatalf("Expected amount 85000, got %f", txns[0].Amount)
	}
	if txns[0].Category != "Income" {
		t.Fatalf("Expected category Income, got %s", txns[0].Category)
	}
	if txns[1].Amount != -450 {
		t.Fatalf("Expected amount -450, got %f", txns[1].Amount)
	}
}

func TestCSVProcessorAmountFormats(t *testing.T) {
	csv := `Date,Description,Amount
2026-03-01,Test One,"1,234.56"
2026-03-02,Test Two,$500.00
2026-03-03,Test Three,(100.00)`

	p := &CSVProcessor{}
	txns, err := p.ProcessDocument(strings.NewReader(csv), "test.csv")
	if err != nil {
		t.Fatalf("ProcessDocument failed: %v", err)
	}
	if len(txns) != 3 {
		t.Fatalf("Expected 3 transactions, got %d", len(txns))
	}
	if txns[0].Amount != 1234.56 {
		t.Fatalf("Comma format: expected 1234.56, got %f", txns[0].Amount)
	}
	if txns[1].Amount != 500.00 {
		t.Fatalf("Dollar sign: expected 500, got %f", txns[1].Amount)
	}
	if txns[2].Amount != -100.00 {
		t.Fatalf("Parentheses: expected -100, got %f", txns[2].Amount)
	}
}

func TestCSVProcessorDateFormats(t *testing.T) {
	csv := `Date,Description,Amount
2026-03-01,ISO Date,100.00
01/02/2026,US Date,200.00
02-Jan-2026,Text Date,300.00`

	p := &CSVProcessor{}
	txns, err := p.ProcessDocument(strings.NewReader(csv), "test.csv")
	if err != nil {
		t.Fatalf("ProcessDocument failed: %v", err)
	}
	if len(txns) < 2 {
		t.Fatalf("Should parse at least 2 transactions, got %d", len(txns))
	}
}

func TestCSVProcessorEmptyFile(t *testing.T) {
	p := &CSVProcessor{}
	_, err := p.ProcessDocument(strings.NewReader(""), "test.csv")
	if err == nil {
		t.Fatal("Should error on empty file")
	}
}

func TestCSVProcessorHeaderOnly(t *testing.T) {
	p := &CSVProcessor{}
	txns, err := p.ProcessDocument(strings.NewReader("Date,Description,Amount\n"), "test.csv")
	if err != nil {
		t.Fatalf("Should not error on header-only: %v", err)
	}
	if len(txns) != 0 {
		t.Fatalf("Expected 0 transactions, got %d", len(txns))
	}
}

func TestCSVProcessorColumnDetection(t *testing.T) {
	// Test with non-standard column names
	csv := `Transaction Date,Memo,Transaction Amount
2026-03-01,Coffee Shop,-5.50`

	p := &CSVProcessor{}
	txns, err := p.ProcessDocument(strings.NewReader(csv), "test.csv")
	if err != nil {
		t.Fatalf("ProcessDocument failed: %v", err)
	}
	if len(txns) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(txns))
	}
}
