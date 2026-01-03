package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// Error Types
// =============================================================================

// var (
// 	// ErrNotFound is returned when a requested record doesn't exist
// 	ErrNotFound = errors.New("not found")
// )

// // IsNotFound checks if an error is a not-found error
// func IsNotFound(err error) bool {
// 	return errors.Is(err, ErrNotFound)
// }

// =============================================================================
// Helper Functions
// =============================================================================

// parseTimestamp parses a timestamp from SQLite TEXT format.
// Tries multiple formats and returns nil if parsing fails.
func parseTimestamp(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}

	// Try RFC3339 format first (with timezone)
	t, err := time.Parse(time.RFC3339, ns.String)
	if err == nil {
		return &t
	}

	// Try SQLite datetime format (no timezone)
	t, err = time.Parse("2006-01-02 15:04:05", ns.String)
	if err == nil {
		return &t
	}

	// Try ISO format with microseconds (no timezone)
	t, err = time.Parse("2006-01-02T15:04:05.999999", ns.String)
	if err == nil {
		return &t
	}

	// If all fail, return nil
	return nil
}

// =============================================================================
// Daily Reading Queries
// =============================================================================

// GetReadingByDate retrieves readings for a specific date.
// Returns ErrNotFound if the date doesn't exist in the database.
//
// This is the most common query - used for /api/v1/readings/date/{date}
func (db *DB) GetReadingByDate(ctx context.Context, date string) (*DailyReading, error) {
	query := `
		SELECT 
			id, date, 
			morning_psalms, evening_psalms,
			first_reading, second_reading, gospel_reading,
			liturgical_info, source_url, scraped_at,
			created_at, updated_at
		FROM daily_readings
		WHERE date = ?
	`

	var reading DailyReading
	var morningPsalmsJSON, eveningPsalmsJSON string
	var liturgicalInfo, sourceURL, scrapedAtStr, createdAtStr, updatedAtStr sql.NullString

	err := db.QueryRowContext(ctx, query, date).Scan(
		&reading.ID,
		&reading.Date,
		&morningPsalmsJSON,
		&eveningPsalmsJSON,
		&reading.FirstReading,
		&reading.SecondReading,
		&reading.GospelReading,
		&liturgicalInfo,
		&sourceURL,
		&scrapedAtStr,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query reading by date: %w", err)
	}

	// Unmarshal JSON psalm arrays
	reading.MorningPsalms, err = UnmarshalPsalms(morningPsalmsJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal morning psalms: %w", err)
	}

	reading.EveningPsalms, err = UnmarshalPsalms(eveningPsalmsJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal evening psalms: %w", err)
	}

	// Handle nullable fields
	if liturgicalInfo.Valid {
		reading.LiturgicalInfo = &liturgicalInfo.String
	}
	reading.SourceURL = NullString(sourceURL)

	// Parse all timestamps from TEXT
	reading.ScrapedAt = parseTimestamp(scrapedAtStr)
	if t := parseTimestamp(createdAtStr); t != nil {
		reading.CreatedAt = *t
	}
	if t := parseTimestamp(updatedAtStr); t != nil {
		reading.UpdatedAt = *t
	}

	return &reading, nil
}

