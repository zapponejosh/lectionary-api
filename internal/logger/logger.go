// Package logger provides structured logging using log/slog.
package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/zapponejosh/lectionary-api/internal/config"
)

// Context keys for request-scoped values
type contextKey string

const (
	// RequestIDKey is the context key for request IDs
	RequestIDKey contextKey = "request_id"
)

// Setup initializes the global logger based on configuration.
// Call this once at application startup.
func Setup(cfg *config.Config) *slog.Logger {
	var handler slog.Handler

	// Set log level
	level := parseLevel(cfg.LogLevel)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // Add source file info in debug mode
	}

	// Choose handler based on format
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	// Create logger and set as default
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}

// parseLevel converts a string log level to slog.Level.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID adds a request ID to the logger context.
// Use this in middleware to tag all logs for a request.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// RequestID extracts the request ID from context.
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// FromContext returns a logger with request-scoped attributes.
// If no request ID is in context, returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	logger := slog.Default()

	if requestID := RequestID(ctx); requestID != "" {
		logger = logger.With(slog.String("request_id", requestID))
	}

	return logger
}

// Error logs an error with context.
// Convenience function that extracts request ID and adds error details.
func Error(ctx context.Context, msg string, err error, args ...any) {
	logger := FromContext(ctx)
	allArgs := append([]any{slog.Any("error", err)}, args...)
	logger.ErrorContext(ctx, msg, allArgs...)
}

// Info logs an info message with context.
func Info(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).InfoContext(ctx, msg, args...)
}

// Debug logs a debug message with context.
func Debug(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).DebugContext(ctx, msg, args...)
}

// Warn logs a warning message with context.
func Warn(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).WarnContext(ctx, msg, args...)
}
