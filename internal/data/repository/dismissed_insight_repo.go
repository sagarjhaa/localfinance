// Package repository contains data-access helpers for Thesaurus models.
//
// Most Thesaurus handlers talk to GORM directly. The DismissedInsight
// resource is the first to use a repository because the upsert/no-op
// semantics on (user_id, insight_key) are easier to test in isolation
// than when interleaved with HTTP plumbing.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/internal/data/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DismissedInsightRepo persists user-level insight dismissals.
type DismissedInsightRepo struct {
	db *gorm.DB
}

func NewDismissedInsightRepo(db *gorm.DB) *DismissedInsightRepo {
	return &DismissedInsightRepo{db: db}
}

// Create inserts a dismissal row, or no-ops if (user_id, insight_key)
// already exists. The passed-in *DismissedInsight is updated with the
// stored row's ID/timestamps on success.
func (r *DismissedInsightRepo) Create(ctx context.Context, d *models.DismissedInsight) error {
	if d == nil {
		return errors.New("dismissed insight is nil")
	}

	// ON CONFLICT (user_id, insight_key) DO NOTHING. We then re-read so
	// the caller sees the canonical row (the one already in the DB if
	// this was a duplicate, or the freshly inserted one otherwise).
	tx := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "insight_key"}},
			DoNothing: true,
		}).
		Create(d)
	if tx.Error != nil {
		return tx.Error
	}

	// If RowsAffected==0 the conflict path fired; load the existing row.
	if tx.RowsAffected == 0 {
		var existing models.DismissedInsight
		if err := r.db.WithContext(ctx).
			Where("user_id = ? AND insight_key = ?", d.UserID, d.InsightKey).
			First(&existing).Error; err != nil {
			return err
		}
		*d = existing
	}
	return nil
}

// ListByUser returns all dismissals for a user, newest first.
func (r *DismissedInsightRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.DismissedInsight, error) {
	var out []models.DismissedInsight
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("dismissed_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Exists reports whether the user has dismissed the given insight key.
func (r *DismissedInsightRepo) Exists(ctx context.Context, userID uuid.UUID, insightKey string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.DismissedInsight{}).
		Where("user_id = ? AND insight_key = ?", userID, insightKey).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Delete removes a dismissal (un-dismiss). Returns the number of rows removed.
func (r *DismissedInsightRepo) Delete(ctx context.Context, userID uuid.UUID, insightKey string) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("user_id = ? AND insight_key = ?", userID, insightKey).
		Delete(&models.DismissedInsight{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
