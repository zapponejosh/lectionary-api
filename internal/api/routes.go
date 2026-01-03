package api

import (
	"log/slog"
	"net/http"

	"github.com/zapponejosh/lectionary-api/internal/config"
)

// SetupRoutes configures all HTTP routes and returns the router.
//
// Route structure:
//
//	Public (no authentication):
//	  GET  /health                        - Health check
//	  GET  /api/v1/readings/today         - Today's readings
//	  GET  /api/v1/readings/date/{date}   - Readings for specific date
//	  GET  /api/v1/readings/range         - Readings for date range
//
//	Authenticated (requires X-API-Key header):
//	  GET    /api/v1/progress             - Get reading progress
//	  POST   /api/v1/progress             - Mark reading complete
//	  DELETE /api/v1/progress/{id}        - Delete progress entry
//	  GET    /api/v1/progress/stats       - Get progress statistics
func SetupRoutes(handlers *Handlers, cfg *config.Config, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Base middleware applied to ALL routes (in order):
	// 1. Recovery - catch panics from everything below
	// 2. RequestID - tag request for correlation
	// 3. Logging - log request/response
	// 4. CORS - handle cross-origin requests
	baseMiddleware := ChainMiddleware(
		RecoveryMiddleware(logger),
		RequestIDMiddleware(),
		LoggingMiddleware(logger),
		CORSMiddleware(),
	)

	// Auth middleware wraps individual handlers (not the whole mux)
	// This prevents double-application of base middleware
	authWrap := AuthMiddleware(cfg, logger)

	// ==========================================================================
	// Public routes (no authentication required)
	// ==========================================================================

	mux.HandleFunc("GET /health", handlers.HealthCheck)
	mux.HandleFunc("GET /api/v1/readings/today", handlers.GetTodayReadings)
	mux.HandleFunc("GET /api/v1/readings/date/{date}", handlers.GetDateReadings)
	mux.HandleFunc("GET /api/v1/readings/range", handlers.GetRangeReadings)

	// ==========================================================================
	// Authenticated routes (require X-API-Key header)
	// ==========================================================================

	// Wrap each authenticated handler with auth middleware only
	mux.Handle("GET /api/v1/progress", authWrap(http.HandlerFunc(handlers.GetProgress)))
	mux.Handle("POST /api/v1/progress", authWrap(http.HandlerFunc(handlers.CreateProgress)))
	mux.Handle("DELETE /api/v1/progress/{id}", authWrap(http.HandlerFunc(handlers.DeleteProgress)))
	mux.Handle("GET /api/v1/progress/stats", authWrap(http.HandlerFunc(handlers.GetProgressStats)))

	// Apply base middleware to entire mux (once, at the outer layer)
	return baseMiddleware(mux)
}
