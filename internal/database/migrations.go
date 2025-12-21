package database

// migrationsSQL contains all database migrations.
// Migrations are applied in order by version number.
// Each migration should be idempotent (safe to run multiple times).
var migrationsSQL = map[int]string{
	1: migrationV1DropOldSchema,
	2: migrationV2PositionBasedSchema,
}

// migrationV1DropOldSchema removes the old date-based tables.
// We're starting fresh with the position-based approach.
//
// In production with existing data, you'd want a more careful migration
// that preserves data. For our initial build, a clean slate is fine.
const migrationV1DropOldSchema = `
-- Migration 001: Clean slate
-- Drop old date-based tables if they exist

DROP TABLE IF EXISTS reading_progress;
DROP TABLE IF EXISTS readings;
DROP TABLE IF EXISTS liturgical_days;
`

// migrationV2PositionBasedSchema creates the new position-based schema.
//
// Key design decisions:
//
// 1. POSITIONS NOT DATES
//   - lectionary_days stores the canonical lectionary structure
//   - Each row is a unique position: period + day_identifier
//   - Example: "1st Week of Advent" + "Sunday" = one position
//   - This never changes; we compute dates at runtime
//
// 2. PSALMS ON POSITION
//   - Psalms are the same for both Year 1 and Year 2
//   - Stored as JSON arrays directly on lectionary_days
//   - Example: ["24", "150"] for morning psalms
//
// 3. READINGS WITH YEAR CYCLE
//   - Scripture readings (first, second, gospel) vary by year
//   - year_cycle column on readings table (1 or 2)
//   - Some positions may only have Year 1 readings (null Year 2)
//
// 4. PERIOD TYPES
//   - liturgical_week: Standard weeks ("1st Week of Advent")
//   - dated_week: Date-anchored ("Week following Sun. between Feb. 11 and 17")
//   - fixed_days: Calendar dates ("Christmas Season" with day "December 25")
//
// 5. READING TYPES
//   - first: First scripture reading (typically OT)
//   - second: Second scripture reading (typically Epistle)
//   - gospel: Gospel reading
//     Note: Psalms are NOT in readings table; they're on lectionary_days
const migrationV2PositionBasedSchema = `
-- Migration 002: Position-based schema
-- This is the production schema for the lectionary API

-- ============================================================================
-- Table: lectionary_days
-- ============================================================================
-- Represents a position in the 2-year lectionary cycle.
-- NOT tied to specific calendar dates - dates are computed at runtime.
-- ============================================================================
CREATE TABLE IF NOT EXISTS lectionary_days (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Position identifier: period + day_identifier must be unique
    -- Examples:
    --   period="1st Week of Advent", day_identifier="Sunday"
    --   period="Christmas Season", day_identifier="December 25"
    --   period="Week following Sun. between Feb. 11 and 17", day_identifier="Monday"
    period TEXT NOT NULL,
    day_identifier TEXT NOT NULL,
    
    -- Type of period (affects date resolution logic)
    -- liturgical_week: weeks relative to Easter/Advent
    -- dated_week: weeks anchored to calendar date ranges
    -- fixed_days: specific calendar dates (Christmas, Epiphany, etc.)
    period_type TEXT NOT NULL CHECK (period_type IN (
        'liturgical_week',
        'dated_week', 
        'fixed_days'
    )),
    
    -- Optional special name (Christmas Day, Epiphany, Ash Wednesday, etc.)
    special_name TEXT,
    
    -- Psalms stored as JSON arrays - shared across both year cycles
    -- Example: '["24", "150"]' or '["119:145-176"]'
    morning_psalms TEXT NOT NULL DEFAULT '[]',
    evening_psalms TEXT NOT NULL DEFAULT '[]',
    
    -- Timestamps for auditing
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- Each position must be unique
    UNIQUE (period, day_identifier)
);

-- Index for the most common lookup: find position by period + day
CREATE INDEX IF NOT EXISTS idx_lectionary_days_position 
    ON lectionary_days(period, day_identifier);

-- Index for finding special days (Christmas, Easter, etc.)
CREATE INDEX IF NOT EXISTS idx_lectionary_days_special 
    ON lectionary_days(special_name) 
    WHERE special_name IS NOT NULL;

-- Index by period type for date resolution queries
CREATE INDEX IF NOT EXISTS idx_lectionary_days_period_type 
    ON lectionary_days(period_type);


-- ============================================================================
-- Table: readings
-- ============================================================================
-- Scripture readings for each position, by year cycle.
-- A position may have multiple readings (first, second, gospel) for each year.
-- ============================================================================
CREATE TABLE IF NOT EXISTS readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Foreign key to lectionary_days
    lectionary_day_id INTEGER NOT NULL,
    
    -- Which year of the 2-year cycle (1 or 2)
    year_cycle INTEGER NOT NULL CHECK (year_cycle IN (1, 2)),
    
    -- Type of reading
    -- first: First scripture reading (often OT)
    -- second: Second scripture reading (often Epistle)  
    -- gospel: Gospel reading
    reading_type TEXT NOT NULL CHECK (reading_type IN (
        'first',
        'second',
        'gospel'
    )),
    
    -- Position within type (for ordering when multiple readings of same type)
    -- Usually 1, but some days have multiple first readings
    position INTEGER NOT NULL DEFAULT 1,
    
    -- The actual scripture reference
    -- Examples: "Isaiah 1:1-9", "Matt. 25:1-13", "Gen. 17:1–12a, 15–16"
    reference TEXT NOT NULL,
    
    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- Foreign key constraint
    FOREIGN KEY (lectionary_day_id) REFERENCES lectionary_days(id) ON DELETE CASCADE
);

-- Primary lookup: get all readings for a position + year
CREATE INDEX IF NOT EXISTS idx_readings_day_year 
    ON readings(lectionary_day_id, year_cycle);

-- For queries filtering by reading type
CREATE INDEX IF NOT EXISTS idx_readings_type 
    ON readings(reading_type);


-- ============================================================================
-- Table: reading_progress
-- ============================================================================
-- Tracks user completion of readings. Preserved from original design.
-- Note: This tracks specific readings, which are year-cycle specific.
-- ============================================================================
CREATE TABLE IF NOT EXISTS reading_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- User identifier (flexible string for future auth schemes)
    -- MVP: single user, use "default" or API key hash
    user_id TEXT NOT NULL,
    
    -- Foreign key to readings (specific to a year cycle)
    reading_id INTEGER NOT NULL,
    
    -- Optional personal notes/reflections
    notes TEXT,
    
    -- When the reading was completed
    completed_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- Foreign key constraint
    FOREIGN KEY (reading_id) REFERENCES readings(id) ON DELETE CASCADE,
    
    -- One completion per user per reading
    UNIQUE (user_id, reading_id)
);

-- User's progress queries
CREATE INDEX IF NOT EXISTS idx_reading_progress_user 
    ON reading_progress(user_id);

-- For joining with readings
CREATE INDEX IF NOT EXISTS idx_reading_progress_reading 
    ON reading_progress(reading_id);

-- For streak calculations (readings completed on dates)
CREATE INDEX IF NOT EXISTS idx_reading_progress_completed 
    ON reading_progress(completed_at);
`
