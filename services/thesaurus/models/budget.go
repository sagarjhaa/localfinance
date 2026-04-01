package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Budget struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	User      *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Category  string         `gorm:"not null" json:"category" binding:"required"`
	Amount    float64        `gorm:"not null" json:"amount" binding:"required"`
	Period    string         `gorm:"not null;default:monthly" json:"period"` // monthly, weekly, yearly
	StartDate time.Time      `json:"start_date"`
	EndDate   time.Time      `json:"end_date"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *Budget) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
