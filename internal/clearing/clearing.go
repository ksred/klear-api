package clearing

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ksred/klear-api/internal/types"
	"github.com/ksred/klear-api/pkg/response"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Service handles trade clearing operations
type Service struct {
	db *Database
}

// NewService creates a new clearing service with the given database connection
func NewService(gormDB *gorm.DB) *Service {
	return &Service{
		db: NewDatabase(gormDB),
	}
}

const (
	StatusPending = "PENDING"
	StatusCleared = "CLEARED"
	StatusFailed  = "FAILED"
)

// ClearTrade handles the clearing process for a trade
// It performs trade netting, calculates margins, and validates clearing rules
// Parameters:
//   - tradeID: ID of the trade to clear
func (s *Service) ClearTrade(tradeID string) (*ClearingResponse, error) {
	logger := log.With().
		Str("trade_id", tradeID).
		Str("service", "clearing").
		Logger()

	logger.Info().Msg("starting clearing process for trade")

	// Create initial clearing record with PENDING status
	clearing := &Clearing{
		ClearingID:     "CLR_" + uuid.New().String(),
		TradeID:        tradeID,
		ClearingStatus: StatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	logger.Debug().
		Str("clearing_id", clearing.ClearingID).
		Str("status", clearing.ClearingStatus).
		Msg("created clearing record")

	// Get execution details
	execution, err := s.db.GetExecutionByID(tradeID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch execution details")
		return nil, fmt.Errorf("failed to fetch execution details: %w", err)
	}

	logger.Debug().
		Str("execution_id", execution.ExecutionID).
		Str("order_id", execution.OrderID).
		Float64("total_quantity", execution.TotalQuantity).
		Float64("average_price", execution.AveragePrice).
		Msg("fetched execution details")

	// Get order details
	order, err := s.db.GetOrderByID(execution.OrderID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch order details")
		return nil, fmt.Errorf("failed to fetch order details: %w", err)
	}

	logger.Debug().
		Str("order_id", order.OrderID).
		Str("client_id", order.ClientID).
		Str("symbol", order.Symbol).
		Str("side", order.Side).
		Float64("quantity", order.Quantity).
		Msg("fetched order details")

	// Perform trade netting
	nettingResult, err := s.calculateTradeNetting(execution, order)
	if err != nil {
		logger.Error().Err(err).Msg("netting calculation failed")
		clearing.ClearingStatus = StatusFailed
		if err := s.db.CreateClearing(clearing); err != nil {
			logger.Error().Err(err).Msg("failed to save failed clearing record")
			return nil, err
		}
		return nil, fmt.Errorf("netting calculation failed: %w", err)
	}

	logger.Info().
		Float64("net_quantity", nettingResult.NetQuantity).
		Float64("net_amount", nettingResult.NetAmount).
		Float64("net_settlement", nettingResult.NetSettlement).
		Float64("net_margin", nettingResult.NetMargin).
		Int("trades_netted", len(nettingResult.OriginalTrades)).
		Msg("completed trade netting calculation")

	// Update clearing with netted values
	clearing.NetPositions = nettingResult.NetQuantity
	clearing.SettlementAmount = nettingResult.NetSettlement
	clearing.MarginRequired = nettingResult.NetMargin

	// Process clearing calculations and validation
	if err := s.processClearingCalculations(clearing, execution, order); err != nil {
		logger.Error().Err(err).Msg("clearing calculations failed")
		clearing.ClearingStatus = StatusFailed
		if err := s.db.CreateClearing(clearing); err != nil {
			logger.Error().Err(err).Msg("failed to save failed clearing record")
			return nil, err
		}
		return nil, err
	}

	clearing.ClearingStatus = StatusCleared

	// Save both netting result and clearing in a transaction
	if err := s.db.SaveNettingResult(nettingResult, clearing); err != nil {
		logger.Error().Err(err).Msg("failed to save netting and clearing results")
		return nil, fmt.Errorf("failed to save netting and clearing results: %w", err)
	}

	logger.Info().
		Str("clearing_id", clearing.ClearingID).
		Str("status", clearing.ClearingStatus).
		Float64("margin_required", clearing.MarginRequired).
		Float64("net_positions", clearing.NetPositions).
		Float64("settlement_amount", clearing.SettlementAmount).
		Msg("clearing process completed successfully")

	return &ClearingResponse{
		ClearingID:       clearing.ClearingID,
		ClearingStatus:   clearing.ClearingStatus,
		MarginRequired:   clearing.MarginRequired,
		NetPositions:     clearing.NetPositions,
		SettlementAmount: clearing.SettlementAmount,
		Timestamp:        time.Now(),
	}, nil
}

// calculateTradeNetting performs multilateral netting for trades
// Groups trades by symbol within the netting window and calculates net positions
func (s *Service) calculateTradeNetting(execution *types.Execution, order *types.Order) (*TradeNetting, error) {
	logger := log.With().
		Str("execution_id", execution.ExecutionID).
		Str("symbol", order.Symbol).
		Str("service", "clearing").
		Logger()

	logger.Info().Msg("starting trade netting calculation")

	// Get all trades for the same symbol within the netting window
	nettingWindowStart := time.Now().Add(-24 * time.Hour)
	executions, err := s.db.GetTradesForNetting(order.Symbol, nettingWindowStart)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch trades for netting")
		return nil, err
	}

	logger.Debug().
		Int("trades_found", len(executions)).
		Time("window_start", nettingWindowStart).
		Msg("fetched trades for netting window")

	// Get all related orders in a single query
	orderMap, err := s.db.GetOrdersForExecutions(executions)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch orders for executions")
		return nil, err
	}

	// Initialize netting result
	netting := &TradeNetting{
		NettingID:      "NET_" + uuid.New().String(),
		Symbol:         order.Symbol,
		WindowStart:    nettingWindowStart,
		WindowEnd:      time.Now(),
		Status:         "PENDING",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		OriginalTrades: "[]", // Will be updated with JSON array
	}

	logger.Debug().
		Str("netting_id", netting.NettingID).
		Time("window_start", netting.WindowStart).
		Time("window_end", netting.WindowEnd).
		Msg("initialized netting record")

	// Process all trades for multilateral netting
	tradeIDs := make([]string, 0, len(executions))
	for _, exec := range executions {
		ord, exists := orderMap[exec.OrderID]
		if !exists {
			logger.Error().
				Str("execution_id", exec.ExecutionID).
				Str("order_id", exec.OrderID).
				Msg("order not found for execution")
			return nil, fmt.Errorf("order not found for execution %s", exec.ExecutionID)
		}

		tradeIDs = append(tradeIDs, exec.ExecutionID)
		if ord.Side == "BUY" {
			netting.NetQuantity += exec.TotalQuantity
			netting.NetAmount += exec.TotalQuantity * exec.AveragePrice
			logger.Debug().
				Str("execution_id", exec.ExecutionID).
				Float64("quantity", exec.TotalQuantity).
				Float64("amount", exec.TotalQuantity*exec.AveragePrice).
				Msg("added buy trade to netting")
		} else {
			netting.NetQuantity -= exec.TotalQuantity
			netting.NetAmount -= exec.TotalQuantity * exec.AveragePrice
			logger.Debug().
				Str("execution_id", exec.ExecutionID).
				Float64("quantity", -exec.TotalQuantity).
				Float64("amount", -exec.TotalQuantity*exec.AveragePrice).
				Msg("added sell trade to netting")
		}
	}

	// Convert trade IDs to JSON string
	tradeIDsJSON, err := json.Marshal(tradeIDs)
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal trade IDs")
		return nil, fmt.Errorf("failed to marshal trade IDs: %w", err)
	}
	netting.OriginalTrades = string(tradeIDsJSON)

	// Calculate net settlement and margin
	netting.NetSettlement = math.Abs(netting.NetAmount)

	// Calculate margin based on net market exposure
	const (
		baseMarginRate = 0.10 // 10% base margin requirement
		// Additional margin rates based on market conditions
		marketVolatilityMultiplier = 1.2  // 20% extra for volatile markets
		concentrationMultiplier    = 1.15 // 15% extra for concentrated positions
	)

	// Start with base margin
	netting.NetMargin = netting.NetSettlement * baseMarginRate
	logger.Debug().
		Float64("base_margin", netting.NetMargin).
		Float64("base_rate", baseMarginRate).
		Msg("calculated base margin")

	// Apply market volatility multiplier
	netting.NetMargin *= marketVolatilityMultiplier
	logger.Debug().
		Float64("adjusted_margin", netting.NetMargin).
		Float64("volatility_multiplier", marketVolatilityMultiplier).
		Msg("applied volatility multiplier")

	// Apply concentration multiplier if net position is large
	if math.Abs(netting.NetQuantity) > 1000 {
		netting.NetMargin *= concentrationMultiplier
		logger.Debug().
			Float64("final_margin", netting.NetMargin).
			Float64("concentration_multiplier", concentrationMultiplier).
			Msg("applied concentration multiplier")
	}

	netting.Status = "COMPLETED"
	logger.Info().
		Float64("net_quantity", netting.NetQuantity).
		Float64("net_amount", netting.NetAmount).
		Float64("net_settlement", netting.NetSettlement).
		Float64("net_margin", netting.NetMargin).
		Int("total_trades", len(tradeIDs)).
		Msg("completed netting calculations")

	return netting, nil
}

