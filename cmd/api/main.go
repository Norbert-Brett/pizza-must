package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"pizza-must/internal/config"
	"pizza-must/internal/database"
	"pizza-must/internal/logger"
	"pizza-must/internal/server"

	"go.uber.org/zap"
)

func gracefulShutdown(apiServer *server.Server, logger *zap.Logger, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	logger.Info("Shutting down gracefully, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// The context is used to inform the server it has 30 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	// Close server resources
	if err := apiServer.Close(); err != nil {
		logger.Error("Error closing server resources", zap.Error(err))
	}

	logger.Info("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log, err := logger.New(cfg.Server.Env)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer log.Sync()

	log.Info("Starting ordering platform API",
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
	)

	// Initialize database
	dbService := database.New()
	db := dbService.DB()

	// Check database health
	health := dbService.Health()
	log.Info("Database health check", zap.Any("health", health))

	// Run migrations
	if err := database.RunMigrations(db, "migrations", log); err != nil {
		log.Fatal("Failed to run migrations", zap.Error(err))
	}
	log.Info("Database migrations completed successfully")

	// Create server
	srv := server.NewServer(cfg, log, db)

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(srv, log, done)

	log.Info("Server listening", zap.String("addr", srv.Addr))

	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal("HTTP server error", zap.Error(err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	log.Info("Graceful shutdown complete")
}
