package migrations

import (
	"github.com/ksred/klear-api/internal/clearing"
	"gorm.io/gorm"
)

// AddTradeNetting creates the trade netting table and required indexes
func AddTradeNetting(db *gorm.DB) error {
	// Create the trade netting table
	if err := db.AutoMigrate(&clearing.TradeNetting{}); err != nil {
		return err
	}

	// Add indexes for better query performance
	// Using raw SQL for index creation to have more control over index types
	indexes := []string{
		// Index for symbol lookups
		`CREATE INDEX IF NOT EXISTS idx_trade_nettings_symbol 
		 ON trade_nettings(symbol)`,

		// Composite index for time window queries
		`CREATE INDEX IF NOT EXISTS idx_trade_nettings_window 
		 ON trade_nettings(window_start, window_end)`,

		// Index for status filtering
		`CREATE INDEX IF NOT EXISTS idx_trade_nettings_status 
		 ON trade_nettings(status)`,

		// Index for created_at timestamp (useful for time-based queries)
		`CREATE INDEX IF NOT EXISTS idx_trade_nettings_created_at 
		 ON trade_nettings(created_at)`,

		// Composite index for symbol and status (common query pattern)
		`CREATE INDEX IF NOT EXISTS idx_trade_nettings_symbol_status 
		 ON trade_nettings(symbol, status)`,
	}

	// Execute each index creation
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
} 