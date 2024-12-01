package types

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	gorm.Model `json:"-"`
	OrderID    string    `gorm:"uniqueIndex" json:"order_id"`
	ClientID   string    `json:"client_id"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`       // BUY or SELL
	OrderType  string    `json:"order_type"` // MARKET or LIMIT
	Quantity   float64   `json:"quantity"`
	Price      float64   `json:"price"`
	Status     string    `json:"status"` // PENDING, FILLED, CANCELLED
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ExchangeFill struct {
	gorm.Model   `json:"-"`
	FillID       string    `gorm:"uniqueIndex" json:"fill_id"`
	ExecutionID  string    `json:"execution_id"`
	ExchangeID   string    `json:"exchange_id"`
	ExchangeName string    `json:"exchange_name"`
	Price        float64   `json:"price"`
	Quantity     float64   `json:"quantity"`
	FeeRate      float64   `json:"fee_rate"`
	FeeAmount    float64   `json:"fee_amount"`
	CreatedAt    time.Time `json:"created_at"`
}

type Execution struct {
	gorm.Model    `json:"-"`
	ExecutionID   string         `gorm:"uniqueIndex" json:"execution_id"`
	OrderID       string         `json:"order_id"`
	TotalQuantity float64       `json:"total_quantity"`
	AveragePrice  float64       `json:"average_price"`
	Side          string        `json:"side"`
	Status        string        `json:"status"` // PENDING, COMPLETED, FAILED
	Fills         []ExchangeFill `json:"fills,omitempty" gorm:"foreignKey:ExecutionID"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
} 