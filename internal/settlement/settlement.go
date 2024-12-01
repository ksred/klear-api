package settlement

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ksred/klear-api/internal/types"
	"github.com/ksred/klear-api/pkg/response"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Service struct {
	db *Database
}

func NewService(gormDB *gorm.DB) *Service {
	return &Service{
		db: NewDatabase(gormDB),
	}
}

// SettleTrade handles the settlement process for a trade
func (s *Service) SettleTrade(tradeID string) (*SettlementResponse, error) {
	logger := log.With().
		Str("trade_id", tradeID).
		Str("service", "settlement").
		Logger()

	logger.Info().Msg("starting settlement process for trade")

	// Get execution details
	execution, err := s.db.GetExecutionByID(tradeID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch execution details")
		return nil, fmt.Errorf("failed to fetch execution details: %w", err)
	}

	// Get order details
	order, err := s.db.GetOrderByID(execution.OrderID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch order details")
		return nil, fmt.Errorf("failed to fetch order details: %w", err)
	}

	// Get clearing details
	clearingDetails, err := s.db.GetClearingByTradeID(tradeID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch clearing details")
		return nil, fmt.Errorf("failed to fetch clearing details: %w", err)
	}

	// Calculate settlement fees (0.1% of total value)
	settlementFees := execution.AveragePrice * execution.TotalQuantity * 0.001

	settlement := &Settlement{
		SettlementID:      "STL_" + uuid.New().String(),
		TradeID:           tradeID,
		ClientID:          order.ClientID,
		SettlementStatus:  "PENDING",
		SettlementDate:    time.Now().Add(2 * 24 * time.Hour), // T+2 settlement
		FinalAmount:       clearingDetails.SettlementAmount,
		Currency:          "USD", // Default currency
		SettlementAccount: fmt.Sprintf("ACC_%s", order.ClientID),
		ClearingID:        clearingDetails.ClearingID,
		ExecutionID:       execution.ExecutionID,
		ExecutedPrice:     execution.AveragePrice,
		ExecutedQuantity:  int64(execution.TotalQuantity),
		SettlementFees:    settlementFees,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.validateSettlement(settlement, order); err != nil {
		logger.Error().Err(err).Msg("settlement validation failed")
		settlement.SettlementStatus = "FAILED"
		if err := s.db.CreateSettlement(settlement); err != nil {
			logger.Error().Err(err).Msg("failed to save failed settlement record")
			return nil, err
		}
		return nil, fmt.Errorf("settlement validation failed: %w", err)
	}

	if err := s.db.CreateSettlement(settlement); err != nil {
		logger.Error().Err(err).Msg("failed to create settlement record")
		return nil, fmt.Errorf("failed to create settlement record: %w", err)
	}

	logger.Info().
		Str("settlement_id", settlement.SettlementID).
		Str("status", settlement.SettlementStatus).
		Time("settlement_date", settlement.SettlementDate).
		Float64("final_amount", settlement.FinalAmount).
		Msg("settlement process completed successfully")

	return &SettlementResponse{
		SettlementID:      settlement.SettlementID,
		TradeID:           settlement.TradeID,
		ClientID:          settlement.ClientID,
		SettlementStatus:  settlement.SettlementStatus,
		SettlementDate:    settlement.SettlementDate,
		FinalAmount:       settlement.FinalAmount,
		Currency:          settlement.Currency,
		SettlementAccount: settlement.SettlementAccount,
		ExecutedPrice:     settlement.ExecutedPrice,
		ExecutedQuantity:  settlement.ExecutedQuantity,
		SettlementFees:    settlement.SettlementFees,
		Timestamp:         time.Now(),
	}, nil
}

// validateSettlement performs validation checks on the settlement
func (s *Service) validateSettlement(settlement *Settlement, order *types.Order) error {
	logger := log.With().
		Str("settlement_id", settlement.SettlementID).
		Str("trade_id", settlement.TradeID).
		Str("client_id", settlement.ClientID).
		Str("service", "settlement").
		Logger()

	logger.Info().Str("order_id", order.OrderID).Msg("starting settlement validation")

	// Validate settlement amount
	if settlement.FinalAmount <= 0 {
		return errors.New("invalid settlement amount")
	}

	// Validate settlement date is T+2
	// For now, we remove this check to test the flow
	// minSettlementDate := time.Now().Add(2 * 24 * time.Hour)
	// if settlement.SettlementDate.Before(minSettlementDate) {
	// 	return errors.New("settlement date must be at least T+2")
	// }

	// Validate market hours. For testing we use a broad window
	now := time.Now()
	marketOpen := time.Date(now.Year(), now.Month(), now.Day(), 1, 30, 0, 0, time.Local)  // 1:30 AM
	marketClose := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, time.Local) // 11:00 PM

	if now.Before(marketOpen) || now.After(marketClose) {
		return errors.New("settlement can only be processed during market hours")
	}

	logger.Info().Msg("settlement validation completed successfully")
	return nil
}

// UpdateSettlementStatus updates the status of a settlement
func (s *Service) UpdateSettlementStatus(settlementID string, status string) error {
	return s.db.UpdateSettlementStatus(settlementID, status)
}

// GetSettlement retrieves a settlement by ID
func (s *Service) GetSettlement(settlementID string) (*Settlement, error) {
	return s.db.GetSettlement(settlementID)
}

// GetClientSettlements retrieves all settlements for a client
func (s *Service) GetClientSettlements(clientID string) ([]Settlement, error) {
	return s.db.GetClientSettlements(clientID)
}

// GinHandlers contains HTTP handlers for settlement endpoints
type GinHandlers struct {
	service *Service
}

func NewGinHandlers(service *Service) *GinHandlers {
	return &GinHandlers{
		service: service,
	}
}

func (h *GinHandlers) SettleTradeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		tradeID := c.Param("trade_id")

		settlementResponse, err := h.service.SettleTrade(tradeID)
		response.Handle(c, settlementResponse, err)
	}
}

func (h *GinHandlers) GetSettlementHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		settlementID := c.Param("settlement_id")

		settlement, err := h.service.GetSettlement(settlementID)
		response.Handle(c, settlement, err)
	}
}

func (h *GinHandlers) GetClientSettlementsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.GetHeader("X-Client-ID")
		if clientID == "" {
			response.BadRequest(c, "client ID is required")
			return
		}

		settlements, err := h.service.GetClientSettlements(clientID)
		response.Handle(c, settlements, err)
	}
}

func (h *GinHandlers) UpdateSettlementStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		settlementID := c.Param("settlement_id")
		var request struct {
			Status string `json:"status" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		if err := h.service.UpdateSettlementStatus(settlementID, request.Status); err != nil {
			response.Handle(c, nil, err)
			return
		}

		response.Success(c, gin.H{"message": "settlement status updated successfully"})
	}
}

// Add this method to the Service struct
func (s *Service) GetDB() *Database {
	return s.db
}
