package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CategoryRule is a user-configurable merchant->category mapping
type CategoryRule struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Pattern   string         `gorm:"not null" json:"pattern"`  // merchant name pattern (case-insensitive match)
	Category  string         `gorm:"not null" json:"category"` // target category
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
