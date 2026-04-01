package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Account struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	User         *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Name         string         `gorm:"not null" json:"name" binding:"required"`
	Type         string         `gorm:"not null" json:"type" binding:"required"` // checking, savings, credit
	Balance      float64        `gorm:"default:0" json:"balance"`
	Currency     string         `gorm:"default:INR" json:"currency"`
	Institution    string  `json:"institution"`
	AccountNumber  string  `json:"account_number"` // masked, e.g. "XXXX1007"
	CreditLimit    float64 `json:"credit_limit,omitempty"`
	PaymentDueDay  int     `json:"payment_due_day,omitempty"`  // day of month
	MinPaymentPct  float64 `json:"min_payment_pct,omitempty"`  // e.g. 0.05 = 5%
	BillingCycleDay int    `json:"billing_cycle_day,omitempty"` // statement closing day
	IsActive       bool    `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Transactions []Transaction  `gorm:"foreignKey:AccountID" json:"transactions,omitempty"`
}

func (a *Account) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
