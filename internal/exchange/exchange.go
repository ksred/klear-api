package exchange

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/ksred/klear-api/internal/types"
)

// Exchange represents a mock trading exchange
type Exchange struct {
	ID              string
	Name            string
	MinLatency      int // in milliseconds
	MaxLatency      int
	LiquidityFactor float64 // 0-1, represents available liquidity
	SuccessRate     float64 // 0-1, probability of successful execution
	FeeRate         float64 // percentage of transaction value
}

var mockExchanges = []*Exchange{
	{
		ID:              "EXCH1",
		Name:            "Primary Exchange",
		MinLatency:      5,
		MaxLatency:      30,
		LiquidityFactor: 0.9,
		SuccessRate:     0.95,
		FeeRate:         0.001, // 0.1%
	},
	{
		ID:              "EXCH2",
		Name:            "Secondary Exchange",
		MinLatency:      10,
		MaxLatency:      50,
		LiquidityFactor: 0.7,
		SuccessRate:     0.90,
		FeeRate:         0.0008, // 0.08%
	},
	{
		ID:              "EXCH3",
		Name:            "Regional Exchange",
		MinLatency:      15,
		MaxLatency:      70,
		LiquidityFactor: 0.5,
		SuccessRate:     0.85,
		FeeRate:         0.0005, // 0.05%
	},
	{
		ID:              "EXCH4",
		Name:            "Dark Pool",
		MinLatency:      20,
		MaxLatency:      100,
		LiquidityFactor: 0.3,
		SuccessRate:     0.75,
		FeeRate:         0.0003, // 0.03%
	},
}

// ExecuteOrder simulates order execution on a specific exchange
func (e *Exchange) ExecuteOrder(order *types.Order) (*types.ExchangeFill, error) {
	logger := log.With().
		Str("exchange_id", e.ID).
		Str("order_id", order.OrderID).
		Float64("quantity", order.Quantity).
		Float64("price", order.Price).
		Str("side", string(order.Side)).
		Logger()

	logger.Info().Msg("attempting to execute order")

	// Simulate random latency
	latency := rand.Intn(e.MaxLatency-e.MinLatency+1) + e.MinLatency
	logger.Debug().Int("latency_ms", latency).Msg("simulated network latency")
	time.Sleep(time.Duration(latency) * time.Millisecond)

	// Simulate execution success/failure based on success rate
	if rand.Float64() > e.SuccessRate {
		logger.Warn().
			Float64("success_rate", e.SuccessRate).
			Msg("order execution failed due to success rate threshold")
		return nil, fmt.Errorf("execution failed on exchange %s", e.ID)
	}

	// Calculate executed price with random variance (±2%)
	priceVariance := order.Price * (1 + (rand.Float64()*0.04 - 0.02))
	logger.Debug().
		Float64("original_price", order.Price).
		Float64("executed_price", priceVariance).
		Msg("price variance applied")

	// Adjust quantity based on liquidity
	executedQty := order.Quantity
	if rand.Float64() > e.LiquidityFactor {
		executedQty = order.Quantity * e.LiquidityFactor
		logger.Debug().
			Float64("liquidity_factor", e.LiquidityFactor).
			Float64("original_quantity", order.Quantity).
			Float64("executed_quantity", executedQty).
			Msg("quantity adjusted due to liquidity")
		
		if executedQty == 0 {
			logger.Error().Msg("insufficient liquidity for execution")
			return nil, fmt.Errorf("insufficient liquidity on exchange %s", e.ID)
		}
	}

	// Calculate fee amount
	feeAmount := priceVariance * executedQty * e.FeeRate

	fill := &types.ExchangeFill{
		FillID:       fmt.Sprintf("FILL-%s-%d", e.ID, rand.Int63()),
		ExchangeID:   e.ID,
		ExchangeName: e.Name,
		Price:        priceVariance,
		Quantity:     executedQty,
		FeeRate:      e.FeeRate,
		FeeAmount:    feeAmount,
		CreatedAt:    time.Now(),
	}

	logger.Info().
		Str("fill_id", fill.FillID).
		Float64("executed_price", fill.Price).
		Float64("executed_quantity", fill.Quantity).
		Float64("fee_amount", fill.FeeAmount).
		Msg("order executed successfully on exchange")

	return fill, nil
}