// GetReadingsByDateRange retrieves readings for a date range (inclusive).
// Returns empty slice if no readings found in range.
//
// Used for /api/v1/readings/range?start=X&end=Y
func (db *DB) GetReadingsByDateRange(ctx context.Context, startDate, endDate string) ([]DailyReading, error) {
	query := `
		SELECT 
			id, date,
			morning_psalms, evening_psalms,
			first_reading, second_reading, gospel_reading,
			liturgical_info, source_url, scraped_at,
			created_at, updated_at
		FROM daily_readings
		WHERE date >= ? AND date <= ?
		ORDER BY date ASC
	`

	rows, err := db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query readings by range: %w", err)
	}
	defer rows.Close()

	var readings []DailyReading

	for rows.Next() {
		var reading DailyReading
		var morningPsalmsJSON, eveningPsalmsJSON string
		var liturgicalInfo, sourceURL, scrapedAtStr, createdAtStr, updatedAtStr sql.NullString

		err := rows.Scan(
			&reading.ID,
			&reading.Date,
			&morningPsalmsJSON,
			&eveningPsalmsJSON,
			&reading.FirstReading,
			&reading.SecondReading,
			&reading.GospelReading,
			&liturgicalInfo,
			&sourceURL,
			&scrapedAtStr,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scan reading row: %w", err)
		}

		// Unmarshal JSON psalm arrays
		reading.MorningPsalms, err = UnmarshalPsalms(morningPsalmsJSON)
		if err != nil {
			return nil, fmt.Errorf("unmarshal morning psalms: %w", err)
		}

		reading.EveningPsalms, err = UnmarshalPsalms(eveningPsalmsJSON)
		if err != nil {
			return nil, fmt.Errorf("unmarshal evening psalms: %w", err)
		}

		// Handle nullable fields
		if liturgicalInfo.Valid {
			reading.LiturgicalInfo = &liturgicalInfo.String
		}
		reading.SourceURL = NullString(sourceURL)

		// Parse all timestamps from TEXT
		reading.ScrapedAt = parseTimestamp(scrapedAtStr)
		if t := parseTimestamp(createdAtStr); t != nil {
			reading.CreatedAt = *t
		}
		if t := parseTimestamp(updatedAtStr); t != nil {
			reading.UpdatedAt = *t
		}

		readings = append(readings, reading)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reading rows: %w", err)
	}

	return readings, nil
}

// UpsertDailyReading inserts or updates a daily reading.
//
// This is IDEMPOTENT - safe to run multiple times with same data.
// Used by the scraper/importer to load data into the database.
//
// Key feature: Uses INSERT ... ON CONFLICT ... DO UPDATE
// This means:
// - If date exists: UPDATE the record
// - If date doesn't exist: INSERT a new record
// - No separate "check if exists" query needed
// - No race conditions
func (db *DB) UpsertDailyReading(ctx context.Context, reading *DailyReading) error {
	// Marshal psalm arrays to JSON
	morningPsalmsJSON, err := MarshalPsalms(reading.MorningPsalms)
	if err != nil {
		return fmt.Errorf("marshal morning psalms: %w", err)
	}

	eveningPsalmsJSON, err := MarshalPsalms(reading.EveningPsalms)
	if err != nil {
		return fmt.Errorf("marshal evening psalms: %w", err)
	}

	query := `
		INSERT INTO daily_readings (
			date, morning_psalms, evening_psalms,
			first_reading, second_reading, gospel_reading,
			liturgical_info, source_url, scraped_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(date) DO UPDATE SET
			morning_psalms = excluded.morning_psalms,
			evening_psalms = excluded.evening_psalms,
			first_reading = excluded.first_reading,
			second_reading = excluded.second_reading,
			gospel_reading = excluded.gospel_reading,
			liturgical_info = excluded.liturgical_info,
			source_url = excluded.source_url,
			scraped_at = excluded.scraped_at,
			updated_at = datetime('now')
	`

	_, err = db.ExecContext(ctx, query,
		reading.Date,
		morningPsalmsJSON,
		eveningPsalmsJSON,
		reading.FirstReading,
		reading.SecondReading,
		reading.GospelReading,
		reading.LiturgicalInfo,
		reading.SourceURL,
		TimeToNullTime(reading.ScrapedAt),
	)

	if err != nil {
		return fmt.Errorf("upsert daily reading: %w", err)
	}

	return nil
}

