-- Migration: 001_initial_schema
-- Description: Create initial tables for lectionary API
-- Created: 2025

-- Enable foreign keys (SQLite requires this per-connection, but we document it here)
-- PRAGMA foreign_keys = ON;

-- ============================================================================
-- Table: liturgical_days
-- ============================================================================
-- One row per calendar date. Contains metadata about the day including
-- liturgical season, year cycle, and any special observances.
--
-- Design decisions:
-- - date is stored as TEXT in ISO 8601 format (YYYY-MM-DD) for SQLite compatibility
-- - year_cycle tracks the two-year lectionary cycle (1 or 2)
-- - season helps with filtering/grouping (Advent, Lent, Easter, etc.)
-- - special_day_name captures observances like "Ash Wednesday", "Easter", etc.
-- ============================================================================
CREATE TABLE IF NOT EXISTS liturgical_days (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- The calendar date (ISO 8601: YYYY-MM-DD)
    date TEXT NOT NULL UNIQUE,
    
    -- Day of week (0=Sunday, 1=Monday, ..., 6=Saturday)
    -- Stored for convenience in queries, derived from date
    weekday INTEGER NOT NULL CHECK (weekday >= 0 AND weekday <= 6),
    
    -- Two-year lectionary cycle (1 or 2)
    year_cycle INTEGER NOT NULL CHECK (year_cycle IN (1, 2)),
    
    -- Liturgical season
    -- Values: advent, christmas, epiphany, lent, holy_week, easter, pentecost, ordinary
    season TEXT NOT NULL,
    
    -- Special day name (nullable for ordinary days)
    -- Examples: "Ash Wednesday", "Palm Sunday", "Easter", "Christmas Day"
    special_day_name TEXT,
    
    -- Timestamps for auditing
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Index for common queries
CREATE INDEX IF NOT EXISTS idx_liturgical_days_date ON liturgical_days(date);
CREATE INDEX IF NOT EXISTS idx_liturgical_days_season ON liturgical_days(season);
CREATE INDEX IF NOT EXISTS idx_liturgical_days_year_cycle ON liturgical_days(year_cycle);

-- ============================================================================
-- Table: readings
-- ============================================================================
-- Multiple readings per day. Each day typically has:
-- - Morning psalms (1-2 psalms)
-- - Evening psalms (1-2 psalms)  
-- - Old Testament reading
-- - Epistle reading
-- - Gospel reading
--
-- Design decisions:
-- - reading_type categorizes the reading for filtering
-- - position orders readings within a type (e.g., first psalm vs second)
-- - reference stores the scripture citation as a string (e.g., "Gen. 17:1–12a")
-- - is_alternative marks optional/alternate readings (indicated by "or" in source)
-- ============================================================================
CREATE TABLE IF NOT EXISTS readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Foreign key to liturgical_days
    day_id INTEGER NOT NULL,
    
    -- Type of reading
    -- Values: morning_psalm, evening_psalm, old_testament, epistle, gospel
    reading_type TEXT NOT NULL CHECK (reading_type IN (
        'morning_psalm', 
        'evening_psalm', 
        'old_testament', 
        'epistle', 
        'gospel'
    )),
    
    -- Position within the type (for ordering multiple psalms)
    -- 1 = primary, 2 = secondary, etc.
    position INTEGER NOT NULL DEFAULT 1,
    
    -- Scripture reference as string (e.g., "Pss. 98; 147:1–11", "Gen. 17:1–12a")
    reference TEXT NOT NULL,
    
    -- Is this an alternative reading? (marked with "or" in source)
    is_alternative BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- Foreign key constraint
    FOREIGN KEY (day_id) REFERENCES liturgical_days(id) ON DELETE CASCADE
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_readings_day_id ON readings(day_id);
CREATE INDEX IF NOT EXISTS idx_readings_type ON readings(reading_type);

-- ============================================================================
-- Table: reading_progress  
-- ============================================================================
-- Tracks user completion of individual readings.
--
-- Design decisions:
-- - user_id is a string to support various auth schemes later
-- - For MVP with single API key, we'll use a constant user_id
-- - notes allow personal reflection/journaling
-- - completed_at tracks when the reading was marked done
-- ============================================================================
CREATE TABLE IF NOT EXISTS reading_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- User identifier (string for flexibility)
    -- MVP: single user, will use "default" or API key hash
    user_id TEXT NOT NULL,
    
    -- Foreign key to readings
    reading_id INTEGER NOT NULL,
    
    -- Optional notes/reflections
    notes TEXT,
    
    -- When the reading was completed
    completed_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- Foreign key constraint
    FOREIGN KEY (reading_id) REFERENCES readings(id) ON DELETE CASCADE,
    
    -- Unique constraint: one progress entry per user per reading
    UNIQUE (user_id, reading_id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_reading_progress_user ON reading_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_reading_progress_reading ON reading_progress(reading_id);
CREATE INDEX IF NOT EXISTS idx_reading_progress_completed ON reading_progress(completed_at);

-- ============================================================================
-- Table: schema_migrations
-- ============================================================================
-- Tracks which migrations have been applied.
-- This is a simple approach - no down migrations, just forward progress.
-- ============================================================================
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);