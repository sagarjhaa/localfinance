package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Conversation struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Title     string         `gorm:"default:'New Chat'" json:"title"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Messages  []ChatMessage  `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type ChatMessage struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ConversationID uuid.UUID `gorm:"type:uuid;not null;index" json:"conversation_id"`
	Role           string    `gorm:"not null" json:"role"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	Confidence     *float64  `json:"confidence,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (m *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
