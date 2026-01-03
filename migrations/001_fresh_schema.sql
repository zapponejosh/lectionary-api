-- Migration 001: Simplified Date-Based Schema (Fresh Start)
-- 
-- This is a fresh start with the simplified approach.
-- Previous position-based schema has been archived.
--
-- Design Philosophy:
-- - Store readings directly by date (YYYY-MM-DD)
-- - Data scraped from PCUSA website
-- - Simple lookups, no complex resolution
-- - Idempotent imports with UPSERT pattern

-- ============================================================================
-- Table: daily_readings
-- ============================================================================
-- One row per calendar date. Contains all readings for that specific date.
-- Data is scraped directly from https://pcusa.org/daily/devotion/YYYY/MM/DD
-- ============================================================================
CREATE TABLE IF NOT EXISTS daily_readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Calendar date in ISO 8601 format (YYYY-MM-DD)
    -- This is the primary lookup key - simple and fast
    date TEXT NOT NULL UNIQUE,
    
    -- Psalms stored as JSON arrays
    -- Example: '["111", "149"]' or '["111", "149:1-8"]'
    -- JSON allows easy marshaling in Go and simple storage
    morning_psalms TEXT NOT NULL DEFAULT '[]',
    evening_psalms TEXT NOT NULL DEFAULT '[]',
    
    -- Scripture readings as plain text references
    -- Example: "1 Kings 19:9-18", "Ephesians 4:17-32"
    first_reading TEXT NOT NULL DEFAULT '',
    second_reading TEXT NOT NULL DEFAULT '',
    gospel_reading TEXT NOT NULL DEFAULT '',
    
    -- Optional: Liturgical metadata for display purposes
    -- Stored as JSON: {"period": "Christmas Season", "special_name": "Epiphany"}
    -- This is purely informational - not used for data lookups
    liturgical_info TEXT,
    
    -- Audit trail: where did this data come from?
    source_url TEXT NOT NULL DEFAULT '',
    scraped_at TEXT,  -- ISO 8601 timestamp of when we scraped this
    
    -- Standard timestamps
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Index for fast date lookups (our primary access pattern)
-- This will be used for: /api/v1/readings/date/{date}
CREATE INDEX IF NOT EXISTS idx_daily_readings_date 
    ON daily_readings(date);

-- Index for date range queries
-- This will be used for: /api/v1/readings/range?start=X&end=Y
CREATE INDEX IF NOT EXISTS idx_daily_readings_date_range 
    ON daily_readings(date ASC);

-- Index for finding gaps in coverage (useful for scraper verification)
CREATE INDEX IF NOT EXISTS idx_daily_readings_scraped_at 
    ON daily_readings(scraped_at);

-- ============================================================================
-- Table: scrape_log
-- ============================================================================
-- Tracks every scrape attempt for auditing and debugging.
-- 
-- Use cases:
-- - Find which dates failed to scrape
-- - Identify when data was last refreshed
-- - Debug scraping issues without re-scraping
-- - Track scraper performance over time
-- ============================================================================
CREATE TABLE IF NOT EXISTS scrape_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Date we attempted to scrape (YYYY-MM-DD)
    date TEXT NOT NULL,
    
    -- When the scrape happened
    scraped_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- URL we scraped from
    source_url TEXT NOT NULL,
    
    -- Raw JSON data we extracted (for debugging)
    -- Helps reproduce issues without re-scraping
    raw_data TEXT,
    
    -- Success flag (1 = success, 0 = failure)
    success INTEGER NOT NULL DEFAULT 1,
    
    -- Error message if failed
    error_message TEXT,
    
    -- How long the scrape took (in milliseconds)
    duration_ms INTEGER
);

-- Index for querying scrape history by date
CREATE INDEX IF NOT EXISTS idx_scrape_log_date 
    ON scrape_log(date);

-- Index for finding failed scrapes
CREATE INDEX IF NOT EXISTS idx_scrape_log_success 
    ON scrape_log(success);

-- Index for time-based queries (most recent scrapes)
CREATE INDEX IF NOT EXISTS idx_scrape_log_scraped_at 
    ON scrape_log(scraped_at DESC);

-- ============================================================================
-- Migration tracking
-- ============================================================================
-- Create schema_migrations table to track applied migrations
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Record that this migration has been applied
INSERT OR IGNORE INTO schema_migrations (version) VALUES (1);