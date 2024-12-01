package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/ksred/klear-api/internal/auth"
	"github.com/ksred/klear-api/internal/clearing"
	"github.com/ksred/klear-api/internal/database"
	"github.com/ksred/klear-api/internal/settlement"
	"github.com/ksred/klear-api/internal/trading"
	"github.com/ksred/klear-api/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func init() {
	// Configure pretty logging for development
	if os.Getenv("ENV") != "production" {
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		zlog.Logger = zerolog.New(output).With().Timestamp().Logger()
	}

	// Set global log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if os.Getenv("DEBUG") == "true" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func main() {
	// Initialize database
	db, err := database.NewDatabase()
	if err != nil {
		zlog.Fatal().Err(err).Msg("Failed to initialize database")
	}

	// Initialize router
	router := gin.Default()

	// Initialize services and handlers
	authService := auth.NewService("klear-secret-key")
	authHandlers := auth.NewGinHandlers(authService)
	// Register test credentials
	authService.RegisterAPICredentials(auth.TestAPIKey, auth.TestAPISecret)

	tradingService := trading.NewService(db)
	tradingHandlers := trading.NewGinHandlers(tradingService)

	clearingService := clearing.NewService(db)
	clearingHandlers := clearing.NewGinHandlers(clearingService)

	settlementService := settlement.NewService(db)
	settlementHandlers := settlement.NewGinHandlers(settlementService)

	// Create and start settlement processor
	settlementProcessor := settlement.NewProcessor(settlementService.GetDB())
	processorCtx, processorCancel := context.WithCancel(context.Background())
	defer processorCancel()

	go settlementProcessor.Start(processorCtx)

	// Setup middleware
	router.Use(middleware.RateLimit())

	// Setup API routes
	setupRoutes(router, authHandlers, tradingHandlers, clearingHandlers, settlementHandlers)

	// Create server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Graceful shutdown setup
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zlog.Fatal().Err(err).Msg("listen")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zlog.Info().Msg("Shutting down server...")

	// Give outstanding operations 5 seconds to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		zlog.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	zlog.Info().Msg("Server exiting")
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
		orders.Use(middleware.JWTAuth())
		{
			orders.POST("", tradingHandlers.CreateOrderHandler())
			orders.GET("/:order_id", tradingHandlers.GetOrderStatusHandler())
		}

		// Internal routes (should be protected by internal network)
		internal := v1.Group("/internal")
		internal.Use(middleware.InternalAuth())
		{
			internal.POST("/execution/:order_id", tradingHandlers.ExecuteOrderHandler())
			internal.POST("/clearing/:trade_id", clearingHandlers.ClearTradeHandler())
			internal.POST("/settlement/:trade_id", settlementHandlers.SettleTradeHandler())
		}
	}
}