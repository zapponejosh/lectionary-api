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

const migrationV3UsersAndAPIKeys = `
-- ============================================================================
-- Users table - stores actual user data
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- User identifiers
    username TEXT NOT NULL UNIQUE,
    email TEXT UNIQUE,  -- Optional for now
    
    -- User profile
    full_name TEXT,
    
    -- User status
    active BOOLEAN NOT NULL DEFAULT 1,
    
    -- Future expansion fields
    -- You can add: subscription_tier, preferences_json, etc.
    
    -- Audit fields
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_login_at TEXT,
    
    CHECK(length(username) >= 3)
);

CREATE INDEX IF NOT EXISTS idx_users_username 
    ON users(username);

CREATE INDEX IF NOT EXISTS idx_users_email 
    ON users(email) 
    WHERE email IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_active 
    ON users(active) 
    WHERE active = 1;

-- ============================================================================
-- API Keys table - multiple keys per user
-- ============================================================================
CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Which user owns this key
    user_id INTEGER NOT NULL,
    
    -- The API key (stored hashed)
    key_hash TEXT NOT NULL UNIQUE,
    
    -- Key metadata
    name TEXT NOT NULL,  -- e.g., "iPhone App", "Web Client", "Development"
    
    -- Key status
    active BOOLEAN NOT NULL DEFAULT 1,
    
    -- Audit fields
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_used_at TEXT,
    revoked_at TEXT,
    
    -- Foreign key
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    CHECK(length(key_hash) = 64)  -- SHA256 = 64 hex chars
);

CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash 
    ON api_keys(key_hash);

CREATE INDEX IF NOT EXISTS idx_api_keys_user_id 
    ON api_keys(user_id);

CREATE INDEX IF NOT EXISTS idx_api_keys_active 
    ON api_keys(active, key_hash) 
    WHERE active = 1;

-- ============================================================================
-- Update reading_progress to reference users table
-- ============================================================================
-- Note: We keep user_id as TEXT for backwards compatibility
-- but now it will store the user.id (as string)
-- Existing data won't break, but new progress will reference users.id

-- Add index for user lookups (if not already present)
CREATE INDEX IF NOT EXISTS idx_reading_progress_user_id 
    ON reading_progress(user_id);

-- ============================================================================
-- Admin user setup
-- ============================================================================
-- Insert your admin user
-- You'll run this manually or via a setup script
INSERT INTO users (username, email, full_name, active) 
VALUES ('admin', 'admin@yourdomain.com', 'Admin User', 1);

-- Your admin API key will be created via the API or direct SQL
-- For initial setup, you can insert it directly:
-- Hash of your chosen admin key (example: 'admin-super-secret-key-12345678901234567890')
-- Run: echo -n 'admin-super-secret-key-12345678901234567890' | sha256sum
-- INSERT INTO api_keys (user_id, key_hash, name, active)
-- VALUES (1, 'YOUR_HASH_HERE', 'Admin Master Key', 1);
`

// migrationsSQL contains all database migrations in order.
// Each migration is identified by its version number (key).
var migrationsSQL = map[int]string{
	1: migrationV1FreshSchema,
	2: migrationV2ProgressTracking,
	3: migrationV3UsersAndAPIKeys,
}
