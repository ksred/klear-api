package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ksred/klear-api/internal/auth"
	"github.com/ksred/klear-api/internal/clearing"
	"github.com/ksred/klear-api/internal/database"
	"github.com/ksred/klear-api/internal/settlement"
	"github.com/ksred/klear-api/internal/trading"
	"github.com/ksred/klear-api/internal/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	minOrders     = 15
	maxOrders     = 150
	numWorkers    = 5
	serverAddress = "http://localhost:8080"
)

var (
	symbols = []string{"AAPL", "GOOGL", "MSFT", "AMZN", "META"}
	sides   = []string{"BUY", "SELL"}
)

func init() {
	// Configure pretty logging
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
}

type simulationClient struct {
	baseURL   string
	authToken string
	client    *http.Client
}

func newSimulationClient() (*simulationClient, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	sc := &simulationClient{
		baseURL: serverAddress,
		client:  client,
	}

	// Get auth token
	token, err := sc.authenticate()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}
	sc.authToken = token

	return sc, nil
}

func (sc *simulationClient) authenticate() (string, error) {
	credentials := map[string]string{
		"api_key":    auth.TestAPIKey,
		"api_secret": auth.TestAPISecret,
	}

	body, err := json.Marshal(credentials)
	if err != nil {
		return "", err
	}

	resp, err := sc.client.Post(
		fmt.Sprintf("%s/api/v1/auth/token", sc.baseURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"jwt_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Token, nil
}

func (sc *simulationClient) createOrder(order *types.Order) (string, error) {
	body, err := json.Marshal(order)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v1/orders", sc.baseURL),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.authToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.New().String())

	resp, err := sc.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create order failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read and log the full response for debugging
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	log.Debug().Str("response", string(respBody)).Msg("Create order response")

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			OrderID string `json:"order_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w, body: %s", err, string(respBody))
	}

	if result.Data.OrderID == "" {
		return "", fmt.Errorf("no order ID in response: %s", string(respBody))
	}

	return result.Data.OrderID, nil
}

func (sc *simulationClient) executeOrder(orderID string) (*types.Execution, error) {
	// Add validation for empty orderID
	if orderID == "" {
		return nil, fmt.Errorf("orderID cannot be empty")
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v1/internal/execution/%s", sc.baseURL, orderID),
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.authToken))
	req.Header.Set("Idempotency-Key", uuid.New().String())

	resp, err := sc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read and log the full response for debugging
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	log.Debug().Str("response", string(respBody)).Msg("Execute order response")

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("execute order failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Success bool            `json:"success"`
		Data    types.Execution `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(respBody))
	}

	if result.Data.ExecutionID == "" {
		return nil, fmt.Errorf("no execution ID in response: %s", string(respBody))
	}

	return &result.Data, nil
}

