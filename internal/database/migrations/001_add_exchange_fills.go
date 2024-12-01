package migrations

import (
	"github.com/ksred/klear-api/internal/types"
	"gorm.io/gorm"
)

func AddExchangeFills(db *gorm.DB) error {
	// Drop the old executions table and recreate with new schema
	// if err := db.Migrator().DropTable(&types.Execution{}); err != nil {
	// 	return err
	// }

	// Create the new tables
	if err := db.AutoMigrate(&types.ExchangeFill{}); err != nil {
		return err
	}

	if err := db.AutoMigrate(&types.Execution{}); err != nil {
		return err
	}

	return nil
}
