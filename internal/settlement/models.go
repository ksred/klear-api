package settlement

import (
	"time"

	"gorm.io/gorm"
)

type Settlement struct {
	gorm.Model       `json:"-"`
	SettlementID     string    `gorm:"uniqueIndex" json:"settlement_id"`
	TradeID          string    `json:"trade_id"`
	ClientID         string    `json:"client_id"`
	SettlementStatus string    `json:"settlement_status"` // PENDING, SETTLING, SETTLED, FAILED
	SettlementDate   time.Time `json:"settlement_date"`
	FinalAmount      float64   `json:"final_amount"`
	Currency         string    `json:"currency"`
	SettlementAccount string   `json:"settlement_account"`
	ClearingID       string    `json:"clearing_id"`
	ExecutionID      string    `json:"execution_id"`
	ExecutedPrice    float64   `json:"executed_price"`
	ExecutedQuantity int64     `json:"executed_quantity"`
	SettlementFees   float64   `json:"settlement_fees"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type SettlementResponse struct {
	SettlementID     string    `json:"settlement_id"`
	TradeID          string    `json:"trade_id"`
	ClientID         string    `json:"client_id"`
	SettlementStatus string    `json:"settlement_status"`
	SettlementDate   time.Time `json:"settlement_date"`
	FinalAmount      float64   `json:"final_amount"`
	Currency         string    `json:"currency"`
	SettlementAccount string   `json:"settlement_account"`
	ExecutedPrice    float64   `json:"executed_price"`
	ExecutedQuantity int64     `json:"executed_quantity"`
	SettlementFees   float64   `json:"settlement_fees"`
	Timestamp        time.Time `json:"timestamp"`
}

// Mock request/response structures for integration
type ClearingDetails struct {
	ClearingID       string    `json:"clearing_id"`
	ClearingStatus   string    `json:"clearing_status"`
	MarginRequired   float64   `json:"margin_required"`
	NetPositions     float64   `json:"net_positions"`
	SettlementAmount float64   `json:"settlement_amount"`
}

type ExecutionDetails struct {
	ExecutionID      string    `json:"execution_id"`
	ExecutedPrice    float64   `json:"executed_price"`
	ExecutedQuantity int64     `json:"executed_quantity"`
	Timestamp        time.Time `json:"timestamp"`
	ExchangeID       string    `json:"exchange_id"`
	ExecutionFees    float64   `json:"execution_fees"`
}
