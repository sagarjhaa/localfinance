package models

import (
	"time"

	"github.com/google/uuid"
)

// Transaction represents a parsed financial transaction
type Transaction struct {
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Category    string    `json:"category"`
	Type        string    `json:"type"` // credit or debit
	Reference   string    `json:"reference"`
	FileSource  string    `json:"file_source"`
	CreatedAt   time.Time `json:"created_at"`
}

// ProcessingResult represents the result of processing a document
type ProcessingResult struct {
	ID                uuid.UUID     `json:"id"`
	Filename          string        `json:"filename"`
	FileType          string        `json:"file_type"`
	FileSize          int64         `json:"file_size"`
	Status            string        `json:"status"` // success, error
	ErrorMessage      string        `json:"error_message,omitempty"`
	TransactionsFound int           `json:"transactions_found"`
	Transactions      []Transaction `json:"transactions"`
	ProcessedAt       time.Time     `json:"processed_at"`
	ProcessingTimeMs  int64         `json:"processing_time_ms"`
}

// ProcessRequest is the request from Thesaurus to process a document
type ProcessRequest struct {
	DocumentID    string    `json:"document_id" binding:"required"`
	FilePath      string    `json:"file_path" binding:"required"`
	FileType      string    `json:"file_type" binding:"required"`
	UserID        uuid.UUID `json:"user_id"`
	AccountID     uuid.UUID `json:"account_id"`
	CorrelationID string    `json:"correlation_id"`
}
