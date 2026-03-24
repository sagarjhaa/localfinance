package processors

import (
	"regexp"
	"strings"
	"time"
)

// StatementMetadata holds detected information about a financial statement
type StatementMetadata struct {
	AccountType     string     `json:"account_type"`      // credit_card, checking, savings
	Institution     string     `json:"institution"`       // Amex, HDFC, SBI, etc.
	AccountNumber   string     `json:"account_number"`    // masked, e.g. "XXXX1007"
	PeriodStart     *time.Time `json:"period_start"`
	PeriodEnd       *time.Time `json:"period_end"`
	PaymentDueDate  *time.Time `json:"payment_due_date"`
	PaymentDueDay   int        `json:"payment_due_day"`   // day of month
	CreditLimit     float64    `json:"credit_limit"`
	MinPaymentDue   float64    `json:"min_payment_due"`
	OpeningBalance  float64    `json:"opening_balance"`
	ClosingBalance  float64    `json:"closing_balance"`
	BillingCycleDay int        `json:"billing_cycle_day"` // statement closing day of month
}

// DetectStatementInfo scans extracted text and detects statement metadata
func DetectStatementInfo(text string) StatementMetadata {
	meta := StatementMetadata{}
	lower := strings.ToLower(text)

	meta.AccountType = detectAccountType(lower)
	meta.Institution = detectInstitution(lower, text)
	meta.AccountNumber = detectAccountNumber(text)
	meta.PeriodStart, meta.PeriodEnd = detectPeriod(text)
	meta.PaymentDueDate, meta.PaymentDueDay = detectPaymentDue(text)
	meta.CreditLimit = detectAmount(text, `(?i)credit\s*limit[:\s]*\$?([\d,]+\.?\d*)`)
	meta.MinPaymentDue = detectAmount(text, `(?i)minimum\s*payment[:\s]*\$?([\d,]+\.?\d*)`)
	meta.OpeningBalance = detectAmount(text, `(?i)(?:opening|previous|beginning)\s*balance[:\s]*\$?([\d,]+\.?\d*)`)
	meta.ClosingBalance = detectAmount(text, `(?i)(?:closing|new|ending|statement)\s*balance[:\s]*\$?([\d,]+\.?\d*)`)

	if meta.PeriodEnd != nil {
		meta.BillingCycleDay = meta.PeriodEnd.Day()
	}

	return meta
}

func detectAccountType(lower string) string {
	creditCardSignals := []string{
		"credit card", "creditcard", "card member", "cardmember",
		"minimum payment", "credit limit", "available credit",
		"statement balance", "new charges", "finance charge",
		"annual percentage rate", "apr ", "billing cycle",
		"cash advance", "payment due date", "pay by",
		"previous balance", "new balance",
	}
	checkingSignals := []string{
		"checking account", "current account", "transaction account",
		"opening balance", "closing balance", "running balance",
		"withdrawal", "deposit", "cheque", "check number",
		"available balance",
	}
	savingsSignals := []string{
		"savings account", "interest earned", "interest rate",
	}

	creditScore := 0
	for _, s := range creditCardSignals {
		if strings.Contains(lower, s) {
			creditScore++
		}
	}
	checkingScore := 0
	for _, s := range checkingSignals {
		if strings.Contains(lower, s) {
			checkingScore++
		}
	}
	savingsScore := 0
	for _, s := range savingsSignals {
		if strings.Contains(lower, s) {
			savingsScore++
		}
	}

	if creditScore > checkingScore && creditScore > savingsScore {
		return "credit_card"
	}
	if savingsScore > checkingScore {
		return "savings"
	}
	if checkingScore > 0 {
		return "checking"
	}
	// Default: if we see "card" anywhere, likely credit card
	if strings.Contains(lower, "card") {
		return "credit_card"
	}
	return "checking"
}

// Known institutions with their variations
var institutions = []struct {
	name     string
	patterns []string
}{
	{"American Express", []string{"american express", "americanexpress", "amex"}},
	{"HDFC Bank", []string{"hdfc bank", "hdfcbank"}},
	{"State Bank of India", []string{"state bank of india", "sbi ", " sbi"}},
	{"ICICI Bank", []string{"icici bank", "icicibank"}},
	{"Axis Bank", []string{"axis bank", "axisbank"}},
	{"Kotak Mahindra", []string{"kotak mahindra", "kotak bank"}},
	{"Yes Bank", []string{"yes bank"}},
	{"IndusInd Bank", []string{"indusind bank"}},
	{"Punjab National Bank", []string{"punjab national bank", "pnb "}},
	{"Bank of Baroda", []string{"bank of baroda", "bob "}},
	{"Canara Bank", []string{"canara bank"}},
	{"Union Bank", []string{"union bank"}},
	{"IDBI Bank", []string{"idbi bank"}},
	{"Federal Bank", []string{"federal bank"}},
	{"RBL Bank", []string{"rbl bank", "ratnakar bank"}},
	{"Chase", []string{"chase bank", "jpmorgan chase", "jpmorganchase"}},
	{"Citibank", []string{"citibank", "citi bank"}},
	{"Bank of America", []string{"bank of america"}},
	{"Wells Fargo", []string{"wells fargo"}},
	{"Capital One", []string{"capital one"}},
	{"Discover", []string{"discover card", "discover financial"}},
	{"Barclays", []string{"barclays"}},
	{"HSBC", []string{"hsbc"}},
	{"Standard Chartered", []string{"standard chartered"}},
	{"DBS Bank", []string{"dbs bank"}},
}

