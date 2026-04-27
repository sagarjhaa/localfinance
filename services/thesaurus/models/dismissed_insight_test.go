package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDismissedInsightBeforeCreate(t *testing.T) {
	d := &DismissedInsight{}
	if err := d.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate failed: %v", err)
	}
	if d.ID == uuid.Nil {
		t.Fatal("BeforeCreate should set a non-nil UUID")
	}
	if d.DismissedAt.IsZero() {
		t.Fatal("BeforeCreate should set DismissedAt when zero")
	}
}

func TestDismissedInsightBeforeCreatePreservesFields(t *testing.T) {
	existingID := uuid.New()
	existingTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &DismissedInsight{ID: existingID, DismissedAt: existingTime}
	if err := d.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate failed: %v", err)
	}
	if d.ID != existingID {
		t.Fatal("BeforeCreate should not overwrite existing ID")
	}
	if !d.DismissedAt.Equal(existingTime) {
		t.Fatal("BeforeCreate should not overwrite existing DismissedAt")
	}
}
