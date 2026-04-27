package parse

import (
	"github.com/sagarjhaa/localfinance/services/logos/models"
)

// ProcessResult is the lightweight result Logos hands off after AI parsing.
// Logos no longer dispatches by file type or runs regex-based parsers — every
// upload is routed to Sophia's /api/v1/parse endpoint, and the parsed
// transactions are wrapped in this struct for the existing categorize / send
// pipeline in main.go.
type ProcessResult struct {
	Transactions      []models.Transaction
	TransactionsFound int
	ProcessingTimeMs  int64
	FileSize          int64
}
