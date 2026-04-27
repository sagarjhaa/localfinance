package database

import (
	"fmt"
	"log"

	"github.com/sagarjhaa/localfinance/services/thesaurus/auth"
	"github.com/sagarjhaa/localfinance/services/thesaurus/config"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Default credentials used by SeedDefaultUser. This is a single-user
// privacy-first app — these are intentionally weak so the user can log in
// on a fresh install and change them in the Profile page.
const (
	DefaultUserEmail     = "local@localfinance.app"
	DefaultUserPassword  = "localfinance"
	DefaultUserFirstName = "Local"
	DefaultUserLastName  = "User"
)

func Initialize(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB for connection pooling
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("🗄️ Database connection established")
	return db, nil
}

func Migrate(db *gorm.DB) error {
	log.Println("🔄 Running database migrations...")
	
	err := db.AutoMigrate(
		&models.User{},
		&models.UserSession{},
		&models.Account{},
		&models.Transaction{},
		&models.Budget{},
		&models.Document{},
		&models.StatementPeriod{},
		&models.CategoryRule{},
		&models.Conversation{},
		&models.ChatMessage{},
		&models.UserPreference{},
	)
	
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✅ Database migrations completed")
	return nil
}

// SeedDefaultUser creates the local-only default user on first boot if no
// users exist. Single-user app: this user is the implicit owner of all data.
// On a fresh install the credentials are local@localfinance.app / localfinance;
// the user can change them in the Profile page.
//
// Idempotent — safe to re-call.
func SeedDefaultUser(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if count > 0 {
		log.Println("👤 default user already exists")
		return nil
	}

	hashed, err := auth.HashPassword(DefaultUserPassword)
	if err != nil {
		return fmt.Errorf("failed to hash default password: %w", err)
	}

	user := models.User{
		Email:     DefaultUserEmail,
		Password:  hashed,
		FirstName: DefaultUserFirstName,
		LastName:  DefaultUserLastName,
		IsActive:  true,
	}

	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("failed to create default user: %w", err)
	}

	log.Printf("👤 seeded default user (%s)", DefaultUserEmail)
	return nil
}