// GetBestExchange selects the best exchange based on liquidity and success rate
func GetBestExchange() *Exchange {
	logger := log.With().Str("component", "exchange_selection").Logger()
	
	totalWeight := 0.0
	for _, ex := range mockExchanges {
		totalWeight += ex.LiquidityFactor * ex.SuccessRate
	}

	choice := rand.Float64() * totalWeight
	currentWeight := 0.0

	logger.Debug().
		Float64("total_weight", totalWeight).
		Float64("random_choice", choice).
		Msg("calculating best exchange")

	for _, ex := range mockExchanges {
		currentWeight += ex.LiquidityFactor * ex.SuccessRate
		if currentWeight >= choice {
			logger.Info().
				Str("selected_exchange", ex.ID).
				Float64("liquidity_factor", ex.LiquidityFactor).
				Float64("success_rate", ex.SuccessRate).
				Msg("exchange selected")
			return ex
		}
	}

	logger.Warn().Msg("falling back to primary exchange")
	return mockExchanges[0]
}

// ExecuteOrderAcrossExchanges attempts to execute an order across multiple exchanges
func ExecuteOrderAcrossExchanges(order *types.Order) (*types.Execution, error) {
	logger := log.With().
		Str("order_id", order.OrderID).
		Float64("total_quantity", order.Quantity).
		Str("side", string(order.Side)).
		Logger()

	logger.Info().Msg("starting cross-exchange execution")

	remainingQty := order.Quantity
	var fills []*types.ExchangeFill
	totalExecutedQty := 0.0
	weightedPrice := 0.0

	for i := 0; i < 3 && remainingQty > 0; i++ {
		logger.Debug().
			Int("attempt", i+1).
			Float64("remaining_quantity", remainingQty).
			Msg("attempting execution on next exchange")

		exchange := GetBestExchange()

		attemptOrder := *order
		attemptOrder.Quantity = remainingQty

		fill, err := exchange.ExecuteOrder(&attemptOrder)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("exchange_id", exchange.ID).
				Msg("execution attempt failed")
			continue
		}

		fills = append(fills, fill)
		totalExecutedQty += fill.Quantity
		weightedPrice += fill.Price * fill.Quantity
		remainingQty -= fill.Quantity

		if remainingQty <= 0 {
			logger.Info().Msg("order fully executed")
			break
		}
	}

	if len(fills) == 0 {
		logger.Error().Msg("failed to execute order on any exchange")
		return nil, fmt.Errorf("failed to execute order on any exchange")
	}

	// Calculate average execution price
	averagePrice := weightedPrice / totalExecutedQty

	execution := &types.Execution{
		ExecutionID:   fmt.Sprintf("EXEC-%d", rand.Int63()),
		OrderID:       order.OrderID,
		TotalQuantity: totalExecutedQty,
		AveragePrice:  averagePrice,
		Side:          order.Side,
		Status:        "COMPLETED",
		Fills:         make([]types.ExchangeFill, len(fills)),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Convert fill pointers to values and prepare fill details for logging
	fillDetails := make([]map[string]interface{}, len(fills))
	for i, fill := range fills {
		fill.ExecutionID = execution.ExecutionID
		execution.Fills[i] = *fill
		
		fillDetails[i] = map[string]interface{}{
			"fill_id":        fill.FillID,
			"exchange_id":    fill.ExchangeID,
			"exchange_name":  fill.ExchangeName,
			"quantity":       fill.Quantity,
			"price":         fill.Price,
			"fee_rate":      fill.FeeRate,
			"fee_amount":    fill.FeeAmount,
		}
	}

	logger.Info().
		Str("execution_id", execution.ExecutionID).
		Float64("total_quantity", execution.TotalQuantity).
		Float64("average_price", execution.AveragePrice).
		Float64("remaining_quantity", remainingQty).
		Interface("fills", fillDetails).
		Int("number_of_fills", len(execution.Fills)).
		Float64("total_fees", calculateTotalFees(fills)).
		Msg("cross-exchange execution completed")

	return execution, nil
}

// Helper function to calculate total fees
func calculateTotalFees(fills []*types.ExchangeFill) float64 {
	var totalFees float64
	for _, fill := range fills {
		totalFees += fill.FeeAmount
	}
	return totalFees
}
