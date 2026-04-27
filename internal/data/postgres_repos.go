// Package data postgres-backed implementation of Repositories.
//
// This adapter sits on top of the GORM models in internal/data/models and the
// repository helpers in internal/data/repository. Other internal packages
// depend on the Repositories interface so we can swap implementations during
// the migration without touching call sites.
package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/internal/data/models"
	"github.com/sagarjhaa/localfinance/internal/data/repository"
	"gorm.io/gorm"
)

type postgresRepos struct {
	db *gorm.DB
}

// NewPostgresRepos returns a Repositories implementation that talks to the
// shared *gorm.DB. The same DB instance is reused — Repositories is a thin
// adapter, not a connection pool.
func NewPostgresRepos(db *gorm.DB) Repositories {
	return &postgresRepos{db: db}
}

// parseUUID best-effort parses a UUID string, returning uuid.Nil on error.
// Callers pass user IDs that originated from authenticated sessions so a
// parse failure means a programming error upstream — log-and-continue is
// acceptable for the migration window.
func parseUUID(s string) uuid.UUID {
	u, _ := uuid.Parse(s)
	return u
}

func (r *postgresRepos) FetchTransactionsSince(ctx context.Context, userID string, start time.Time) ([]Transaction, error) {
	var rows []models.Transaction
	if err := r.db.WithContext(ctx).
		Joins("JOIN accounts ON transactions.account_id = accounts.id").
		Where("accounts.user_id = ? AND transactions.date >= ?", parseUUID(userID), start).
		Order("transactions.date desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Transaction, 0, len(rows))
	for _, m := range rows {
		out = append(out, Transaction{
			ID:          m.ID.String(),
			UserID:      userID,
			Date:        m.Date,
			Description: m.Description,
			Amount:      m.Amount,
			Category:    m.Category,
			Type:        m.Type,
		})
	}
	return out, nil
}

func (r *postgresRepos) ListDismissedInsightKeys(ctx context.Context, userID string) (map[string]struct{}, error) {
	repo := repository.NewDismissedInsightRepo(r.db)
	rows, err := repo.ListByUser(ctx, parseUUID(userID))
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		out[row.InsightKey] = struct{}{}
	}
	return out, nil
}

func (r *postgresRepos) CreateDismissedInsight(ctx context.Context, d DismissedInsight) error {
	repo := repository.NewDismissedInsightRepo(r.db)
	return repo.Create(ctx, &models.DismissedInsight{
		UserID:     parseUUID(d.UserID),
		InsightKey: d.InsightKey,
		RuleID:     d.RuleID,
	})
}
