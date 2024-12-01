package settlement

import (
	"errors"
	"fmt"
	"time"

	"github.com/ksred/klear-api/internal/clearing"
	"github.com/ksred/klear-api/internal/types"
	"gorm.io/gorm"
)

type Database struct {
	db *gorm.DB
}

func NewDatabase(db *gorm.DB) *Database {
	return &Database{db: db}
}

func (d *Database) CreateSettlement(settlement *Settlement) error {
	return d.db.Create(settlement).Error
}

func (d *Database) GetSettlement(settlementID string) (*Settlement, error) {
	var settlement Settlement
	if err := d.db.Where("settlement_id = ?", settlementID).First(&settlement).Error; err != nil {
		return nil, err
	}
	return &settlement, nil
}

func (d *Database) GetSettlementByTradeID(tradeID string) (*Settlement, error) {
	var settlement Settlement
	if err := d.db.Where("trade_id = ?", tradeID).First(&settlement).Error; err != nil {
		return nil, err
	}
	return &settlement, nil
}

func (d *Database) UpdateSettlement(settlement *Settlement) error {
	return d.db.Save(settlement).Error
}

func (d *Database) UpdateSettlementStatus(settlementID string, status string) error {
	result := d.db.Model(&Settlement{}).
		Where("settlement_id = ?", settlementID).
		Updates(map[string]interface{}{
			"settlement_status": status,
			"updated_at":       time.Now(),
		})
	
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return errors.New("settlement not found")
	}
	
	return nil
}

func (d *Database) GetPendingSettlements() ([]Settlement, error) {
	var settlements []Settlement
	if err := d.db.Where("settlement_status = ?", "PENDING").Find(&settlements).Error; err != nil {
		return nil, err
	}
	return settlements, nil
}

func (d *Database) GetClientSettlements(clientID string) ([]Settlement, error) {
	var settlements []Settlement
	if err := d.db.Where("client_id = ?", clientID).Order("created_at DESC").Find(&settlements).Error; err != nil {
		return nil, err
	}
	return settlements, nil
}

func (d *Database) GetSettlementsByDateRange(startDate, endDate time.Time) ([]Settlement, error) {
	var settlements []Settlement
	if err := d.db.Where("settlement_date BETWEEN ? AND ?", startDate, endDate).
		Order("settlement_date DESC").
		Find(&settlements).Error; err != nil {
		return nil, err
	}
	return settlements, nil
}

// GetExecutionByID retrieves execution details by ID
func (d *Database) GetExecutionByID(executionID string) (*types.Execution, error) {
	var execution types.Execution
	if err := d.db.Where("execution_id = ?", executionID).First(&execution).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch execution: %w", err)
	}
	return &execution, nil
}

// GetOrderByID retrieves order details by ID
func (d *Database) GetOrderByID(orderID string) (*types.Order, error) {
	var order types.Order
	if err := d.db.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	return &order, nil
}

// GetClearingByTradeID retrieves clearing details by trade ID
func (d *Database) GetClearingByTradeID(tradeID string) (*clearing.Clearing, error) {
	var clearingRecord clearing.Clearing
	if err := d.db.Where("trade_id = ?", tradeID).First(&clearingRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch clearing: %w", err)
	}
	return &clearingRecord, nil
} 