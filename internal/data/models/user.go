package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email" binding:"required,email"`
	Password  string         `gorm:"not null" json:"-"`
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Accounts  []Account      `gorm:"foreignKey:UserID" json:"accounts,omitempty"`
	Budgets   []Budget       `gorm:"foreignKey:UserID" json:"budgets,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type UserSession struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Token     string    `gorm:"not null;index" json:"token"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *UserSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
