package models

import (
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID               string    `gorm:"primaryKey" json:"id"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Filename         string    `gorm:"not null" json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	FileSize         int64     `json:"file_size"`
	Status           string    `gorm:"default:uploaded" json:"status"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	CorrelationID    string    `json:"correlation_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
