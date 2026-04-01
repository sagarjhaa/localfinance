package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestUserBeforeCreate(t *testing.T) {
	user := &User{}
	if err := user.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate failed: %v", err)
	}
	if user.ID == uuid.Nil {
		t.Fatal("BeforeCreate should set a non-nil UUID")
	}
}

func TestUserBeforeCreatePreservesID(t *testing.T) {
	existingID := uuid.New()
	user := &User{ID: existingID}
	if err := user.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate failed: %v", err)
	}
	if user.ID != existingID {
		t.Fatal("BeforeCreate should not overwrite existing ID")
	}
}

func TestAccountBeforeCreate(t *testing.T) {
	account := &Account{}
	if err := account.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate failed: %v", err)
	}
	if account.ID == uuid.Nil {
		t.Fatal("BeforeCreate should set a non-nil UUID")
	}
}

func TestTransactionBeforeCreate(t *testing.T) {
	txn := &Transaction{}
	if err := txn.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate failed: %v", err)
	}
	if txn.ID == uuid.Nil {
		t.Fatal("BeforeCreate should set a non-nil UUID")
	}
}

func TestBudgetBeforeCreate(t *testing.T) {
	budget := &Budget{}
	if err := budget.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate failed: %v", err)
	}
	if budget.ID == uuid.Nil {
		t.Fatal("BeforeCreate should set a non-nil UUID")
	}
}

func TestUserSessionBeforeCreate(t *testing.T) {
	session := &UserSession{}
	if err := session.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate failed: %v", err)
	}
	if session.ID == uuid.Nil {
		t.Fatal("BeforeCreate should set a non-nil UUID")
	}
}
