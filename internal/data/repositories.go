// Package data owns persistence. The Repositories aggregate is the single
// surface other packages depend on so we can swap implementations (HTTP-to-
// Thesaurus during the migration; direct GORM after Phase 2).
package data

import (
	"context"
	"time"
)

// Transaction is the lean shape consumers use. Persisted shape lives in
// internal/data/models once Phase 2 lifts the GORM models.
type Transaction struct {
	ID          string
	UserID      string
	Date        time.Time
	Description string
	Amount      float64
	Category    string
	Type        string
}

// DismissedInsight is the lean shape for the dismissals table.
type DismissedInsight struct {
	UserID      string
	InsightKey  string
	RuleID      string
	DismissedAt time.Time
}

// Repositories is the umbrella interface other packages depend on.
type Repositories interface {
	FetchTransactionsSince(ctx context.Context, userID string, start time.Time) ([]Transaction, error)
	ListDismissedInsightKeys(ctx context.Context, userID string) (map[string]struct{}, error)
	CreateDismissedInsight(ctx context.Context, d DismissedInsight) error
}
