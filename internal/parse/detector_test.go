package parse

import (
	"testing"
)

func TestDetectAccountTypeCreditCard(t *testing.T) {
	text := "Your Credit Card Statement. Minimum Payment Due: $25.00. Credit Limit: $5,000. Payment Due Date: 02/15/2026."
	meta := DetectStatementInfo(text)
	if meta.AccountType != "credit_card" {
		t.Fatalf("Expected credit_card, got %s", meta.AccountType)
	}
}

func TestDetectAccountTypeChecking(t *testing.T) {
	text := "Checking Account Statement. Opening Balance: 50,000.00. Closing Balance: 45,000.00. Withdrawal and Deposit details below."
	meta := DetectStatementInfo(text)
	if meta.AccountType != "checking" {
		t.Fatalf("Expected checking, got %s", meta.AccountType)
	}
}

func TestDetectAccountTypeSavings(t *testing.T) {
	text := "Savings Account Statement. Interest Earned: 125.50. Interest Rate: 3.5% p.a."
	meta := DetectStatementInfo(text)
	if meta.AccountType != "savings" {
		t.Fatalf("Expected savings, got %s", meta.AccountType)
	}
}

func TestDetectInstitutionAmex(t *testing.T) {
	text := "American Express Card Member Statement. Account Ending 0-11007."
	meta := DetectStatementInfo(text)
	if meta.Institution != "American Express" {
		t.Fatalf("Expected American Express, got %s", meta.Institution)
	}
}

func TestDetectInstitutionHDFC(t *testing.T) {
	text := "HDFC Bank Statement of Account. A/C No: XXXX1234."
	meta := DetectStatementInfo(text)
	if meta.Institution != "HDFC Bank" {
		t.Fatalf("Expected HDFC Bank, got %s", meta.Institution)
	}
}

func TestDetectInstitutionSBI(t *testing.T) {
	text := "State Bank of India. Account Statement for the period."
	meta := DetectStatementInfo(text)
	if meta.Institution != "State Bank of India" {
		t.Fatalf("Expected State Bank of India, got %s", meta.Institution)
	}
}

func TestDetectInstitutionICICI(t *testing.T) {
	text := "ICICI Bank Account Statement. From 01-Mar-2026 To 31-Mar-2026."
	meta := DetectStatementInfo(text)
	if meta.Institution != "ICICI Bank" {
		t.Fatalf("Expected ICICI Bank, got %s", meta.Institution)
	}
}

func TestDetectAccountNumber(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"Account Ending 0-11007", "XXXX11007"},
		{"Card No. XXXX-1234", "XXXX1234"},
		{"A/C No: XXXX5678", "XXXX5678"},
		{"Account number: XXXXXXXX9012", "XXXX9012"},
		{"No account info here", ""},
	}

	for _, tt := range tests {
		meta := DetectStatementInfo(tt.text)
		if meta.AccountNumber != tt.want {
			preview := tt.text
			if len(preview) > 30 { preview = preview[:30] }
			t.Errorf("Text %q: expected account %q, got %q", preview, tt.want, meta.AccountNumber)
		}
	}
}

func TestDetectPeriod(t *testing.T) {
	text := "Statement Period: 01/05/2026 to 02/03/2026"
	meta := DetectStatementInfo(text)
	if meta.PeriodStart == nil || meta.PeriodEnd == nil {
		t.Fatal("Should detect period start and end")
	}
}

func TestDetectPaymentDue(t *testing.T) {
	text := "Payment Due Date: 02/15/2026. Minimum Payment: $25.00"
	meta := DetectStatementInfo(text)
	if meta.PaymentDueDate == nil {
		t.Fatal("Should detect payment due date")
	}
	if meta.PaymentDueDay == 0 {
		t.Fatal("Should detect payment due day")
	}
}

func TestDetectAmexStatement(t *testing.T) {
	// Simulated Amex concatenated text
	text := `AmericanExpressCardMemberStatementAccountEnding0-11007
ClosingDate02/03/21PaymentDueDatebyFebruary28,2021
CreditCard minimum payment due $25.00 credit limit available
01/05/21AMZNMKTPUS*9T57777M3BOOKSTORES$10.87
01/06/21AMAZON.COM*B97HS3473MERCHANDISE$75.18`

	meta := DetectStatementInfo(text)
	if meta.AccountType != "credit_card" {
		t.Fatalf("Expected credit_card, got %s", meta.AccountType)
	}
	if meta.Institution != "American Express" {
		t.Fatalf("Expected American Express, got %s", meta.Institution)
	}
}

func TestDetectEmpty(t *testing.T) {
	meta := DetectStatementInfo("")
	if meta.AccountType != "checking" {
		t.Fatalf("Empty text should default to checking, got %s", meta.AccountType)
	}
	if meta.Institution != "" {
		t.Fatalf("Empty text should have no institution, got %s", meta.Institution)
	}
}
