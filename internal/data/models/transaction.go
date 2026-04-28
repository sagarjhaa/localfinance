package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Transaction struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	AccountID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"account_id"`
	Account     *Account       `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	Date        time.Time      `gorm:"not null;index" json:"date" binding:"required"`
	Description string         `gorm:"not null" json:"description" binding:"required"`
	Amount      float64        `gorm:"not null" json:"amount" binding:"required"`
	Category    string         `gorm:"index" json:"category"`
	Type        string         `json:"type"` // debit, credit
	Reference   string         `json:"reference"`
	DocumentID  string         `gorm:"index" json:"document_id,omitempty"`
	Notes       string         `json:"notes"`
	// User feedback. Set via POST /api/v1/transactions/:id/feedback. Used
	// downstream by chat + the parse pipeline to lean on user-corrected
	// categories instead of LLM guesses.
	UserSignal           string `json:"user_signal,omitempty"`            // "" | "up" | "down"
	UserCorrectedCategory string `json:"user_corrected_category,omitempty"`
	UserFeedbackReason    string `json:"user_feedback_reason,omitempty"`   // "wrong_category" | "wrong_amount" | "not_mine" | "other"
	UserFeedbackNote      string `json:"user_feedback_note,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

type TransactionFilter struct {
	UserID      *uuid.UUID `json:"user_id"`
	AccountID   *uuid.UUID `json:"account_id"`
	Category    *string    `json:"category"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
	MinAmount   *float64   `json:"min_amount"`
	MaxAmount   *float64   `json:"max_amount"`
	Description *string    `json:"description"`
	Limit       int        `json:"limit"`
	Offset      int        `json:"offset"`
}

type SpendingSummary struct {
	Category    string  `json:"category"`
	TotalAmount float64 `json:"total_amount"`
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
}
