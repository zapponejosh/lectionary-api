package api

import (
	"log/slog"
	"net/http"

	"github.com/zapponejosh/lectionary-api/internal/config"
)

// SetupRoutes configures all HTTP routes and returns the router.
func SetupRoutes(handlers *Handlers, cfg *config.Config, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Apply global middleware
	middleware := ChainMiddleware(
		RecoveryMiddleware(logger),
		RequestIDMiddleware(),
		LoggingMiddleware(logger),
		CORSMiddleware(),
	)

	// Public routes (no authentication)
	mux.HandleFunc("GET /health", handlers.HealthCheck)
	mux.HandleFunc("GET /api/v1/readings/today", handlers.GetTodayReadings)
	mux.HandleFunc("GET /api/v1/readings/date/{date}", handlers.GetDateReadings)
	mux.HandleFunc("GET /api/v1/readings/range", handlers.GetRangeReadings)

	// Authenticated routes (require API key)
	authMiddleware := ChainMiddleware(
		RecoveryMiddleware(logger),
		RequestIDMiddleware(),
		LoggingMiddleware(logger),
		CORSMiddleware(),
		AuthMiddleware(cfg, logger),
	)

	// Apply auth middleware to authenticated routes
	mux.Handle("GET /api/v1/progress", authMiddleware(http.HandlerFunc(handlers.GetProgress)))
	mux.Handle("POST /api/v1/progress", authMiddleware(http.HandlerFunc(handlers.CreateProgress)))
	mux.Handle("DELETE /api/v1/progress/{id}", authMiddleware(http.HandlerFunc(handlers.DeleteProgress)))
	mux.Handle("GET /api/v1/progress/stats", authMiddleware(http.HandlerFunc(handlers.GetProgressStats)))

	// Apply global middleware to all routes
	return middleware(mux)
}
