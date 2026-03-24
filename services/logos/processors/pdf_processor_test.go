package processors

import (
	"testing"
	"time"
)

func TestPDFProcessorCanProcess(t *testing.T) {
	p := &PDFProcessor{}
	if !p.CanProcess("pdf") {
		t.Fatal("Should process pdf")
	}
	if p.CanProcess("csv") {
		t.Fatal("Should not process csv")
	}
}

func TestParseStatementTextSBIStyle(t *testing.T) {
	text := `Statement of Account
Account No: XXXX1234

01/03/2026    SALARY CREDIT MAR 2026                           85,000.00
03/03/2026    UPI/DR/SWIGGY                           450.00
05/03/2026    ATM WDL/CASH                          5,000.00
`

	txns := ParseStatementText(text, "test.pdf")
	if len(txns) < 2 {
		t.Fatalf("Expected at least 2 transactions, got %d", len(txns))
	}

	// Check that we found a salary-like transaction
	found := false
	for _, txn := range txns {
		if txn.Amount > 10000 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Should find a large-amount transaction (salary)")
	}
}

func TestParseStatementTextISO(t *testing.T) {
	text := `2026-03-01  Payment Received  5,000.00
2026-03-02  Grocery Store  1,250.50`

	txns := ParseStatementText(text, "test.pdf")
	if len(txns) < 1 {
		t.Fatalf("Expected at least 1 transaction, got %d", len(txns))
	}
}

func TestParseStatementTextEmpty(t *testing.T) {
	txns := ParseStatementText("", "test.pdf")
	if len(txns) != 0 {
		t.Fatalf("Expected 0 transactions from empty text, got %d", len(txns))
	}
}

func TestParseStatementTextNoTransactions(t *testing.T) {
	text := `This is just some random text
with no dates or amounts
nothing to see here`

	txns := ParseStatementText(text, "test.pdf")
	if len(txns) != 0 {
		t.Fatalf("Expected 0 transactions, got %d", len(txns))
	}
}

func TestParseStatementTextMultipleAmounts(t *testing.T) {
	// Debit | Credit | Balance format
	text := `01/03/2026   SALARY               0.00      85,000.00    85,000.00
02/03/2026   GROCERY             450.00          0.00    84,550.00`

	txns := ParseStatementText(text, "test.pdf")
	if len(txns) < 1 {
		t.Fatalf("Expected at least 1 transaction, got %d", len(txns))
	}
}

func TestNormalizeDateFormats(t *testing.T) {
	tests := []struct {
		input    string
		wantYear int
	}{
		{"2026-03-01", 2026},
		{"01/03/2026", 2026},
		{"01-03-2026", 2026},
		{"01 Mar 2026", 2026},
		{"1 Jan 2026", 2026},
	}

	for _, tt := range tests {
		d := normalizeDate(tt.input)
		if d.Year() != tt.wantYear {
			t.Errorf("normalizeDate(%q) year = %d, want %d", tt.input, d.Year(), tt.wantYear)
		}
	}
}

func TestNormalizeDateInvalid(t *testing.T) {
	d := normalizeDate("garbage")
	// Should fallback to current time
	if time.Since(d) > time.Hour {
		t.Fatal("Invalid date should return approximate current time")
	}
}

func TestNormalizeDateEmpty(t *testing.T) {
	d := normalizeDate("")
	if time.Since(d) > time.Hour {
		t.Fatal("Empty date should return approximate current time")
	}
}
