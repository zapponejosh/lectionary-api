package database

// initialSchemaMigration is the first migration that creates all tables.
// This is embedded in the binary for portability.
const initialSchemaMigration = `
-- Migration: 001_initial_schema
-- Description: Create initial tables for lectionary API

-- ============================================================================
-- Table: liturgical_days
-- ============================================================================
CREATE TABLE IF NOT EXISTS liturgical_days (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- The calendar date (ISO 8601: YYYY-MM-DD)
    date TEXT NOT NULL UNIQUE,
    
    -- Day of week (0=Sunday, 1=Monday, ..., 6=Saturday)
    weekday INTEGER NOT NULL CHECK (weekday >= 0 AND weekday <= 6),
    
    -- Two-year lectionary cycle (1 or 2)
    year_cycle INTEGER NOT NULL CHECK (year_cycle IN (1, 2)),
    
    -- Liturgical season
    season TEXT NOT NULL,
    
    -- Special day name (nullable for ordinary days)
    special_day_name TEXT,
    
    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_liturgical_days_date ON liturgical_days(date);
CREATE INDEX IF NOT EXISTS idx_liturgical_days_season ON liturgical_days(season);
CREATE INDEX IF NOT EXISTS idx_liturgical_days_year_cycle ON liturgical_days(year_cycle);

-- ============================================================================
-- Table: readings
-- ============================================================================
CREATE TABLE IF NOT EXISTS readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Foreign key to liturgical_days
    day_id INTEGER NOT NULL,
    
    -- Type of reading
    reading_type TEXT NOT NULL CHECK (reading_type IN (
        'morning_psalm', 
        'evening_psalm', 
        'old_testament', 
        'epistle', 
        'gospel'
    )),
    
    -- Position within the type (for ordering)
    position INTEGER NOT NULL DEFAULT 1,
    
    -- Scripture reference as string
    reference TEXT NOT NULL,
    
    -- Is this an alternative reading?
    is_alternative BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    FOREIGN KEY (day_id) REFERENCES liturgical_days(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_readings_day_id ON readings(day_id);
CREATE INDEX IF NOT EXISTS idx_readings_type ON readings(reading_type);

-- ============================================================================
-- Table: reading_progress  
-- ============================================================================
CREATE TABLE IF NOT EXISTS reading_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- User identifier
    user_id TEXT NOT NULL,
    
    -- Foreign key to readings
    reading_id INTEGER NOT NULL,
    
    -- Optional notes
    notes TEXT,
    
    -- When the reading was completed
    completed_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    FOREIGN KEY (reading_id) REFERENCES readings(id) ON DELETE CASCADE,
    
    UNIQUE (user_id, reading_id)
);

CREATE INDEX IF NOT EXISTS idx_reading_progress_user ON reading_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_reading_progress_reading ON reading_progress(reading_id);
CREATE INDEX IF NOT EXISTS idx_reading_progress_completed ON reading_progress(completed_at);
`
