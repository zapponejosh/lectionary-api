// Package main is the entry point for the Lectionary API server.
package main

import (
	"log/slog"
	"os"

	"github.com/zapponejosh/lectionary-api/internal/config"
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

	// TODO: Phase 1.2 - Initialize database
	// TODO: Phase 3.1 - Start HTTP server

	log.Info("lectionary API ready")
}
