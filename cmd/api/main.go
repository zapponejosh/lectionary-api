// Package main is the entry point for the Lectionary API server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zapponejosh/lectionary-api/internal/api"
	"github.com/zapponejosh/lectionary-api/internal/config"
	"github.com/zapponejosh/lectionary-api/internal/database"
	"github.com/zapponejosh/lectionary-api/internal/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// Setup structured logging
	log := logger.Setup(cfg)

	// Log startup info
	log.Info("starting lectionary API",
		slog.String("env", cfg.Env),
		slog.Int("port", cfg.Port),
		slog.String("log_level", cfg.LogLevel),
	)

	// Initialize database
	log.Info("connecting to database", slog.String("path", cfg.DatabasePath))
	db, err := database.Open(database.DefaultConfig(cfg.DatabasePath), log)
	if err != nil {
		log.Error("failed to open database", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	ctx := context.Background()
	migrated, err := db.Migrate(ctx)
	if err != nil {
		log.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}
	log.Info("migrations complete", slog.Int("applied", migrated))

	// Setup handlers and routes
	handlers := api.NewHandlers(db, cfg, log)
	router := api.SetupRoutes(handlers, cfg, log)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info("server starting", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("server stopped")
}
