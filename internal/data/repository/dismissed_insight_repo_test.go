package repository

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/internal/data/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestDB spins up an in-memory SQLite DB and migrates the
// DismissedInsight table. Each test gets a fresh DB so they're isolated.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.DismissedInsight{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestDismissedInsightRepo_CreateNew(t *testing.T) {
	db := newTestDB(t)
	repo := NewDismissedInsightRepo(db)
	ctx := context.Background()
	uid := uuid.New()

	d := &models.DismissedInsight{UserID: uid, InsightKey: "k1", RuleID: "rule.high_spend"}
	if err := repo.Create(ctx, d); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if d.ID == uuid.Nil {
		t.Fatal("Create should populate ID")
	}
	if d.DismissedAt.IsZero() {
		t.Fatal("Create should populate DismissedAt")
	}
}

func TestDismissedInsightRepo_CreateDuplicateNoOp(t *testing.T) {
	db := newTestDB(t)
	repo := NewDismissedInsightRepo(db)
	ctx := context.Background()
	uid := uuid.New()

	first := &models.DismissedInsight{UserID: uid, InsightKey: "k1", RuleID: "rule.a"}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create #1: %v", err)
	}

	second := &models.DismissedInsight{UserID: uid, InsightKey: "k1", RuleID: "rule.b"}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create #2: %v", err)
	}

	// The duplicate insert must yield the original row.
	if second.ID != first.ID {
		t.Fatalf("expected duplicate Create to return original ID %s, got %s", first.ID, second.ID)
	}

	// And there must still be exactly one row in the DB.
	var count int64
	if err := db.Model(&models.DismissedInsight{}).
		Where("user_id = ? AND insight_key = ?", uid, "k1").
		Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row after duplicate Create, got %d", count)
	}
}

func TestDismissedInsightRepo_ListByUser(t *testing.T) {
	db := newTestDB(t)
	repo := NewDismissedInsightRepo(db)
	ctx := context.Background()
	uid := uuid.New()
	other := uuid.New()

	// Empty list.
	got, err := repo.ListByUser(ctx, uid)
	if err != nil {
		t.Fatalf("ListByUser empty: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(got))
	}

	// Add two for our user, one for someone else.
	for _, k := range []string{"k1", "k2"} {
		if err := repo.Create(ctx, &models.DismissedInsight{
			UserID: uid, InsightKey: k, RuleID: "r",
		}); err != nil {
			t.Fatalf("Create %s: %v", k, err)
		}
	}
	if err := repo.Create(ctx, &models.DismissedInsight{
		UserID: other, InsightKey: "k1", RuleID: "r",
	}); err != nil {
		t.Fatalf("Create other: %v", err)
	}

	got, err = repo.ListByUser(ctx, uid)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows for user, got %d", len(got))
	}
	for _, r := range got {
		if r.UserID != uid {
			t.Fatalf("ListByUser leaked another user's row: %+v", r)
		}
	}
}

func TestDismissedInsightRepo_Exists(t *testing.T) {
	db := newTestDB(t)
	repo := NewDismissedInsightRepo(db)
	ctx := context.Background()
	uid := uuid.New()

	ok, err := repo.Exists(ctx, uid, "k1")
	if err != nil {
		t.Fatalf("Exists empty: %v", err)
	}
	if ok {
		t.Fatal("Exists should be false on empty table")
	}

	if err := repo.Create(ctx, &models.DismissedInsight{
		UserID: uid, InsightKey: "k1", RuleID: "r",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	ok, err = repo.Exists(ctx, uid, "k1")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !ok {
		t.Fatal("Exists should be true after Create")
	}

	ok, err = repo.Exists(ctx, uid, "missing")
	if err != nil {
		t.Fatalf("Exists missing: %v", err)
	}
	if ok {
		t.Fatal("Exists should be false for unknown key")
	}
}

func TestDismissedInsightRepo_Delete(t *testing.T) {
	db := newTestDB(t)
	repo := NewDismissedInsightRepo(db)
	ctx := context.Background()
	uid := uuid.New()

	// Delete missing row → 0 affected, no error.
	affected, err := repo.Delete(ctx, uid, "nope")
	if err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
	if affected != 0 {
		t.Fatalf("expected 0 rows affected, got %d", affected)
	}

	if err := repo.Create(ctx, &models.DismissedInsight{
		UserID: uid, InsightKey: "k1", RuleID: "r",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	affected, err = repo.Delete(ctx, uid, "k1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected 1 row affected, got %d", affected)
	}

	exists, _ := repo.Exists(ctx, uid, "k1")
	if exists {
		t.Fatal("row should be gone after Delete")
	}
}
