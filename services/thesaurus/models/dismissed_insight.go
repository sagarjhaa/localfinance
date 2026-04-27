package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DismissedInsight records when a user dismisses a recomputed insight.
// Insights themselves are recomputed on demand by Sophia (Hybrid C);
// this table only persists the dismissal so it survives reloads.
//
// (user_id, insight_key) is unique — a second dismissal of the same
// insight is a no-op upsert.
type DismissedInsight struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_user_insight" json:"user_id"`
	InsightKey  string    `gorm:"not null;index;uniqueIndex:idx_user_insight" json:"insight_key"`
	RuleID      string    `gorm:"not null" json:"rule_id"`
	DismissedAt time.Time `json:"dismissed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (d *DismissedInsight) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.DismissedAt.IsZero() {
		d.DismissedAt = time.Now()
	}
	return nil
}
