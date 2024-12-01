package database

import (
	"fmt"

	"github.com/ksred/klear-api/internal/clearing"
	"github.com/ksred/klear-api/internal/database/migrations"
	"github.com/ksred/klear-api/internal/settlement"
	"github.com/ksred/klear-api/internal/trading"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewDatabase initializes and returns a new GORM DB connection
func NewDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := migrations.AddExchangeFills(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := migrations.AddTradeNetting(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Auto-migrate other schemas
	err = db.AutoMigrate(
		&trading.Order{},
		&trading.IdempotencyRecord{},
		&clearing.Clearing{},
		&settlement.Settlement{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}
