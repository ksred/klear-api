package clearing

import (
	"fmt"
	"time"

	"github.com/ksred/klear-api/internal/types"
	"gorm.io/gorm"
)

type Database struct {
	db *gorm.DB
}

func NewDatabase(db *gorm.DB) *Database {
	return &Database{db: db}
}

// CreateClearing creates a new clearing record
func (d *Database) CreateClearing(clearing *Clearing) error {
	return d.db.Create(clearing).Error
}

func (d *Database) GetClearing(clearingID string) (*Clearing, error) {
	var clearing Clearing
	if err := d.db.Where("clearing_id = ?", clearingID).First(&clearing).Error; err != nil {
		return nil, err
	}
	return &clearing, nil
}

func (d *Database) UpdateClearing(clearing *Clearing) error {
	return d.db.Save(clearing).Error
}

// CreateTradeNetting creates a new trade netting record
func (d *Database) CreateTradeNetting(netting *TradeNetting) error {
	return d.db.Create(netting).Error
}

// GetTradeNetting retrieves a trade netting record by ID
func (d *Database) GetTradeNetting(nettingID string) (*TradeNetting, error) {
	var netting TradeNetting
	if err := d.db.Where("netting_id = ?", nettingID).First(&netting).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch netting record: %w", err)
	}
	return &netting, nil
}

// UpdateTradeNetting updates a trade netting record
func (d *Database) UpdateTradeNetting(netting *TradeNetting) error {
	return d.db.Save(netting).Error
}

// GetLatestNettingBySymbol retrieves the latest netting record for a symbol
func (d *Database) GetLatestNettingBySymbol(symbol string) (*TradeNetting, error) {
	var netting TradeNetting
	if err := d.db.Where("symbol = ?", symbol).
		Order("created_at DESC").
		First(&netting).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch latest netting for symbol: %w", err)
	}
	return &netting, nil
}

// GetNettingsByTimeWindow retrieves all netting records within a time window
func (d *Database) GetNettingsByTimeWindow(start, end time.Time) ([]TradeNetting, error) {
	var nettings []TradeNetting
	if err := d.db.Where("window_start >= ? AND window_end <= ?", start, end).
		Order("created_at DESC").
		Find(&nettings).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch nettings for time window: %w", err)
	}
	return nettings, nil
}

// SaveNettingResult saves the netting result in a transaction
func (d *Database) SaveNettingResult(netting *TradeNetting, clearing *Clearing) error {
	// Start transaction
	tx := d.db.Begin()
	if err := tx.Error; err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Save netting record
	if err := tx.Create(netting).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save netting record: %w", err)
	}

	// Update clearing record
	if err := tx.Save(clearing).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update clearing record: %w", err)
	}

	return tx.Commit().Error
}

// GetExecutionByID retrieves an execution by its ID
func (d *Database) GetExecutionByID(executionID string) (*types.Execution, error) {
	var execution types.Execution
	if err := d.db.Where("execution_id = ?", executionID).First(&execution).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch execution: %w", err)
	}
	return &execution, nil
}

// GetOrderByID retrieves an order by its ID
func (d *Database) GetOrderByID(orderID string) (*types.Order, error) {
	var order types.Order
	if err := d.db.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	return &order, nil
}

// GetTradesForNetting retrieves all trades within the netting window for a given symbol
func (d *Database) GetTradesForNetting(symbol string, windowStart time.Time) ([]types.Execution, error) {
	var executions []types.Execution
	if err := d.db.
		Joins("JOIN orders ON orders.order_id = executions.order_id").
		Where("orders.symbol = ? AND executions.created_at > ?", symbol, windowStart).
		Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch trades for netting: %w", err)
	}
	return executions, nil
}

// GetOrdersForExecutions retrieves orders for a list of executions
func (d *Database) GetOrdersForExecutions(executions []types.Execution) (map[string]types.Order, error) {
	orderMap := make(map[string]types.Order)

	// Extract order IDs
	var orderIDs []string
	for _, exec := range executions {
		orderIDs = append(orderIDs, exec.OrderID)
	}

	// Fetch all orders in a single query
	var orders []types.Order
	if err := d.db.Where("order_id IN ?", orderIDs).Find(&orders).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch orders: %w", err)
	}

	// Create map for easy lookup
	for _, order := range orders {
		orderMap[order.OrderID] = order
	}

	return orderMap, nil
}

// GetDailyNetPosition retrieves the current day's net position for a client
func (d *Database) GetDailyNetPosition(clientID string) (float64, error) {
	var netPosition float64

	// Get start of day in UTC
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Query to calculate net position from executions and orders
	query := `
		SELECT COALESCE(SUM(
			CASE 
				WHEN orders.side = 'BUY' THEN executions.total_quantity 
				WHEN orders.side = 'SELL' THEN -executions.total_quantity
				ELSE 0 
			END
		), 0) as net_position
		FROM executions
		JOIN orders ON orders.order_id = executions.order_id
		WHERE orders.client_id = ?
		AND executions.created_at >= ?
		AND executions.created_at < ?
		AND executions.status = 'COMPLETED'`

	if err := d.db.Raw(query, clientID, startOfDay, endOfDay).Scan(&netPosition).Error; err != nil {
		return 0, fmt.Errorf("failed to calculate daily net position: %w", err)
	}

	return netPosition, nil
}

// GetDailyTradingVolume retrieves the current day's trading volume for a client
func (d *Database) GetDailyTradingVolume(clientID string) (float64, error) {
	var totalVolume float64

	// Get start of day in UTC
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Query to calculate total trading volume (sum of all trades regardless of side)
	query := `
		SELECT COALESCE(SUM(executions.total_quantity * executions.average_price), 0) as total_volume
		FROM executions
		JOIN orders ON orders.order_id = executions.order_id
		WHERE orders.client_id = ?
		AND executions.created_at >= ?
		AND executions.created_at < ?
		AND executions.status = 'COMPLETED'`

	if err := d.db.Raw(query, clientID, startOfDay, endOfDay).Scan(&totalVolume).Error; err != nil {
		return 0, fmt.Errorf("failed to calculate daily trading volume: %w", err)
	}

	return totalVolume, nil
}

// GetDailyTradingStats retrieves both net position and volume in a single query for efficiency
func (d *Database) GetDailyTradingStats(clientID string) (netPosition, tradingVolume float64, err error) {
	type Result struct {
		NetPosition   float64
		TradingVolume float64
	}
	var result Result

	// Get start of day in UTC
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Combined query to get both stats in one go
	query := `
		SELECT 
			COALESCE(SUM(
				CASE 
					WHEN orders.side = 'BUY' THEN executions.total_quantity 
					WHEN orders.side = 'SELL' THEN -executions.total_quantity
					ELSE 0 
				END
			), 0) as net_position,
			COALESCE(SUM(executions.total_quantity * executions.average_price), 0) as trading_volume
		FROM executions
		JOIN orders ON orders.order_id = executions.order_id
		WHERE orders.client_id = ?
		AND executions.created_at >= ?
		AND executions.created_at < ?
		AND executions.status = 'COMPLETED'`

	if err := d.db.Raw(query, clientID, startOfDay, endOfDay).Scan(&result).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to calculate daily trading stats: %w", err)
	}

	return result.NetPosition, result.TradingVolume, nil
}
