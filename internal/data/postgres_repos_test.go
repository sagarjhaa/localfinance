package data

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/internal/data/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestDB spins up an in-memory SQLite DB and migrates the relevant tables
// for the postgres-backed Repositories adapter. SQLite stands in for Postgres
// in tests so the suite stays hermetic; production wires the same code to a
// real *gorm.DB pointed at Postgres.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Account{},
		&models.Transaction{},
		&models.DismissedInsight{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestFetchTransactionsSince(t *testing.T) {
	db := newTestDB(t)
	repos := NewPostgresRepos(db)
	ctx := context.Background()

	userID := uuid.New()
	otherUser := uuid.New()

	// Two accounts: one for our user, one for somebody else.
	mine := &models.Account{UserID: userID, Name: "mine", Type: "checking"}
	theirs := &models.Account{UserID: otherUser, Name: "theirs", Type: "checking"}
	if err := db.Create(mine).Error; err != nil {
		t.Fatalf("create mine: %v", err)
	}
	if err := db.Create(theirs).Error; err != nil {
		t.Fatalf("create theirs: %v", err)
	}

	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -30)

	// In-window for our user.
	tx1 := &models.Transaction{AccountID: mine.ID, Date: now.AddDate(0, 0, -1), Description: "fresh", Amount: 10}
	// Out-of-window for our user.
	tx2 := &models.Transaction{AccountID: mine.ID, Date: now.AddDate(0, 0, -60), Description: "stale", Amount: 20}
	// In-window but for the other user.
	tx3 := &models.Transaction{AccountID: theirs.ID, Date: now.AddDate(0, 0, -1), Description: "leak", Amount: 30}
	for _, tx := range []*models.Transaction{tx1, tx2, tx3} {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("create tx: %v", err)
		}
	}

	got, err := repos.FetchTransactionsSince(ctx, userID.String(), cutoff)
	if err != nil {
		t.Fatalf("FetchTransactionsSince: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 row in window, got %d (%+v)", len(got), got)
	}
	if got[0].Description != "fresh" {
		t.Fatalf("expected 'fresh', got %q", got[0].Description)
	}
	if got[0].UserID != userID.String() {
		t.Fatalf("expected UserID propagated to lean shape, got %q", got[0].UserID)
	}
}

func TestListDismissedInsightKeys(t *testing.T) {
	db := newTestDB(t)
	repos := NewPostgresRepos(db)
	ctx := context.Background()

	userID := uuid.New()
	other := uuid.New()

	// Empty case.
	keys, err := repos.ListDismissedInsightKeys(ctx, userID.String())
	if err != nil {
		t.Fatalf("ListDismissedInsightKeys empty: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected empty set, got %v", keys)
	}

	// Seed two dismissals for our user, one for someone else.
	for _, k := range []string{"k1", "k2"} {
		if err := db.Create(&models.DismissedInsight{UserID: userID, InsightKey: k, RuleID: "r"}).Error; err != nil {
			t.Fatalf("seed %s: %v", k, err)
		}
	}
	if err := db.Create(&models.DismissedInsight{UserID: other, InsightKey: "k1", RuleID: "r"}).Error; err != nil {
		t.Fatalf("seed other: %v", err)
	}

	keys, err = repos.ListDismissedInsightKeys(ctx, userID.String())
	if err != nil {
		t.Fatalf("ListDismissedInsightKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys for user, got %d (%v)", len(keys), keys)
	}
	if _, ok := keys["k1"]; !ok {
		t.Fatal("expected k1 in set")
	}
	if _, ok := keys["k2"]; !ok {
		t.Fatal("expected k2 in set")
	}
}

func TestCreateDismissedInsightIdempotent(t *testing.T) {
	db := newTestDB(t)
	repos := NewPostgresRepos(db)
	ctx := context.Background()

	userID := uuid.New()
	d := DismissedInsight{
		UserID:     userID.String(),
		InsightKey: "k1",
		RuleID:     "rule.high_spend",
	}

	if err := repos.CreateDismissedInsight(ctx, d); err != nil {
		t.Fatalf("CreateDismissedInsight #1: %v", err)
	}
	if err := repos.CreateDismissedInsight(ctx, d); err != nil {
		t.Fatalf("CreateDismissedInsight #2 (duplicate): %v", err)
	}

	var count int64
	if err := db.Model(&models.DismissedInsight{}).
		Where("user_id = ? AND insight_key = ?", userID, "k1").
		Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row after duplicate Create, got %d", count)
	}
}
