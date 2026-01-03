package database

import (
	"database/sql"
	"encoding/json"
	"time"
)

// =============================================================================
// Core Data Models
// =============================================================================

// DailyReading represents a single day's readings.
// This is a direct mapping of what we scrape from PCUSA.
type DailyReading struct {
	ID             int64      `json:"id"`
	Date           string     `json:"date"`                      // YYYY-MM-DD
	MorningPsalms  []string   `json:"morning_psalms"`            // ["111", "149"]
	EveningPsalms  []string   `json:"evening_psalms"`            // ["107", "15"]
	FirstReading   string     `json:"first_reading"`             // "1 Kings 19:9-18"
	SecondReading  string     `json:"second_reading"`            // "Ephesians 4:17-32"
	GospelReading  string     `json:"gospel_reading"`            // "John 6:15-27"
	LiturgicalInfo *string    `json:"liturgical_info,omitempty"` // Optional JSON metadata
	SourceURL      string     `json:"source_url"`
	ScrapedAt      *time.Time `json:"scraped_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ScrapeLogEntry tracks a scraping attempt for debugging.
type ScrapeLogEntry struct {
	ID           int64     `json:"id"`
	Date         string    `json:"date"`
	ScrapedAt    time.Time `json:"scraped_at"`
	SourceURL    string    `json:"source_url"`
	RawData      *string   `json:"raw_data,omitempty"`
	Success      bool      `json:"success"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	DurationMs   *int64    `json:"duration_ms,omitempty"`
}

// ReadingStats provides overview statistics about the database.
type ReadingStats struct {
	TotalDays     int        `json:"total_days"`
	EarliestDate  string     `json:"earliest_date"`
	LatestDate    string     `json:"latest_date"`
	LastScrapedAt *time.Time `json:"last_scraped_at,omitempty"`
}

// =============================================================================
// Progress Tracking Models (Date-Based)
// =============================================================================

// ReadingProgress tracks a user's completion of a daily reading.
type ReadingProgress struct {
	ID          int64     `json:"id"`
	UserID      string    `json:"user_id"`
	ReadingDate string    `json:"reading_date"` // YYYY-MM-DD
	Notes       *string   `json:"notes,omitempty"`
	CompletedAt time.Time `json:"completed_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProgressStats contains statistics about a user's reading progress.
type ProgressStats struct {
	TotalDays         int     `json:"total_days"`                    // Total days in database
	CompletedDays     int     `json:"completed_days"`                // Days user has completed
	CompletionPercent float64 `json:"completion_percent"`            // Percentage completed
	CurrentStreak     int     `json:"current_streak"`                // Consecutive days (ending today/yesterday)
	LongestStreak     int     `json:"longest_streak"`                // Best streak ever
	LastCompletedDate *string `json:"last_completed_date,omitempty"` // Most recent completion (YYYY-MM-DD)
}

// ReadingWithProgress combines a daily reading with its completion status.
type ReadingWithProgress struct {
	Reading   *DailyReading    `json:"reading"`
	Progress  *ReadingProgress `json:"progress,omitempty"`
	Completed bool             `json:"completed"`
}

// User represents a user of the API.
type User struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Email       *string    `json:"email,omitempty"`
	FullName    *string    `json:"full_name,omitempty"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// APIKey represents an API key for authentication.
type APIKey struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	KeyHash    string     `json:"-"` // Never expose the hash
	Name       string     `json:"name"`
	Active     bool       `json:"active"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

// APIKeyWithPlaintext is returned when creating a new key.
// The plaintext key is only shown once.
type APIKeyWithPlaintext struct {
	APIKey
	PlaintextKey string `json:"plaintext_key"` // Only populated on creation
}

// UserWithKeys combines user info with their API keys.
type UserWithKeys struct {
	User
	APIKeys []APIKey `json:"api_keys"`
}

// =============================================================================
// JSON Helper Functions
// =============================================================================

// MarshalPsalms converts a string slice to JSON for storage.
// Example: ["111", "149"] → '["111","149"]'
func MarshalPsalms(psalms []string) (string, error) {
	if psalms == nil {
		return "[]", nil
	}
	data, err := json.Marshal(psalms)
	if err != nil {
		return "[]", err
	}
	return string(data), nil
}

// UnmarshalPsalms converts JSON string to string slice.
// Example: '["111","149"]' → ["111", "149"]
func UnmarshalPsalms(data string) ([]string, error) {
	if data == "" || data == "[]" {
		return []string{}, nil
	}

	var psalms []string
	err := json.Unmarshal([]byte(data), &psalms)
	if err != nil {
		return []string{}, err
	}
	return psalms, nil
}

// =============================================================================
// Database Helper Functions
// =============================================================================
// These help convert between Go types and SQL nullable types

// NullString returns the string value or empty string if null
func NullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// NullTime returns a pointer to time or nil if null
func NullTime(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

// NullInt64 returns a pointer to int64 or nil if null
func NullInt64(ni sql.NullInt64) *int64 {
	if ni.Valid {
		return &ni.Int64
	}
	return nil
}

// StringToNullString converts string to NullString (for inserts)
func StringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// TimeToNullTime converts time pointer to NullTime (for inserts)
func TimeToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