// DeleteDailyReading removes a reading by date.
// Returns ErrNotFound if date doesn't exist.
//
// Mainly for testing/debugging - unlikely to be used in production.
func (db *DB) DeleteDailyReading(ctx context.Context, date string) error {
	query := `DELETE FROM daily_readings WHERE date = ?`

	result, err := db.ExecContext(ctx, query, date)
	if err != nil {
		return fmt.Errorf("delete daily reading: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// GetReadingStats returns statistics about the readings in the database.
//
// Useful for:
// - Health check endpoint
// - Verifying scraper coverage
// - Dashboard/admin views
func (db *DB) GetReadingStats(ctx context.Context) (*ReadingStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_days,
			COALESCE(MIN(date), '') as earliest_date,
			COALESCE(MAX(date), '') as latest_date,
			MAX(scraped_at) as last_scraped_at
		FROM daily_readings
	`

	var stats ReadingStats
	var lastScrapedAtStr sql.NullString // SQLite stores as TEXT, not native time

	err := db.QueryRowContext(ctx, query).Scan(
		&stats.TotalDays,
		&stats.EarliestDate,
		&stats.LatestDate,
		&lastScrapedAtStr,
	)

	if err != nil {
		return nil, fmt.Errorf("query reading stats: %w", err)
	}

	// Parse the timestamp string if present
	if lastScrapedAtStr.Valid && lastScrapedAtStr.String != "" {
		t, err := time.Parse(time.RFC3339, lastScrapedAtStr.String)
		if err != nil {
			// Try parsing without timezone (SQLite datetime format)
			t, err = time.Parse("2006-01-02 15:04:05", lastScrapedAtStr.String)
			if err != nil {
				// If still fails, just leave it nil
				stats.LastScrapedAt = nil
			} else {
				stats.LastScrapedAt = &t
			}
		} else {
			stats.LastScrapedAt = &t
		}
	}

	return &stats, nil
}

// =============================================================================
// Scrape Log Queries
// =============================================================================

// LogScrapeAttempt records a scraping attempt in the scrape_log table.
//
// Used by the scraper to track:
// - Which dates were attempted
// - Success/failure status
// - Performance metrics
// - Raw data for debugging
func (db *DB) LogScrapeAttempt(ctx context.Context, entry *ScrapeLogEntry) error {
	query := `
		INSERT INTO scrape_log (
			date, source_url, raw_data, success, error_message, duration_ms
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.ExecContext(ctx, query,
		entry.Date,
		entry.SourceURL,
		entry.RawData,
		entry.Success,
		entry.ErrorMessage,
		entry.DurationMs,
	)

	if err != nil {
		return fmt.Errorf("log scrape attempt: %w", err)
	}

	return nil
}

// GetRecentScrapeLogs retrieves recent scrape log entries.
//
// Useful for:
// - Debugging scraper issues
// - Monitoring scraper health
// - Admin dashboard
func (db *DB) GetRecentScrapeLogs(ctx context.Context, limit int) ([]ScrapeLogEntry, error) {
	query := `
		SELECT id, date, scraped_at, source_url, raw_data, success, error_message, duration_ms
		FROM scrape_log
		ORDER BY scraped_at DESC
		LIMIT ?
	`

	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query scrape logs: %w", err)
	}
	defer rows.Close()

	var logs []ScrapeLogEntry

	for rows.Next() {
		var entry ScrapeLogEntry
		var rawData, errorMessage sql.NullString
		var durationMs sql.NullInt64

		err := rows.Scan(
			&entry.ID,
			&entry.Date,
			&entry.ScrapedAt,
			&entry.SourceURL,
			&rawData,
			&entry.Success,
			&errorMessage,
			&durationMs,
		)
		if err != nil {
			return nil, fmt.Errorf("scan scrape log row: %w", err)
		}

		if rawData.Valid {
			entry.RawData = &rawData.String
		}
		if errorMessage.Valid {
			entry.ErrorMessage = &errorMessage.String
		}
		if durationMs.Valid {
			entry.DurationMs = &durationMs.Int64
		}

		logs = append(logs, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scrape log rows: %w", err)
	}

	return logs, nil
}

// =============================================================================
// Progress Tracking Queries (Date-Based)
// =============================================================================

// CreateProgress marks a reading as completed for a user.
// Returns ErrDuplicate if the user has already completed this date.
func (db *DB) CreateProgress(ctx context.Context, progress *ReadingProgress) error {
	query := `
		INSERT INTO reading_progress (user_id, reading_date, notes, completed_at)
		VALUES (?, ?, ?, ?)
	`

	completedAtStr := progress.CompletedAt.Format("2006-01-02 15:04:05")

	result, err := db.ExecContext(ctx, query,
		progress.UserID,
		progress.ReadingDate,
		progress.Notes,
		completedAtStr,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return ErrDuplicate
		}
		if strings.Contains(err.Error(), "FOREIGN KEY constraint") {
			return fmt.Errorf("reading date not found in database")
		}
		return fmt.Errorf("insert progress: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	progress.ID = id
	progress.CreatedAt = time.Now()
	progress.UpdatedAt = time.Now()

	return nil
}

// GetProgressByUser retrieves a user's reading progress with pagination.
// Results are ordered by completion date (most recent first).
func (db *DB) GetProgressByUser(ctx context.Context, userID string, limit, offset int) ([]ReadingProgress, error) {
	query := `
		SELECT id, user_id, reading_date, notes, completed_at, created_at, updated_at
		FROM reading_progress
		WHERE user_id = ?
		ORDER BY completed_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query progress by user: %w", err)
	}
	defer rows.Close()

	var progressList []ReadingProgress

	for rows.Next() {
		var p ReadingProgress
		var notes sql.NullString
		var completedAtStr, createdAtStr, updatedAtStr sql.NullString

		if err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.ReadingDate,
			&notes,
			&completedAtStr,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			return nil, fmt.Errorf("scan progress: %w", err)
		}

		// Handle nullable notes
		if notes.Valid {
			p.Notes = &notes.String
		}

		// Parse timestamps
		if t := parseTimestamp(completedAtStr); t != nil {
			p.CompletedAt = *t
		}
		if t := parseTimestamp(createdAtStr); t != nil {
			p.CreatedAt = *t
		}
		if t := parseTimestamp(updatedAtStr); t != nil {
			p.UpdatedAt = *t
		}

		progressList = append(progressList, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate progress: %w", err)
	}

	return progressList, nil
}

// GetProgressByDate retrieves a progress entry for a specific user and date.
// Returns ErrNotFound if no progress exists for that date.
func (db *DB) GetProgressByDate(ctx context.Context, userID string, date string) (*ReadingProgress, error) {
	query := `
		SELECT id, user_id, reading_date, notes, completed_at, created_at, updated_at
		FROM reading_progress
		WHERE user_id = ? AND reading_date = ?
	`

	var p ReadingProgress
	var notes sql.NullString
	var completedAtStr, createdAtStr, updatedAtStr sql.NullString

	err := db.QueryRowContext(ctx, query, userID, date).Scan(
		&p.ID,
		&p.UserID,
		&p.ReadingDate,
		&notes,
		&completedAtStr,
		&createdAtStr,
		&updatedAtStr,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query progress by date: %w", err)
	}

	// Handle nullable notes
	if notes.Valid {
		p.Notes = &notes.String
	}

	// Parse timestamps
	if t := parseTimestamp(completedAtStr); t != nil {
		p.CompletedAt = *t
	}
	if t := parseTimestamp(createdAtStr); t != nil {
		p.CreatedAt = *t
	}
	if t := parseTimestamp(updatedAtStr); t != nil {
		p.UpdatedAt = *t
	}

	return &p, nil
}

// DeleteProgress removes a progress entry by date.
// Returns ErrNotFound if no progress exists for that date.
func (db *DB) DeleteProgress(ctx context.Context, userID string, date string) error {
	query := `
		DELETE FROM reading_progress 
		WHERE user_id = ? AND reading_date = ?
	`

	result, err := db.ExecContext(ctx, query, userID, date)
	if err != nil {
		return fmt.Errorf("delete progress: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetProgressStats calculates reading statistics for a user.
func (db *DB) GetProgressStats(ctx context.Context, userID string) (*ProgressStats, error) {
	// Get total days available in database
	totalQuery := `SELECT COUNT(*) FROM daily_readings`
	var totalDays int
	if err := db.QueryRowContext(ctx, totalQuery).Scan(&totalDays); err != nil {
		return nil, fmt.Errorf("count total days: %w", err)
	}

	// Get completed days count
	completedQuery := `
		SELECT COUNT(*)
		FROM reading_progress
		WHERE user_id = ?
	`
	var completedDays int
	if err := db.QueryRowContext(ctx, completedQuery, userID).Scan(&completedDays); err != nil {
		return nil, fmt.Errorf("count completed days: %w", err)
	}

	// Calculate completion percentage
	completionPercent := 0.0
	if totalDays > 0 {
		completionPercent = (float64(completedDays) / float64(totalDays)) * 100.0
	}

	// Get last completed date
	var lastCompletedDate *string
	lastDateQuery := `
		SELECT reading_date
		FROM reading_progress
		WHERE user_id = ?
		ORDER BY completed_at DESC
		LIMIT 1
	`
	var dateStr string
	err := db.QueryRowContext(ctx, lastDateQuery, userID).Scan(&dateStr)
	if err == nil {
		lastCompletedDate = &dateStr
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("get last completed date: %w", err)
	}

	// Calculate current and longest streaks
	currentStreak, longestStreak := db.calculateStreaks(ctx, userID)

	stats := &ProgressStats{
		TotalDays:         totalDays,
		CompletedDays:     completedDays,
		CompletionPercent: completionPercent,
		CurrentStreak:     currentStreak,
		LongestStreak:     longestStreak,
		LastCompletedDate: lastCompletedDate,
	}

	return stats, nil
}

// calculateStreaks calculates current and longest reading streaks.
// Current streak: consecutive days ending today or yesterday.
// Longest streak: best streak in history.
func (db *DB) calculateStreaks(ctx context.Context, userID string) (current, longest int) {
	query := `
		SELECT DATE(reading_date) as date
		FROM reading_progress
		WHERE user_id = ?
		ORDER BY date DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return 0, 0
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			continue
		}
		dates = append(dates, date)
	}

	if len(dates) == 0 {
		return 0, 0
	}

	// Calculate current streak (must end today or yesterday)
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	currentStreak := 0
	if dates[0] == today || dates[0] == yesterday {
		expectedDate, _ := time.Parse("2006-01-02", dates[0])
		currentStreak = 1

		for i := 1; i < len(dates); i++ {
			expectedDate = expectedDate.AddDate(0, 0, -1)

			if dates[i] == expectedDate.Format("2006-01-02") {
				currentStreak++
			} else {
				break
			}
		}
	}

	// Calculate longest streak
	longestStreak := currentStreak
	streak := 1

	for i := 1; i < len(dates); i++ {
		prevDate, _ := time.Parse("2006-01-02", dates[i-1])
		expectedDate := prevDate.AddDate(0, 0, -1)

		if dates[i] == expectedDate.Format("2006-01-02") {
			streak++
			if streak > longestStreak {
				longestStreak = streak
			}
		} else {
			streak = 1
		}
	}

	return currentStreak, longestStreak
}

// ============================================================================
// User Queries
// ============================================================================

// GetUserByID retrieves a user by ID.
func (db *DB) GetUserByID(ctx context.Context, id int64) (*User, error) {
	query := `
		SELECT id, username, email, full_name, active, 
		       created_at, updated_at, last_login_at
		FROM users
		WHERE id = ?
	`

	var u User
	var email, fullName sql.NullString
	var lastLoginAt sql.NullString
	var createdAtStr, updatedAtStr string

	err := db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&email,
		&fullName,
		&u.Active,
		&createdAtStr,
		&updatedAtStr,
		&lastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	if email.Valid {
		u.Email = &email.String
	}
	if fullName.Valid {
		u.FullName = &fullName.String
	}
	if t := parseTimestamp(sql.NullString{String: createdAtStr, Valid: true}); t != nil {
		u.CreatedAt = *t
	}
	if t := parseTimestamp(sql.NullString{String: updatedAtStr, Valid: true}); t != nil {
		u.UpdatedAt = *t
	}
	if t := parseTimestamp(lastLoginAt); t != nil {
		u.LastLoginAt = t
	}

	return &u, nil
}

// GetUserByUsername retrieves a user by username.
func (db *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, email, full_name, active, 
		       created_at, updated_at, last_login_at
		FROM users
		WHERE username = ?
	`

	var u User
	var email, fullName sql.NullString
	var lastLoginAt sql.NullString
	var createdAtStr, updatedAtStr string

	err := db.QueryRowContext(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&email,
		&fullName,
		&u.Active,
		&createdAtStr,
		&updatedAtStr,
		&lastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	if email.Valid {
		u.Email = &email.String
	}
	if fullName.Valid {
		u.FullName = &fullName.String
	}
	if t := parseTimestamp(sql.NullString{String: createdAtStr, Valid: true}); t != nil {
		u.CreatedAt = *t
	}
	if t := parseTimestamp(sql.NullString{String: updatedAtStr, Valid: true}); t != nil {
		u.UpdatedAt = *t
	}
	if t := parseTimestamp(lastLoginAt); t != nil {
		u.LastLoginAt = t
	}

	return &u, nil
}

// CreateUser creates a new user.
func (db *DB) CreateUser(ctx context.Context, username string, email, fullName *string) (*User, error) {
	query := `
		INSERT INTO users (username, email, full_name, active)
		VALUES (?, ?, ?, 1)
	`

	result, err := db.ExecContext(ctx, query, username, email, fullName)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	id, _ := result.LastInsertId()

	return db.GetUserByID(ctx, id)
}

// UpdateUserLastLogin updates the last_login_at timestamp.
func (db *DB) UpdateUserLastLogin(ctx context.Context, userID int64) error {
	query := `UPDATE users SET last_login_at = datetime('now') WHERE id = ?`
	_, err := db.ExecContext(ctx, query, userID)
	return err
}

// ListUsers returns all users (admin only).
func (db *DB) ListUsers(ctx context.Context) ([]User, error) {
	query := `
		SELECT id, username, email, full_name, active, 
		       created_at, updated_at, last_login_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var email, fullName sql.NullString
		var lastLoginAt sql.NullString
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&u.ID,
			&u.Username,
			&email,
			&fullName,
			&u.Active,
			&createdAtStr,
			&updatedAtStr,
			&lastLoginAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}

		if email.Valid {
			u.Email = &email.String
		}
		if fullName.Valid {
			u.FullName = &fullName.String
		}
		if t := parseTimestamp(sql.NullString{String: createdAtStr, Valid: true}); t != nil {
			u.CreatedAt = *t
		}
		if t := parseTimestamp(sql.NullString{String: updatedAtStr, Valid: true}); t != nil {
			u.UpdatedAt = *t
		}
		if t := parseTimestamp(lastLoginAt); t != nil {
			u.LastLoginAt = t
		}

		users = append(users, u)
	}

	return users, nil
}

// ============================================================================
// API Key Queries
// ============================================================================

// ValidateAPIKey checks if a key is valid and returns the user.
// Returns ErrNotFound if key doesn't exist or is inactive.
// Updates last_used_at timestamp.
func (db *DB) ValidateAPIKey(ctx context.Context, apiKey string) (*User, error) {
	// Hash the provided key
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	query := `
		SELECT u.id, u.username, u.email, u.full_name, u.active,
		       u.created_at, u.updated_at, u.last_login_at,
		       k.id as key_id
		FROM users u
		INNER JOIN api_keys k ON k.user_id = u.id
		WHERE k.key_hash = ? AND k.active = 1 AND u.active = 1
	`

	var u User
	var keyID int64
	var email, fullName sql.NullString
	var lastLoginAt sql.NullString
	var createdAtStr, updatedAtStr string

	err := db.QueryRowContext(ctx, query, keyHash).Scan(
		&u.ID,
		&u.Username,
		&email,
		&fullName,
		&u.Active,
		&createdAtStr,
		&updatedAtStr,
		&lastLoginAt,
		&keyID,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("validate api key: %w", err)
	}

	if email.Valid {
		u.Email = &email.String
	}
	if fullName.Valid {
		u.FullName = &fullName.String
	}
	if t := parseTimestamp(sql.NullString{String: createdAtStr, Valid: true}); t != nil {
		u.CreatedAt = *t
	}
	if t := parseTimestamp(sql.NullString{String: updatedAtStr, Valid: true}); t != nil {
		u.UpdatedAt = *t
	}
	if t := parseTimestamp(lastLoginAt); t != nil {
		u.LastLoginAt = t
	}

	// Update last_used_at (async, don't block)
	go func() {
		updateQuery := `UPDATE api_keys SET last_used_at = datetime('now') WHERE id = ?`
		db.ExecContext(context.Background(), updateQuery, keyID)

		// Also update user's last_login_at
		db.UpdateUserLastLogin(context.Background(), u.ID)
	}()

	return &u, nil
}

// CreateAPIKey generates and stores a new API key for a user.
// Returns the plaintext key (only time it's visible) and the record.
func (db *DB) CreateAPIKey(ctx context.Context, userID int64, name string) (*APIKeyWithPlaintext, error) {
	// Verify user exists
	_, err := db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Generate cryptographically secure random key
	keyBytes := make([]byte, 32) // 32 bytes = 64 hex chars
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("generate random key: %w", err)
	}
	plainKey := "key_" + hex.EncodeToString(keyBytes)

	// Hash for storage
	hash := sha256.Sum256([]byte(plainKey))
	keyHash := hex.EncodeToString(hash[:])

	query := `
		INSERT INTO api_keys (user_id, key_hash, name, active)
		VALUES (?, ?, ?, 1)
	`

	result, err := db.ExecContext(ctx, query, userID, keyHash, name)
	if err != nil {
		return nil, fmt.Errorf("insert api key: %w", err)
	}

	id, _ := result.LastInsertId()

	return &APIKeyWithPlaintext{
		APIKey: APIKey{
			ID:        id,
			UserID:    userID,
			KeyHash:   keyHash,
			Name:      name,
			Active:    true,
			CreatedAt: time.Now(),
		},
		PlaintextKey: plainKey,
	}, nil
}