func detectInstitution(lower, original string) string {
	for _, inst := range institutions {
		for _, pattern := range inst.patterns {
			if strings.Contains(lower, pattern) {
				return inst.name
			}
		}
	}
	return ""
}

// Account number patterns
var accountPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)account\s*(?:ending|number|no\.?|#)\s*:?\s*[X*x0-]*[-\s]?(\d{4,6})`),
	regexp.MustCompile(`(?i)card\s*(?:ending|number|no\.?|#)\s*:?\s*[X*x0-]*[-\s]?(\d{4,6})`),
	regexp.MustCompile(`(?i)ending\s*(?:in\s+)?[X*x0-]*[-]?(\d{4,6})`),
	regexp.MustCompile(`(?i)a/c\s*(?:no\.?)?\s*:?\s*[X*x]*(\d{4,6})`),
	regexp.MustCompile(`XXXX[-\s]?(\d{4,6})`),
	regexp.MustCompile(`[X*]{4,}[-\s]?(\d{4,6})`),
}

func detectAccountNumber(text string) string {
	for _, pat := range accountPatterns {
		m := pat.FindStringSubmatch(text)
		if m != nil && len(m) >= 2 {
			return "XXXX" + m[1]
		}
	}
	return ""
}

// Period detection patterns
var periodPatterns = []*regexp.Regexp{
	// "Statement Period: 01/05/2026 to 02/03/2026"
	regexp.MustCompile(`(?i)(?:statement\s*period|billing\s*period|from)\s*:?\s*(\d{1,2}[/\-]\d{1,2}[/\-]\d{2,4})\s*(?:to|through|-|–)\s*(\d{1,2}[/\-]\d{1,2}[/\-]\d{2,4})`),
	// "From 01 Jan 2026 To 31 Jan 2026"
	regexp.MustCompile(`(?i)from\s+(\d{1,2}\s+[A-Za-z]{3}\s+\d{2,4})\s+to\s+(\d{1,2}\s+[A-Za-z]{3}\s+\d{2,4})`),
	// "Closing Date 02/03/21"
	regexp.MustCompile(`(?i)closing\s*date\s*:?\s*(\d{1,2}[/\-]\d{1,2}[/\-]\d{2,4})`),
}

func detectPeriod(text string) (*time.Time, *time.Time) {
	for _, pat := range periodPatterns {
		m := pat.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		if len(m) >= 3 {
			start := normalizeDate(m[1])
			end := normalizeDate(m[2])
			if !start.IsZero() && !end.IsZero() {
				return &start, &end
			}
		}
		if len(m) >= 2 {
			// Closing date only — estimate period as 30 days before
			end := normalizeDate(m[1])
			if !end.IsZero() {
				start := end.AddDate(0, -1, 0)
				return &start, &end
			}
		}
	}
	return nil, nil
}

var dueDatePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:payment\s*due\s*date|pay\s*by|due\s*by|due\s*date)\s*:?\s*(\d{1,2}[/\-]\d{1,2}[/\-]\d{2,4})`),
	regexp.MustCompile(`(?i)(?:payment\s*due\s*date|pay\s*by|due\s*by)\s*:?\s*(\d{1,2}\s+[A-Za-z]{3}\s+\d{2,4})`),
}

func detectPaymentDue(text string) (*time.Time, int) {
	for _, pat := range dueDatePatterns {
		m := pat.FindStringSubmatch(text)
		if m != nil && len(m) >= 2 {
			d := normalizeDate(m[1])
			if !d.IsZero() {
				return &d, d.Day()
			}
		}
	}
	return nil, 0
}

func detectAmount(text, pattern string) float64 {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(text)
	if m != nil && len(m) >= 2 {
		cleaned := strings.ReplaceAll(m[1], ",", "")
		var val float64
		fmt_scan(cleaned, &val)
		return val
	}
	return 0
}

// simple float parse without importing strconv to avoid circular deps
func fmt_scan(s string, v *float64) {
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	var result float64
	var decimal float64
	inDecimal := false
	divisor := 1.0
	for _, c := range s {
		if c == '.' {
			inDecimal = true
			continue
		}
		if c >= '0' && c <= '9' {
			digit := float64(c - '0')
			if inDecimal {
				divisor *= 10
				decimal += digit / divisor
			} else {
				result = result*10 + digit
			}
		}
	}
	result += decimal
	if neg {
		result = -result
	}
	*v = result
}
