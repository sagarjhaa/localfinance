package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StatementPeriod tracks each uploaded billing cycle for an account
type StatementPeriod struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	AccountID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"account_id"`
	Account          *Account   `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	DocumentID       string     `gorm:"index" json:"document_id"`
	PeriodStart      time.Time  `json:"period_start"`
	PeriodEnd        time.Time  `json:"period_end"`
	OpeningBalance   float64    `json:"opening_balance"`
	ClosingBalance   float64    `json:"closing_balance"`
	TotalCredits     float64    `json:"total_credits"`
	TotalDebits      float64    `json:"total_debits"`
	PaymentDueDate   *time.Time `json:"payment_due_date,omitempty"`
	MinPaymentDue    float64    `json:"min_payment_due,omitempty"`
	TransactionCount int        `json:"transaction_count"`
	CreatedAt        time.Time  `json:"created_at"`
}

func (s *StatementPeriod) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
