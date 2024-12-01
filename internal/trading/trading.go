package trading

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ksred/klear-api/internal/exchange"
	"github.com/ksred/klear-api/internal/types"
	"github.com/ksred/klear-api/pkg/response"
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

// CreateOrder creates a new order with idempotency support
func (s *Service) CreateOrder(order *types.Order, idempotencyKey string) error {
	// Check for existing idempotency record
	record, err := s.db.GetIdempotencyRecord(idempotencyKey)

	// If record exists and hasn't expired
	if err == nil && record.ExpiresAt.After(time.Now()) {
		// Return existing order
		existingOrder, err := s.db.GetOrder(record.ResourceID)
		if err != nil {
			return err
		}
		*order = *existingOrder
		return nil
	}

	// Prepare new order
	order.OrderID = uuid.New().String()
	order.Status = "PENDING"
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	return s.db.CreateOrderWithIdempotency(order, idempotencyKey)
}

// GetOrder retrieves an order by ID
func (s *Service) GetOrder(orderID string) (*types.Order, error) {
	return s.db.GetOrder(orderID)
}

// ExecuteOrder executes an existing order with idempotency support
func (s *Service) ExecuteOrder(orderID string, idempotencyKey string) (*types.Execution, error) {
	// Check for existing idempotency record
	record, err := s.db.GetIdempotencyRecord(idempotencyKey)

	// If record exists and hasn't expired
	if err == nil && record.ExpiresAt.After(time.Now()) {
		// Return existing execution
		existingExecution, err := s.db.GetExecution(record.ResourceID)
		if err != nil {
			return nil, err
		}
		return existingExecution, nil
	}

	order, err := s.db.GetOrder(orderID)
	if err != nil {
		return nil, err
	}

	// Use the mock exchange system to execute the order
	execution, err := exchange.ExecuteOrderAcrossExchanges(order)
	if err != nil {
		return nil, err
	}

	// Set execution ID
	execution.ExecutionID = uuid.New().String()

	// Save the execution to database with idempotency
	if err := s.db.CreateExecutionWithIdempotency(execution, idempotencyKey); err != nil {
		return nil, err
	}

	// Update order status
	order.Status = "FILLED"
	order.UpdatedAt = time.Now()
	// QUESTION: do we need toupdate the order fill price?
	if err := s.db.UpdateOrder(order); err != nil {
		return nil, err
	}

	return execution, nil
}

// GinHandlers contains HTTP handlers for trading endpoints
type GinHandlers struct {
	service *Service
}

func NewGinHandlers(service *Service) *GinHandlers {
	return &GinHandlers{
		service: service,
	}
}

func (h *GinHandlers) CreateOrderHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get idempotency key from header
		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey == "" {
			response.BadRequest(c, "Idempotency-Key header is required")
			return
		}

		var order types.Order
		if err := c.ShouldBindJSON(&order); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		if err := h.service.CreateOrder(&order, idempotencyKey); err != nil {
			response.InternalError(c, err.Error())
			return
		}

		response.Success(c, order)
	}
}

func (h *GinHandlers) GetOrderStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("order_id")

		order, err := h.service.GetOrder(orderID)
		if err != nil {
			response.NotFound(c, "Order not found")
			return
		}

		response.Success(c, order)
	}
}

func (h *GinHandlers) ExecuteOrderHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get idempotency key from header
		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey == "" {
			response.BadRequest(c, "Idempotency-Key header is required")
			return
		}

		orderID := c.Param("order_id")

		execution, err := h.service.ExecuteOrder(orderID, idempotencyKey)
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}

		response.Success(c, execution)
	}
}
