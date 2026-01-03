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
