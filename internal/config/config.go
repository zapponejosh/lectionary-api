// Package config handles application configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
// Fields are populated from environment variables.
type Config struct {
	// Server settings
	Port int    // HTTP port to listen on
	Env  string // development, staging, production

	// Database
	DatabasePath string // Path to SQLite file

	// Authentication
	APIKey string // API key for authenticated endpoints

	// Logging
	LogLevel  string // debug, info, warn, error
	LogFormat string // json, text
}

// Environment constants
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

// Load reads configuration from environment variables.
// In development, it first loads from .env file if present.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	// This is a no-op in production where env vars are set directly
	_ = godotenv.Load()

	cfg := &Config{}

	// Server settings
	cfg.Port = getEnvInt("PORT", 8080)
	cfg.Env = getEnv("ENV", EnvDevelopment)

	// Database
	cfg.DatabasePath = getEnv("DATABASE_PATH", "./data/lectionary.db")

	// Authentication
	cfg.APIKey = getEnv("API_KEY", "")

	// Logging
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.LogFormat = getEnv("LOG_FORMAT", "text")

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks that all required configuration is present and valid.
func (c *Config) Validate() error {
	var errs []error

	// Validate port range
	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("PORT must be between 1 and 65535, got %d", c.Port))
	}

	// Validate environment
	switch c.Env {
	case EnvDevelopment, EnvStaging, EnvProduction:
		// Valid
	default:
		errs = append(errs, fmt.Errorf("ENV must be one of: development, staging, production; got %q", c.Env))
	}

	// Validate database path is set
	if c.DatabasePath == "" {
		errs = append(errs, errors.New("DATABASE_PATH is required"))
	}

	// API key is required in production
	if c.Env == EnvProduction && c.APIKey == "" {
		errs = append(errs, errors.New("API_KEY is required in production"))
	}

	// Warn in development if API key is default/empty
	// (We don't error, just let it be for easier local dev)

	// Validate log level
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
		// Valid
	default:
		errs = append(errs, fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error; got %q", c.LogLevel))
	}

	// Validate log format
	switch c.LogFormat {
	case "json", "text":
		// Valid
	default:
		errs = append(errs, fmt.Errorf("LOG_FORMAT must be one of: json, text; got %q", c.LogFormat))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Env == EnvDevelopment
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Env == EnvProduction
}

// getEnv reads an environment variable with a default fallback.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt reads an environment variable as an integer with a default fallback.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
