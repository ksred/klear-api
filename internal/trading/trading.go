package trading

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ksred/klear-api/internal/auth"
	"github.com/ksred/klear-api/internal/exchange"
	"github.com/ksred/klear-api/internal/types"
	"github.com/ksred/klear-api/pkg/response"
	"gorm.io/gorm"
)

// Service handles trading operations and order management
type Service struct {
	db *Database
}

// NewService creates a new trading service with the given database connection
func NewService(gormDB *gorm.DB) *Service {
	return &Service{
		db: NewDatabase(gormDB),
	}
}

// CreateOrder creates a new order with idempotency support
// It checks for existing orders with the same idempotency key and returns the existing order if found
// Parameters:
//   - order: The order to create
//   - idempotencyKey: Unique key to prevent duplicate order creation
func (s *Service) CreateOrder(order *types.Order, idempotencyKey string) error {
	// Check for existing idempotency record
	record, err := s.db.GetIdempotencyRecord(idempotencyKey)

	// If record exists and hasn't expired
	if err == nil && record.ExpiresAt.After(time.Now()) && record != nil {
		// Return existing order
		existingOrder, err := s.db.GetOrder(record.ResourceID)
		if err != nil {
			return err
		}
		if existingOrder == nil {
			return errors.New("order not found")
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

// GetOrder retrieves an order by its ID
func (s *Service) GetOrder(orderID string) (*types.Order, error) {
	return s.db.GetOrder(orderID)
}

// GetOrderByOrderIDAndClientID retrieves an order by its ID and client ID
func (s *Service) GetOrderByOrderIDAndClientID(orderID, clientID string) (*types.Order, error) {
	return s.db.GetOrderByOrderIDAndClientID(orderID, clientID)
}

// ExecuteOrder executes an existing order with idempotency support
// It routes the order to available exchanges and records the execution results
// Parameters:
//   - orderID: ID of the order to execute
//   - idempotencyKey: Unique key to prevent duplicate execution
func (s *Service) ExecuteOrder(orderID string, idempotencyKey string) (*types.Execution, error) {
	// Check for existing idempotency record
	record, err := s.db.GetIdempotencyRecord(idempotencyKey)

	// If record exists and hasn't expired
	if err == nil && record.ExpiresAt.After(time.Now()) && record != nil {
		// Return existing execution
		existingExecution, err := s.db.GetExecution(record.ResourceID)
		if err != nil {
			return nil, err
		}
		return existingExecution, nil
	}

	order, err := s.db.GetOrder(orderID)
	if err != nil || order == nil {
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

// NewGinHandlers creates a new set of HTTP handlers for trading endpoints
func NewGinHandlers(service *Service) *GinHandlers {
	return &GinHandlers{
		service: service,
	}
}

// CreateOrderHandler handles POST requests to create new orders
// Requires a valid JWT token and idempotency key in headers
// Request body should contain the order details
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

// GetOrderStatusHandler handles GET requests to retrieve order status
// Requires a valid JWT token
// URL parameter: order_id
func (h *GinHandlers) GetOrderStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get claims from context
		claims, exists := c.Get("claims")
		if !exists {
			response.Unauthorized(c, "Missing authentication claims")
			return
		}

		// Get client ID from claims
		clientID := auth.GetClientID(claims)
		if clientID == "" {
			response.Unauthorized(c, "Invalid client ID in token")
			return
		}

		orderID := c.Param("order_id")
		if orderID == "" {
			response.BadRequest(c, "Order ID is required")
			return
		}

		order, err := h.service.GetOrderByOrderIDAndClientID(orderID, clientID)
		if err != nil || order == nil {
			response.NotFound(c, "Order not found")
			return
		}

		response.Success(c, order)
	}
}

// ExecuteOrderHandler handles POST requests to execute orders
// Requires internal authentication and idempotency key
// URL parameter: order_id
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
