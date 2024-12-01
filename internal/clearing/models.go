package clearing

import (
	"time"

	"gorm.io/gorm"
)

type Clearing struct {
	gorm.Model       `json:"-"`
	ClearingID       string    `gorm:"uniqueIndex" json:"clearing_id"`
	TradeID          string    `json:"trade_id"`
	ClearingStatus   string    `json:"clearing_status"` // PENDING, CLEARED, FAILED
	MarginRequired   float64   `json:"margin_required"`
	NetPositions     float64   `json:"net_positions"`
	SettlementAmount float64   `json:"settlement_amount"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ClearingResponse struct {
	ClearingID       string    `json:"clearing_id"`
	ClearingStatus   string    `json:"clearing_status"`
	MarginRequired   float64   `json:"margin_required"`
	NetPositions     float64   `json:"net_positions"`
	SettlementAmount float64   `json:"settlement_amount"`
	Timestamp        time.Time `json:"timestamp"`
}

type TradeNetting struct {
	gorm.Model      `json:"-"`
	NettingID       string    `gorm:"uniqueIndex" json:"netting_id"`
	Symbol          string    `json:"symbol"`
	WindowStart     time.Time `json:"window_start"`
	WindowEnd       time.Time `json:"window_end"`
	NetQuantity     float64   `json:"net_quantity"`
	NetAmount       float64   `json:"net_amount"`
	NetSettlement   float64   `json:"net_settlement"`
	NetMargin       float64   `json:"net_margin"`
	Status          string    `json:"status"` // PENDING, COMPLETED, FAILED
	OriginalTrades  string    `json:"original_trades"` // JSON array of trade IDs
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