// processClearingCalculations performs the core clearing calculations
func (s *Service) processClearingCalculations(clearing *Clearing, execution *types.Execution, order *types.Order) error {
	// Calculate settlement amount based on actual execution price and quantity
	clearing.SettlementAmount = execution.AveragePrice * execution.TotalQuantity

	// Calculate net positions
	positionMultiplier := 1.0
	if execution.Side == "SELL" {
		positionMultiplier = -1.0
	}
	clearing.NetPositions = execution.TotalQuantity * positionMultiplier

	// Validate the clearing
	if err := s.validateClearing(clearing, order); err != nil {
		return fmt.Errorf("clearing validation failed: %w", err)
	}

	return nil
}

// validateClearing performs validation checks on the clearing
// Verifies position limits, margin requirements, and risk thresholds
func (s *Service) validateClearing(clearing *Clearing, order *types.Order) error {
	logger := log.With().
		Str("clearing_id", clearing.ClearingID).
		Str("order_id", order.OrderID).
		Str("client_id", order.ClientID).
		Str("service", "clearing").
		Logger()

	logger.Info().Msg("starting clearing validation")

	// Mock client risk limits
	const (
		maxDailyNetPosition  = 1000000.0 // $1M max daily net position
		maxMarginUtilization = 0.80      // 80% max margin utilization
		availableMargin      = 1000000.0 // $1M available margin (should come from client config)
		positionLimit        = 500000.0  // $500K position limit per trade
		dailyTradingLimit    = 5000000.0 // $5M daily trading limit
	)

	logger.Debug().
		Float64("max_daily_net_position", maxDailyNetPosition).
		Float64("max_margin_utilization", maxMarginUtilization).
		Float64("available_margin", availableMargin).
		Float64("position_limit", positionLimit).
		Float64("daily_trading_limit", dailyTradingLimit).
		Msg("using risk limits")

	// Ensure settlement amount is positive and within limits
	if clearing.SettlementAmount <= 0 {
		logger.Error().
			Float64("settlement_amount", clearing.SettlementAmount).
			Msg("invalid settlement amount")
		return errors.New("invalid settlement amount")
	}
	if clearing.SettlementAmount > positionLimit {
		logger.Error().
			Float64("settlement_amount", clearing.SettlementAmount).
			Float64("position_limit", positionLimit).
			Msg("settlement amount exceeds position limit")
		return fmt.Errorf("settlement amount %f exceeds position limit of %f",
			clearing.SettlementAmount, positionLimit)
	}

	logger.Debug().
		Float64("settlement_amount", clearing.SettlementAmount).
		Msg("settlement amount validation passed")

	// Ensure margin required is positive and within client's available margin
	if clearing.MarginRequired <= 0 {
		logger.Error().
			Float64("margin_required", clearing.MarginRequired).
			Msg("invalid margin requirement")
		return errors.New("invalid margin requirement")
	}
	marginUtilization := clearing.MarginRequired / availableMargin
	if marginUtilization > maxMarginUtilization {
		logger.Error().
			Float64("margin_utilization", marginUtilization).
			Float64("max_margin_utilization", maxMarginUtilization).
			Float64("margin_required", clearing.MarginRequired).
			Float64("available_margin", availableMargin).
			Msg("margin utilization exceeds maximum allowed")
		return fmt.Errorf("margin utilization %f exceeds maximum allowed %f",
			marginUtilization, maxMarginUtilization)
	}

	logger.Debug().
		Float64("margin_required", clearing.MarginRequired).
		Float64("margin_utilization", marginUtilization).
		Msg("margin validation passed")

	// Get current day's net position
	currentDayNetPosition, err := s.db.GetDailyNetPosition(order.ClientID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get daily net position")
		return fmt.Errorf("failed to get daily net position: %w", err)
	}

	projectedNetPosition := math.Abs(currentDayNetPosition + clearing.NetPositions)
	if projectedNetPosition > maxDailyNetPosition {
		logger.Error().
			Float64("projected_net_position", projectedNetPosition).
			Float64("max_daily_net_position", maxDailyNetPosition).
			Float64("current_net_position", currentDayNetPosition).
			Float64("new_position", clearing.NetPositions).
			Msg("projected net position would exceed daily limit")
		return fmt.Errorf("projected net position %f would exceed daily limit of %f",
			projectedNetPosition, maxDailyNetPosition)
	}

	logger.Debug().
		Float64("current_net_position", currentDayNetPosition).
		Float64("projected_net_position", projectedNetPosition).
		Msg("position limit validation passed")

	// Get current day's trading volume
	currentDayVolume, err := s.db.GetDailyTradingVolume(order.ClientID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get daily trading volume")
		return fmt.Errorf("failed to get daily trading volume: %w", err)
	}

	projectedDailyVolume := currentDayVolume + clearing.SettlementAmount
	if projectedDailyVolume > dailyTradingLimit {
		logger.Error().
			Float64("projected_daily_volume", projectedDailyVolume).
			Float64("daily_trading_limit", dailyTradingLimit).
			Float64("current_volume", currentDayVolume).
			Float64("new_volume", clearing.SettlementAmount).
			Msg("projected daily volume would exceed limit")
		return fmt.Errorf("projected daily volume %f would exceed limit of %f",
			projectedDailyVolume, dailyTradingLimit)
	}

	logger.Debug().
		Float64("current_volume", currentDayVolume).
		Float64("projected_volume", projectedDailyVolume).
		Msg("volume limit validation passed")

	// Validate trade timing (mock market hours check). Using large values for testing
	now := time.Now()
	marketOpen := time.Date(now.Year(), now.Month(), now.Day(), 1, 30, 0, 0, time.Local)  // 9:30 AM
	marketClose := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, time.Local) // 4:00 PM

	logger.Debug().
		Time("current_time", now).
		Time("market_open", marketOpen).
		Time("market_close", marketClose).
		Msg("checking market hours")

	if now.Before(marketOpen) || now.After(marketClose) {
		logger.Error().
			Time("current_time", now).
			Time("market_open", marketOpen).
			Time("market_close", marketClose).
			Msg("clearing attempted outside market hours")
		return errors.New("clearing can only be processed during market hours")
	}

	// Mock risk scoring
	riskScore := s.calculateMockRiskScore(clearing, order)
	if riskScore > 0.8 { // 80% risk threshold
		logger.Error().
			Float64("risk_score", riskScore).
			Float64("risk_threshold", 0.8).
			Msg("risk score exceeds acceptable threshold")
		return fmt.Errorf("risk score %f exceeds acceptable threshold", riskScore)
	}

	logger.Debug().
		Float64("risk_score", riskScore).
		Msg("risk score validation passed")

	logger.Info().Msg("clearing validation completed successfully")
	return nil
}

