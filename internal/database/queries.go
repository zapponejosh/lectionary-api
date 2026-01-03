package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// =============================================================================
// Error Types
// =============================================================================

var (
	// ErrNotFound is returned when a requested record doesn't exist
	ErrNotFound = errors.New("not found")
)

// IsNotFound checks if an error is a not-found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

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
