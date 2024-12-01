package trading

import (
	"errors"
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

func (d *Database) CreateOrder(order *types.Order) error {
	return d.db.Create(order).Error
}

func (d *Database) GetOrder(orderID string) (*types.Order, error) {
	var order types.Order
	if err := d.db.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (d *Database) GetOrderByOrderIDAndClientID(orderID, clientID string) (*types.Order, error) {
	var order types.Order
	if err := d.db.Where("order_id = ? AND client_id = ?", orderID, clientID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (d *Database) UpdateOrder(order *types.Order) error {
	return d.db.Save(order).Error
}

func (d *Database) CreateExecution(execution *types.Execution) error {
	return d.db.Create(execution).Error
}

func (d *Database) GetExecution(executionID string) (*types.Execution, error) {
	var execution types.Execution
	if err := d.db.Where("execution_id = ?", executionID).First(&execution).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

func (d *Database) UpdateExecution(execution *types.Execution) error {
	return d.db.Save(execution).Error
}

// CreateOrderWithIdempotency creates a new order and idempotency record in a transaction
func (d *Database) CreateOrderWithIdempotency(order *types.Order, idempotencyKey string) error {
	// Begin transaction
	tx := d.db.Begin()
	if err := tx.Error; err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Create idempotency record
	record := IdempotencyRecord{
		IdempotencyKey: idempotencyKey,
		ResourceID:     order.OrderID,
		ResourceType:   "order",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// GetIdempotencyRecord retrieves an idempotency record by key
func (d *Database) GetIdempotencyRecord(key string) (*IdempotencyRecord, error) {
	var record IdempotencyRecord
	if err := d.db.Where("idempotency_key = ?", key).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &record, nil
		}
		return nil, err
	}
	return &record, nil
}

// CreateExecutionWithIdempotency creates a new execution and idempotency record in a transaction
func (d *Database) CreateExecutionWithIdempotency(execution *types.Execution, idempotencyKey string) error {
	// Begin transaction
	tx := d.db.Begin()
	if err := tx.Error; err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(execution).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Create idempotency record
	record := IdempotencyRecord{
		IdempotencyKey: idempotencyKey,
		ResourceID:     execution.ExecutionID,
		ResourceType:   "execution",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