// calculateMockRiskScore calculates a simple mock risk score between 0 and 1
func (s *Service) calculateMockRiskScore(clearing *Clearing, order *types.Order) float64 {
	// Mock factors for risk calculation
	const (
		positionFactor   = 0.4  // 40% weight for position size
		marginFactor     = 0.3  // 30% weight for margin utilization
		volatilityFactor = 0.3  // 30% weight for market volatility
		baseVolatility   = 0.15 // 15% base market volatility
	)

	// Position size risk (larger positions = higher risk)
	positionRisk := math.Min(math.Abs(clearing.NetPositions)/1000000.0, 1.0) // Normalized to 1M

	// Margin utilization risk
	marginRisk := clearing.MarginRequired / 1000000.0 // Normalized to 1M

	// Mock volatility risk (in reality, this would come from market data)
	mockVolatility := baseVolatility
	if order.OrderType == "MARKET" {
		mockVolatility *= 1.2 // 20% higher risk for market orders
	}

	// Calculate weighted risk score
	riskScore := (positionRisk * positionFactor) +
		(marginRisk * marginFactor) +
		(mockVolatility * volatilityFactor)

	return math.Min(riskScore, 1.0) // Ensure score is between 0 and 1
}

// GetClearingStatus retrieves the current status of a clearing
func (s *Service) GetClearingStatus(clearingID string) (*ClearingResponse, error) {
	clearing, err := s.db.GetClearing(clearingID)
	if err != nil {
		return nil, err
	}

	return &ClearingResponse{
		ClearingID:       clearing.ClearingID,
		ClearingStatus:   clearing.ClearingStatus,
		MarginRequired:   clearing.MarginRequired,
		NetPositions:     clearing.NetPositions,
		SettlementAmount: clearing.SettlementAmount,
		Timestamp:        clearing.UpdatedAt,
	}, nil
}

// GinHandlers contains HTTP handlers for clearing endpoints
type GinHandlers struct {
	service *Service
}

// NewGinHandlers creates a new set of HTTP handlers for clearing endpoints
func NewGinHandlers(service *Service) *GinHandlers {
	return &GinHandlers{
		service: service,
	}
}

// ClearTradeHandler handles POST requests to clear trades
// Requires internal authentication
// URL parameter: trade_id
func (h *GinHandlers) ClearTradeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		tradeID := c.Param("trade_id")

		clearingResponse, err := h.service.ClearTrade(tradeID)
		response.Handle(c, clearingResponse, err)
	}
}

func (h *GinHandlers) GetClearingStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clearingID := c.Param("clearing_id")

		clearingResponse, err := h.service.GetClearingStatus(clearingID)
		response.Handle(c, clearingResponse, err)
	}
}