func (sc *simulationClient) getOrder(orderID string) (*types.Order, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v1/orders/%s", sc.baseURL, orderID),
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.authToken))

	resp, err := sc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	log.Debug().Str("response", string(respBody)).Msg("Get order response")

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("get order failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Success bool        `json:"success"`
		Data    types.Order `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(respBody))
	}

	return &result.Data, nil
}

func (sc *simulationClient) clearTrade(execID string) (*types.ClearingResponse, error) {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v1/internal/clearing/%s", sc.baseURL, execID),
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.authToken))
	req.Header.Set("Idempotency-Key", uuid.New().String())

	resp, err := sc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	log.Debug().Str("response", string(respBody)).Msg("Clear trade response")

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("clear trade failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Success bool                   `json:"success"`
		Data    types.ClearingResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(respBody))
	}

	if result.Data.ClearingID == "" {
		return nil, fmt.Errorf("no clearing ID in response: %s", string(respBody))
	}

	return &result.Data, nil
}

func (sc *simulationClient) settleTrade(execID string) (*types.SettlementResponse, error) {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v1/internal/settlement/%s", sc.baseURL, execID),
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.authToken))
	req.Header.Set("Idempotency-Key", uuid.New().String())

	resp, err := sc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	log.Debug().Str("response", string(respBody)).Msg("Settle trade response")

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("settle trade failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Success bool                     `json:"success"`
		Data    types.SettlementResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(respBody))
	}

	if result.Data.SettlementID == "" {
		return nil, fmt.Errorf("no settlement ID in response: %s", string(respBody))
	}

	return &result.Data, nil
}

func main() {
	// Start the server in a goroutine
	go func() {
		if err := startServer(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Initialize simulation client
	simClient, err := newSimulationClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize simulation client")
	}

	// Generate random number of orders to process
	targetOrders := rand.Intn(maxOrders-minOrders) + minOrders
	log.Info().Int("target_orders", targetOrders).Msg("Starting simulation")

	// Channel to collect order IDs
	ordersChan := make(chan string, targetOrders)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			createOrdersHTTP(workerID, targetOrders/numWorkers, simClient, ordersChan)
		}(i)
	}

	// Wait for all orders to be created
	wg.Wait()
	close(ordersChan)

	// Collect all order IDs
	var orderIDs []string
	for orderID := range ordersChan {
		orderIDs = append(orderIDs, orderID)
	}

	log.Info().Int("orders_created", len(orderIDs)).Msg("All orders created")

	// Collect statistics during processing
	stats := struct {
		TotalOrders      int
		ExecutedOrders   int
		ClearedTrades    int
		SettledTrades    int
		TotalValue       float64
		FailedOrders     int
		FailedClearing   int
		FailedSettlement int
		StartTime        time.Time
		Symbols          map[string]int
		Sides            map[string]int
	}{
		StartTime: time.Now(),
		Symbols:   make(map[string]int),
		Sides:     make(map[string]int),
	}

	// Update the processing loops to collect statistics
	stats.TotalOrders = len(orderIDs)

	// Execute trades with stats
	var executionIDs []string
	for _, orderID := range orderIDs {
		if orderID == "" {
			log.Error().Msg("Empty order ID encountered, skipping")
			stats.FailedOrders++
			continue
		}

		execution, err := simClient.executeOrder(orderID)
		if err != nil {
			log.Error().Err(err).
				Str("order_id", orderID).
				Msg("Failed to execute order")
			stats.FailedOrders++
			continue
		}
		executionIDs = append(executionIDs, execution.ExecutionID)
		stats.ExecutedOrders++
		stats.TotalValue += execution.AveragePrice * execution.TotalQuantity

		// Get order details for statistics
		order, err := simClient.getOrder(orderID)
		if err == nil && order != nil {
			stats.Symbols[order.Symbol]++
			stats.Sides[order.Side]++
		}

		log.Info().
			Str("order_id", orderID).
			Str("execution_id", execution.ExecutionID).
			Float64("price", execution.AveragePrice).
			Float64("quantity", execution.TotalQuantity).
			Msg("Order executed")
	}

	// Clear and settle trades with stats
	for _, execID := range executionIDs {
		clearing, err := simClient.clearTrade(execID)
		if err != nil {
			log.Error().Err(err).Str("execution_id", execID).Msg("Failed to clear trade")
			stats.FailedClearing++
			continue
		}
		stats.ClearedTrades++
		log.Info().
			Str("execution_id", execID).
			Str("clearing_id", clearing.ClearingID).
			Float64("settlement_amount", clearing.SettlementAmount).
			Msg("Trade cleared")

		settlement, err := simClient.settleTrade(execID)
		if err != nil {
			log.Error().Err(err).Str("execution_id", execID).Msg("Failed to settle trade")
			stats.FailedSettlement++
			continue
		}
		stats.SettledTrades++
		log.Info().
			Str("execution_id", execID).
			Str("settlement_id", settlement.SettlementID).
			Float64("final_amount", settlement.FinalAmount).
			Time("settlement_date", settlement.SettlementDate).
			Msg("Trade settled")
	}

	// Print summary
	duration := time.Since(stats.StartTime)
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🚀 TRADING SIMULATION SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf(`
📊 Order Statistics
------------------
Total Orders:     %d
Executed:         %d
Cleared:          %d
Settled:          %d
Failed Orders:    %d
Failed Clearing:  %d
Failed Settlement:%d
Total Value:      $%.2f
Duration:         %v

📈 Symbol Distribution
--------------------
`, stats.TotalOrders, stats.ExecutedOrders, stats.ClearedTrades, stats.SettledTrades,
		stats.FailedOrders, stats.FailedClearing, stats.FailedSettlement,
		stats.TotalValue, duration.Round(time.Millisecond))

	// Print symbol distribution with simple ASCII bar chart
	maxSymbolCount := 0
	for _, count := range stats.Symbols {
		if count > maxSymbolCount {
			maxSymbolCount = count
		}
	}

	for symbol, count := range stats.Symbols {
		barLength := int(float64(count) / float64(maxSymbolCount) * 20)
		bar := strings.Repeat("█", barLength)
		fmt.Printf("%-6s: %s (%d)\n", symbol, bar, count)
	}

	fmt.Println("\n📉 Side Distribution")
	fmt.Println("------------------")
	for side, count := range stats.Sides {
		barLength := int(float64(count) / float64(stats.TotalOrders) * 20)
		bar := strings.Repeat("█", barLength)
		fmt.Printf("%-4s: %s (%d)\n", side, bar, count)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))

	// Success rate calculation
	successRate := float64(stats.SettledTrades) / float64(stats.TotalOrders) * 100
	log.Info().
		Float64("success_rate", successRate).
		Int("total_orders", stats.TotalOrders).
		Int("settled_trades", stats.SettledTrades).
		Float64("total_value", stats.TotalValue).
		Dur("duration", duration).
		Msg("Simulation completed")
}

func createOrdersHTTP(workerID, numOrders int, simClient *simulationClient, ordersChan chan<- string) {
	for i := 0; i < numOrders; i++ {
		order := &types.Order{
			ClientID:  fmt.Sprintf("CLIENT_%d", workerID),
			Symbol:    symbols[rand.Intn(len(symbols))],
			Side:      sides[rand.Intn(len(sides))],
			OrderType: "MARKET",
			Quantity:  float64(rand.Intn(100) + 1),
			Price:     float64(rand.Intn(1000) + 100),
			Status:    "PENDING",
		}

		orderID, err := simClient.createOrder(order)
		if err != nil {
			log.Error().Err(err).
				Str("worker_id", fmt.Sprintf("%d", workerID)).
				Str("symbol", order.Symbol).
				Msg("Failed to create order")
			continue
		}

		ordersChan <- orderID
		log.Info().
			Str("worker_id", fmt.Sprintf("%d", workerID)).
			Str("order_id", orderID).
			Str("symbol", order.Symbol).
			Str("side", order.Side).
			Float64("quantity", order.Quantity).
			Float64("price", order.Price).
			Msg("Order created")

		// Random sleep between orders
		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	}
}

func startServer() error {
	// Initialize database
	db, err := database.NewDatabase()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize services
	authService := auth.NewService("klear-secret-key")
	tradingService := trading.NewService(db)
	clearingService := clearing.NewService(db)
	settlementService := settlement.NewService(db)

	// Register test credentials
	authService.RegisterAPICredentials(auth.TestAPIKey, auth.TestAPISecret)

	// Initialize router
	router := gin.Default()
	authHandlers := auth.NewGinHandlers(authService)
	tradingHandlers := trading.NewGinHandlers(tradingService)
	clearingHandlers := clearing.NewGinHandlers(clearingService)
	settlementHandlers := settlement.NewGinHandlers(settlementService)

	// Setup routes
	setupRoutes(router, authHandlers, tradingHandlers, clearingHandlers, settlementHandlers)

	// Start the server
	return router.Run(":8080")
}

func setupRoutes(
	router *gin.Engine,
	authHandlers *auth.GinHandlers,
	tradingHandlers *trading.GinHandlers,
	clearingHandlers *clearing.GinHandlers,
	settlementHandlers *settlement.GinHandlers,
) {
	v1 := router.Group("/api/v1")
	{
		// Auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/token", authHandlers.GenerateTokenHandler())
		}

		// Order routes
		orders := v1.Group("/orders")
		{
			orders.POST("", tradingHandlers.CreateOrderHandler())
			orders.GET("/:order_id", tradingHandlers.GetOrderStatusHandler())
		}

		// Internal routes
		internal := v1.Group("/internal")
		{
			internal.POST("/execution/:order_id", tradingHandlers.ExecuteOrderHandler())
			internal.POST("/clearing/:trade_id", clearingHandlers.ClearTradeHandler())
			internal.POST("/settlement/:trade_id", settlementHandlers.SettleTradeHandler())
		}
	}
}