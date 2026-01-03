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
//	TODO: Document route structure here

func SetupRoutes(handlers *Handlers, cfg *config.Config, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	baseMiddleware := ChainMiddleware(
		RecoveryMiddleware(logger),
		RequestIDMiddleware(),
		LoggingMiddleware(logger),
		CORSMiddleware(),
	)

	// Auth middleware for regular users
	authWrap := AuthMiddleware(handlers.db, logger)

	// Admin-only middleware
	adminWrap := func(h http.Handler) http.Handler {
		return AdminOnlyMiddleware(cfg, logger)(h)
	}

	// ==========================================================================
	// Public routes
	// ==========================================================================
	mux.HandleFunc("GET /health", handlers.HealthCheck)
	mux.HandleFunc("GET /api/v1/readings/today", handlers.GetTodayReadings)
	mux.HandleFunc("GET /api/v1/readings/date/{date}", handlers.GetDateReadings)
	mux.HandleFunc("GET /api/v1/readings/range", handlers.GetRangeReadings)

	// ==========================================================================
	// User routes (authenticated)
	// ==========================================================================
	mux.Handle("GET /api/v1/me", authWrap(http.HandlerFunc(handlers.GetCurrentUser)))
	mux.Handle("GET /api/v1/me/keys", authWrap(http.HandlerFunc(handlers.GetMyAPIKeys)))
	mux.Handle("DELETE /api/v1/me/keys/{keyID}", authWrap(http.HandlerFunc(handlers.RevokeMyAPIKey)))

	mux.Handle("GET /api/v1/progress", authWrap(http.HandlerFunc(handlers.GetProgress)))
	mux.Handle("POST /api/v1/progress", authWrap(http.HandlerFunc(handlers.CreateProgress)))
	mux.Handle("DELETE /api/v1/progress/{id}", authWrap(http.HandlerFunc(handlers.DeleteProgress)))
	mux.Handle("GET /api/v1/progress/stats", authWrap(http.HandlerFunc(handlers.GetProgressStats)))

	// ==========================================================================
	// Admin routes (admin key only)
	// ==========================================================================
	mux.Handle("GET /api/v1/admin/users", adminWrap(http.HandlerFunc(handlers.ListUsers)))
	mux.Handle("POST /api/v1/admin/users", adminWrap(http.HandlerFunc(handlers.CreateUser)))
	mux.Handle("POST /api/v1/admin/users/{userID}/keys", adminWrap(http.HandlerFunc(handlers.CreateAPIKey)))

	return baseMiddleware(mux)
}
