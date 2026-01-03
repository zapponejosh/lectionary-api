package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/zapponejosh/lectionary-api/internal/config"
)

// Middleware is a function that wraps an HTTP handler.
type Middleware func(http.Handler) http.Handler

// ChainMiddleware chains multiple middleware functions together.
// Middleware is applied in the order provided, with the first middleware
// being the outermost (first to receive the request, last to process the response).
//
// Example:
//
//	chain := ChainMiddleware(Recovery, Logging, Auth)
//	// Request flow:  Recovery → Logging → Auth → Handler
//	// Response flow: Handler → Auth → Logging → Recovery
func ChainMiddleware(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// RequestIDMiddleware adds a unique request ID to each request.
// The ID is added to both the request header and response header as X-Request-ID.
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := generateRequestID()
			r.Header.Set("X-Request-ID", requestID)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware logs HTTP requests with structured logging.
// It captures the request method, path, status code, and duration.
func LoggingMiddleware(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap ResponseWriter to capture status code
			wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", duration),
				slog.String("request_id", r.Header.Get("X-Request-ID")),
			)
		})
	}
}

// statusResponseWriter wraps http.ResponseWriter to capture the status code.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *statusResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *statusResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// CORSMiddleware adds CORS headers to responses.
// For production, you should configure allowed origins rather than using "*".
func CORSMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Make this configurable for production
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, X-Timezone")
			w.Header().Set("Access-Control-Max-Age", "3600")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware recovers from panics and returns a 500 error.
// It logs the panic with stack trace information.
func RecoveryMiddleware(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						slog.Any("error", err),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
						slog.String("request_id", r.Header.Get("X-Request-ID")),
					)
					WriteInternalError(w, "Internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware validates API key for authenticated endpoints.
// The API key should be passed in the X-API-Key header.
func AuthMiddleware(cfg *config.Config, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth in development if no API key is configured
			if cfg.IsDevelopment() && cfg.APIKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				WriteUnauthorized(w, "Missing API key")
				return
			}

			if apiKey != cfg.APIKey {
				logger.Warn("invalid API key attempt",
					slog.String("remote_addr", r.RemoteAddr),
					slog.String("path", r.URL.Path),
					slog.String("request_id", r.Header.Get("X-Request-ID")),
				)
				WriteUnauthorized(w, "Invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID extracts the user ID from the request.
// For MVP, we use a hash of the API key as the user ID.
// This allows tracking per-user progress without a full auth system.
func GetUserID(r *http.Request, cfg *config.Config) string {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		return "default"
	}

	// Hash the API key to use as user ID
	// Using first 16 chars provides sufficient uniqueness while keeping IDs manageable
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])[:16]
}

// GetRequestTimezone extracts the timezone from the request.
// It checks the X-Timezone header first, then falls back to UTC.
// Returns the timezone location and whether it was explicitly provided.
func GetRequestTimezone(r *http.Request) (*time.Location, bool) {
	// Check X-Timezone header
	if tz := r.Header.Get("X-Timezone"); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc, true
		}
	}

	// Check query parameter (useful for testing/debugging)
	if tz := r.URL.Query().Get("tz"); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc, true
		}
	}

	// Default to UTC
	return time.UTC, false
}

// GetTodayForRequest returns "today" in the context of the request's timezone.
// The returned time is normalized to midnight in the requested timezone,
// then converted to UTC for consistent storage/lookup.
func GetTodayForRequest(r *http.Request) time.Time {
	loc, _ := GetRequestTimezone(r)
	now := time.Now().In(loc)
	// Return midnight in the user's timezone, converted to UTC
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// generateRequestID generates a unique request ID.
// Format: timestamp-randomhex (e.g., "20060102150405-a1b2c3d4")
func generateRequestID() string {
	timestamp := time.Now().Format("20060102150405")
	randomPart := randomHex(4) // 4 bytes = 8 hex chars
	return fmt.Sprintf("%s-%s", timestamp, randomPart)
}

// randomHex generates a cryptographically random hex string of n bytes.
func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based if crypto/rand fails (extremely rare)
		return fmt.Sprintf("%x", time.Now().UnixNano())[:n*2]
	}
	return hex.EncodeToString(bytes)
}
