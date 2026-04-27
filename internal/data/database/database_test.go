package database

import (
	"os"
	"testing"

	"github.com/sagarjhaa/localfinance/internal/auth"
	"github.com/sagarjhaa/localfinance/internal/data/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestDefaultUserCredentialsHashable ensures the baked-in default password
// can be hashed and re-verified — a fast guard that runs without a DB.
// Catches typos in the constants or a regression in the Argon2id helper.
func TestDefaultUserCredentialsHashable(t *testing.T) {
	if DefaultUserEmail == "" || DefaultUserPassword == "" {
		t.Fatal("default email and password must be set")
	}

	hashed, err := auth.HashPassword(DefaultUserPassword)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	ok, err := auth.VerifyPassword(DefaultUserPassword, hashed)
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if !ok {
		t.Fatal("expected default password to verify against its own hash")
	}

	// Negative: a different password must not verify
	ok, err = auth.VerifyPassword("not-the-password", hashed)
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail verification")
	}
}

// TestSeedDefaultUser_PostgresIntegration runs against a real Postgres if
// TEST_DATABASE_URL is set (e.g. in CI or via `make dev-up`). On a fresh DB
// it should create the user; a second call must be a no-op (idempotent).
//
// Skipped when the env var is missing — keeps `go test ./...` green on any
// dev machine without standing up Postgres. The full local-stack contract
// is covered by `make test-e2e` and the smoke verification in §4.
func TestSeedDefaultUser_PostgresIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test postgres: %v", err)
	}

	// Use a clean slate — wipe any prior users in the test DB.
	if err := db.Exec("DELETE FROM users").Error; err != nil {
		t.Fatalf("failed to truncate users: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// First call — should seed.
	if err := SeedDefaultUser(db); err != nil {
		t.Fatalf("SeedDefaultUser (first call) failed: %v", err)
	}
	var u models.User
	if err := db.Where("email = ?", DefaultUserEmail).First(&u).Error; err != nil {
		t.Fatalf("default user not found after seed: %v", err)
	}
	if u.FirstName != DefaultUserFirstName || u.LastName != DefaultUserLastName {
		t.Fatalf("seeded user has wrong name fields: %+v", u)
	}

	// Second call — must be a no-op (still exactly 1 user).
	if err := SeedDefaultUser(db); err != nil {
		t.Fatalf("SeedDefaultUser (second call) failed: %v", err)
	}
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 user after idempotent re-seed, got %d", count)
	}
}
