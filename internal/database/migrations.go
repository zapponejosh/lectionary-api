package database

// migrationsSQL contains all database migrations.
// Migrations are applied in order by version number.
// Each migration should be idempotent (safe to run multiple times).
//
// Note: We use a simple forward-only migration strategy.
// No down migrations - if we need to change schema, we add a new migration.
var migrationsSQL = map[int]string{
	1: migrationV1FreshSchema,
}

// migrationV1FreshSchema creates the simplified date-based schema.
//
// Design Philosophy:
// - Store readings directly by date (no complex liturgical position lookups)
// - Data scraped from PCUSA website
// - Simple, fast date lookups
// - Idempotent imports with INSERT OR REPLACE
//
// This is a fresh start after archiving the previous position-based approach.
const migrationV1FreshSchema = `
-- Migration 001: Simplified Date-Based Schema (Fresh Start)

-- ============================================================================
-- Table: daily_readings
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

-- ============================================================================
-- Table: scrape_log
-- ============================================================================
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
