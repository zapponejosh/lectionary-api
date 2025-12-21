// Package database provides database access for the lectionary API.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// =============================================================================
// Database Connection
// =============================================================================

// DB wraps the standard sql.DB with lectionary-specific methods.
type DB struct {
	*sql.DB
	logger *slog.Logger
}

// Config holds database configuration options.
type Config struct {
	Path            string        // Path to SQLite database file
	MaxOpenConns    int           // Maximum open connections (default: 1 for SQLite)
	MaxIdleConns    int           // Maximum idle connections (default: 1)
	ConnMaxLifetime time.Duration // Connection max lifetime (default: 1 hour)
}

// DefaultConfig returns sensible defaults for SQLite.
//
// Why these values?
//   - MaxOpenConns=1: SQLite only allows one writer at a time. Multiple connections
//     can cause "database is locked" errors under write load.
//   - WAL mode (set in DSN): Allows concurrent readers while writing.
//   - Busy timeout (set in DSN): Waits up to 5s if database is locked.
func DefaultConfig(path string) Config {
	return Config{
		Path:            path,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
	}
}

// Open creates a new database connection with SQLite-optimized settings.
//
// The caller is responsible for calling Close() when done.
func Open(cfg Config, logger *slog.Logger) (*DB, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Ensure the directory exists
	dir := filepath.Dir(cfg.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	// Build connection string with SQLite pragmas
	// _journal_mode=WAL: Better concurrent read performance
	// _foreign_keys=ON: Enforce referential integrity
	// _busy_timeout=5000: Wait up to 5s if database is locked
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000",
		cfg.Path)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	logger.Info("database connected",
		slog.String("path", cfg.Path),
		slog.Int("max_open_conns", cfg.MaxOpenConns),
	)

	return &DB{
		DB:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	db.logger.Info("closing database connection")
	return db.DB.Close()
}

// Health checks if the database connection is healthy.
func (db *DB) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	return nil
}

// =============================================================================
// Migrations
// =============================================================================

// Migrate runs all pending database migrations.
//
// This uses a simple forward-only migration strategy:
// 1. Check which migrations have been applied (via schema_migrations table)
// 2. Apply any new ones in order
//
// Returns the number of migrations applied.
func (db *DB) Migrate(ctx context.Context) (int, error) {
	db.logger.Info("running database migrations")

	// Start a transaction for safety
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // No-op if committed

	// Ensure schema_migrations table exists
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return 0, fmt.Errorf("create schema_migrations table: %w", err)
	}
	db.logger.Info("schema_migrations ensured (inside tx)")

	// Get already applied versions
	applied := make(map[int]bool)
	rows, err := tx.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return 0, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return 0, fmt.Errorf("scan migration version: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate migration versions: %w", err)
	}

	// Apply migrations in order
	count := 0
	for version := 1; version <= len(migrationsSQL); version++ {
		if applied[version] {
			db.logger.Debug("migration already applied",
				slog.Int("version", version),
			)
			continue
		}

		db.logger.Info("applying migration",
			slog.Int("version", version),
		)

		content, ok := migrationsSQL[version]
		if !ok {
			return count, fmt.Errorf("migration %d not found", version)
		}

		if _, err := tx.ExecContext(ctx, content); err != nil {
			return count, fmt.Errorf("execute migration %d: %w", version, err)
		}

		_, err = tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (version) VALUES (?)",
			version,
		)
		if err != nil {
			return count, fmt.Errorf("record migration %d: %w", version, err)
		}

		count++
	}

	if err := tx.Commit(); err != nil {
		return count, fmt.Errorf("commit migrations: %w", err)
	}

	db.logger.Info("migrations complete",
		slog.Int("applied", count),
		slog.Int("total", len(migrationsSQL)),
	)

	return count, nil
}

// =============================================================================
// Transaction Helpers
// =============================================================================

// Tx represents a database transaction with helper methods.
type Tx struct {
	*sql.Tx
}

// BeginTx starts a new transaction.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

// WithTx executes a function within a transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, it's committed.
//
// Example:
//
//	err := db.WithTx(ctx, func(tx *database.Tx) error {
//	    // do work with tx
//	    return nil
//	})
func (db *DB) WithTx(ctx context.Context, fn func(*Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// =============================================================================
// Error Types
// =============================================================================

// ErrNotFound is returned when a requested record doesn't exist.
var ErrNotFound = errors.New("record not found")

// ErrDuplicate is returned when a unique constraint is violated.
var ErrDuplicate = errors.New("duplicate record")

// IsNotFound checks if an error is a "not found" error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, sql.ErrNoRows)
}
