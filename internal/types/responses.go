package types

import "time"

// ClearingResponse represents the response from the clearing service
type ClearingResponse struct {
	ClearingID       string    `json:"clearing_id"`
	TradeID          string    `json:"trade_id"`
	ClearingStatus   string    `json:"clearing_status"`
	MarginRequired   float64   `json:"margin_required"`
	NetPositions     float64   `json:"net_positions"`
	SettlementAmount float64   `json:"settlement_amount"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// SettlementResponse represents the response from the settlement service
type SettlementResponse struct {
	SettlementID      string    `json:"settlement_id"`
	TradeID           string    `json:"trade_id"`
	SettlementStatus  string    `json:"settlement_status"`
	FinalAmount       float64   `json:"final_amount"`
	SettlementDate    time.Time `json:"settlement_date"`
	SettlementAccount string    `json:"settlement_account"`
	Currency          string    `json:"currency"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
} 