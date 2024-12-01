package trading

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

type Execution struct {
	gorm.Model  `json:"-"`
	ExecutionID string    `gorm:"uniqueIndex" json:"execution_id"`
	OrderID     string    `json:"order_id"`
	Price       float64   `json:"price"`
	Quantity    float64   `json:"quantity"`
	Side        string    `json:"side"`
	Status      string    `json:"status"` // PENDING, COMPLETED, FAILED
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type IdempotencyRecord struct {
	gorm.Model
	IdempotencyKey string    `gorm:"uniqueIndex" json:"idempotency_key"`
	ResourceID     string    `json:"resource_id"`
	ResourceType   string    `json:"resource_type"`
	ExpiresAt      time.Time `json:"expires_at"`
}
