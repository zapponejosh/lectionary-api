-- ============================================================================
-- Migration: Add Progress Tracking (Date-Based)
-- ============================================================================
-- This migration adds progress tracking for the simplified date-based schema.
-- 
-- Design decisions:
-- - Tracks completion by DATE, not by reading_id
-- - user_id is a string (hash of API key for MVP)
-- - One progress entry per user per date
-- - Foreign key to daily_readings ensures referential integrity
-- ============================================================================

CREATE TABLE IF NOT EXISTS reading_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- User identifier (string for flexibility)
    -- MVP: hash of API key
    user_id TEXT NOT NULL,
    
    -- Date of the reading (YYYY-MM-DD)
    -- Foreign key to daily_readings
    reading_date TEXT NOT NULL,
    
    -- Optional notes/reflections
    notes TEXT,
    
    -- When the reading was marked complete
    completed_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    
    -- Foreign key constraint
    FOREIGN KEY (reading_date) REFERENCES daily_readings(date) ON DELETE CASCADE,
    
    -- Unique constraint: one progress entry per user per date
    UNIQUE (user_id, reading_date)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_reading_progress_user 
    ON reading_progress(user_id);

CREATE INDEX IF NOT EXISTS idx_reading_progress_date 
    ON reading_progress(reading_date);

CREATE INDEX IF NOT EXISTS idx_reading_progress_completed 
    ON reading_progress(completed_at);

-- Index for streak calculations (user + completed date)
CREATE INDEX IF NOT EXISTS idx_reading_progress_user_completed
    ON reading_progress(user_id, completed_at);