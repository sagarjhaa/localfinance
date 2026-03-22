package database

import (
	"fmt"
	"log"

	"github.com/sagarjhaa/localfinance/services/thesaurus/config"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
		&models.Account{},
		&models.Transaction{},
		&models.Budget{},
	)
	
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✅ Database migrations completed")
	return nil
}