// ListUserAPIKeys returns all API keys for a user.
func (db *DB) ListUserAPIKeys(ctx context.Context, userID int64) ([]APIKey, error) {
	query := `
		SELECT id, user_id, key_hash, name, active, 
		       created_at, last_used_at, revoked_at
		FROM api_keys
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list user api keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var createdAtStr string
		var lastUsedAt, revokedAt sql.NullString

		err := rows.Scan(
			&k.ID,
			&k.UserID,
			&k.KeyHash,
			&k.Name,
			&k.Active,
			&createdAtStr,
			&lastUsedAt,
			&revokedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}

		if t := parseTimestamp(sql.NullString{String: createdAtStr, Valid: true}); t != nil {
			k.CreatedAt = *t
		}
		if t := parseTimestamp(lastUsedAt); t != nil {
			k.LastUsedAt = t
		}
		if t := parseTimestamp(revokedAt); t != nil {
			k.RevokedAt = t
		}

		keys = append(keys, k)
	}

	return keys, nil
}

// RevokeAPIKey marks an API key as inactive.
func (db *DB) RevokeAPIKey(ctx context.Context, keyID int64, userID int64) error {
	query := `
		UPDATE api_keys 
		SET active = 0, revoked_at = datetime('now')
		WHERE id = ? AND user_id = ?
	`

	result, err := db.ExecContext(ctx, query, keyID, userID)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
