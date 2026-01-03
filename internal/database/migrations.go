package database

// migrationV1FreshSchema creates the simplified date-based daily_readings table.
const migrationV1FreshSchema = `
-- ============================================================================
-- Migration 001: Simplified Date-Based Schema (Fresh Start)
-- ============================================================================
CREATE TABLE IF NOT EXISTS daily_readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL UNIQUE,
    morning_psalms TEXT NOT NULL DEFAULT '[]',
    evening_psalms TEXT NOT NULL DEFAULT '[]',
    first_reading TEXT NOT NULL DEFAULT '',
    second_reading TEXT NOT NULL DEFAULT '',
    gospel_reading TEXT NOT NULL DEFAULT '',
    liturgical_info TEXT,
    source_url TEXT NOT NULL DEFAULT '',
    scraped_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_daily_readings_date 
    ON daily_readings(date);

CREATE INDEX IF NOT EXISTS idx_daily_readings_date_range 
    ON daily_readings(date ASC);

CREATE INDEX IF NOT EXISTS idx_daily_readings_scraped_at 
    ON daily_readings(scraped_at);

CREATE TABLE IF NOT EXISTS scrape_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL,
    scraped_at TEXT NOT NULL DEFAULT (datetime('now')),
    source_url TEXT NOT NULL,
    raw_data TEXT,
    success INTEGER NOT NULL DEFAULT 1,
    error_message TEXT,
    duration_ms INTEGER
);

CREATE INDEX IF NOT EXISTS idx_scrape_log_date 
    ON scrape_log(date);

CREATE INDEX IF NOT EXISTS idx_scrape_log_success 
    ON scrape_log(success);

CREATE INDEX IF NOT EXISTS idx_scrape_log_scraped_at 
    ON scrape_log(scraped_at DESC);
`

// migrationV2ProgressTracking adds progress tracking tables.
const migrationV2ProgressTracking = `
-- ============================================================================
-- Migration: Add Progress Tracking (Date-Based)
-- ============================================================================
CREATE TABLE IF NOT EXISTS reading_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    reading_date TEXT NOT NULL,
    notes TEXT,
    completed_at TEXT NOT NULL DEFAULT (datetime('now')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (reading_date) REFERENCES daily_readings(date) ON DELETE CASCADE,
    UNIQUE (user_id, reading_date)
);

CREATE INDEX IF NOT EXISTS idx_reading_progress_user 
    ON reading_progress(user_id);

CREATE INDEX IF NOT EXISTS idx_reading_progress_date 
    ON reading_progress(reading_date);

CREATE INDEX IF NOT EXISTS idx_reading_progress_completed 
    ON reading_progress(completed_at);

CREATE INDEX IF NOT EXISTS idx_reading_progress_user_completed
    ON reading_progress(user_id, completed_at);
`

// migrationsSQL contains all database migrations in order.
// Each migration is identified by its version number (key).
var migrationsSQL = map[int]string{
	1: migrationV1FreshSchema,
	2: migrationV2ProgressTracking,
}
