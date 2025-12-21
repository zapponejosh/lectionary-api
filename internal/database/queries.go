package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// LectionaryDay Queries
// =============================================================================

// GetDayByPosition retrieves a lectionary day by its position (period + day_identifier).
// This is the primary lookup for the API.
func (db *DB) GetDayByPosition(ctx context.Context, period, dayIdentifier string) (*LectionaryDay, error) {
	query := `
		SELECT id, period, day_identifier, period_type, special_name,
		       morning_psalms, evening_psalms, created_at, updated_at
		FROM lectionary_days
		WHERE period = ? AND day_identifier = ?
	`

	return db.scanLectionaryDay(db.QueryRowContext(ctx, query, period, dayIdentifier))
}

// GetDayByID retrieves a lectionary day by its ID.
func (db *DB) GetDayByID(ctx context.Context, id int64) (*LectionaryDay, error) {
	query := `
		SELECT id, period, day_identifier, period_type, special_name,
		       morning_psalms, evening_psalms, created_at, updated_at
		FROM lectionary_days
		WHERE id = ?
	`

	return db.scanLectionaryDay(db.QueryRowContext(ctx, query, id))
}

// GetDayBySpecialName retrieves a lectionary day by its special name.
// Useful for looking up "Christmas Day", "Epiphany", etc.
func (db *DB) GetDayBySpecialName(ctx context.Context, specialName string) (*LectionaryDay, error) {
	query := `
		SELECT id, period, day_identifier, period_type, special_name,
		       morning_psalms, evening_psalms, created_at, updated_at
		FROM lectionary_days
		WHERE special_name = ?
	`

	return db.scanLectionaryDay(db.QueryRowContext(ctx, query, specialName))
}

// GetDaysByPeriod retrieves all days for a given period.
// Returns days ordered by a sensible day order (Sunday first, then weekdays).
func (db *DB) GetDaysByPeriod(ctx context.Context, period string) ([]LectionaryDay, error) {
	query := `
		SELECT id, period, day_identifier, period_type, special_name,
		       morning_psalms, evening_psalms, created_at, updated_at
		FROM lectionary_days
		WHERE period = ?
		ORDER BY 
			CASE day_identifier
				WHEN 'Sunday' THEN 1
				WHEN 'Monday' THEN 2
				WHEN 'Tuesday' THEN 3
				WHEN 'Wednesday' THEN 4
				WHEN 'Thursday' THEN 5
				WHEN 'Friday' THEN 6
				WHEN 'Saturday' THEN 7
				ELSE 8
			END
	`

	rows, err := db.QueryContext(ctx, query, period)
	if err != nil {
		return nil, fmt.Errorf("query days by period: %w", err)
	}
	defer rows.Close()

	return scanLectionaryDays(rows)
}

// GetDaysByPeriodType retrieves all days of a given period type.
func (db *DB) GetDaysByPeriodType(ctx context.Context, periodType PeriodType) ([]LectionaryDay, error) {
	query := `
		SELECT id, period, day_identifier, period_type, special_name,
		       morning_psalms, evening_psalms, created_at, updated_at
		FROM lectionary_days
		WHERE period_type = ?
		ORDER BY period, day_identifier
	`

	rows, err := db.QueryContext(ctx, query, periodType)
	if err != nil {
		return nil, fmt.Errorf("query days by period type: %w", err)
	}
	defer rows.Close()

	return scanLectionaryDays(rows)
}

// GetAllDays retrieves all lectionary days.
func (db *DB) GetAllDays(ctx context.Context) ([]LectionaryDay, error) {
	query := `
		SELECT id, period, day_identifier, period_type, special_name,
		       morning_psalms, evening_psalms, created_at, updated_at
		FROM lectionary_days
		ORDER BY id
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all days: %w", err)
	}
	defer rows.Close()

	return scanLectionaryDays(rows)
}

// CountDays returns the total number of lectionary days.
func (db *DB) CountDays(ctx context.Context) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM lectionary_days").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count days: %w", err)
	}
	return count, nil
}

// =============================================================================
// Reading Queries
// =============================================================================

// GetReadingsByDayAndYear retrieves all readings for a position and year cycle.
// This is the primary lookup for serving API requests.
func (db *DB) GetReadingsByDayAndYear(ctx context.Context, dayID int64, yearCycle int) ([]Reading, error) {
	query := `
		SELECT id, lectionary_day_id, year_cycle, reading_type, position, reference,
		       created_at, updated_at
		FROM readings
		WHERE lectionary_day_id = ? AND year_cycle = ?
		ORDER BY 
			CASE reading_type
				WHEN 'first' THEN 1
				WHEN 'second' THEN 2
				WHEN 'gospel' THEN 3
			END,
			position ASC
	`

	rows, err := db.QueryContext(ctx, query, dayID, yearCycle)
	if err != nil {
		return nil, fmt.Errorf("query readings by day and year: %w", err)
	}
	defer rows.Close()

	return scanReadings(rows)
}

// GetReadingsByDayID retrieves all readings for a position (both years).
func (db *DB) GetReadingsByDayID(ctx context.Context, dayID int64) ([]Reading, error) {
	query := `
		SELECT id, lectionary_day_id, year_cycle, reading_type, position, reference,
		       created_at, updated_at
		FROM readings
		WHERE lectionary_day_id = ?
		ORDER BY year_cycle,
			CASE reading_type
				WHEN 'first' THEN 1
				WHEN 'second' THEN 2
				WHEN 'gospel' THEN 3
			END,
			position ASC
	`

	rows, err := db.QueryContext(ctx, query, dayID)
	if err != nil {
		return nil, fmt.Errorf("query readings by day: %w", err)
	}
	defer rows.Close()

	return scanReadings(rows)
}

// GetReadingByID retrieves a specific reading by ID.
func (db *DB) GetReadingByID(ctx context.Context, id int64) (*Reading, error) {
	query := `
		SELECT id, lectionary_day_id, year_cycle, reading_type, position, reference,
		       created_at, updated_at
		FROM readings
		WHERE id = ?
	`

	var r Reading
	var createdAt, updatedAt string

	err := db.QueryRowContext(ctx, query, id).Scan(
		&r.ID,
		&r.LectionaryDayID,
		&r.YearCycle,
		&r.ReadingType,
		&r.Position,
		&r.Reference,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query reading by id: %w", err)
	}

	r.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	r.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)

	return &r, nil
}

// CountReadings returns the total number of readings.
func (db *DB) CountReadings(ctx context.Context) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM readings").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count readings: %w", err)
	}
	return count, nil
}

// CountReadingsByYear returns the count of readings for each year cycle.
func (db *DB) CountReadingsByYear(ctx context.Context) (year1, year2 int, err error) {
	query := `
		SELECT 
			SUM(CASE WHEN year_cycle = 1 THEN 1 ELSE 0 END),
			SUM(CASE WHEN year_cycle = 2 THEN 1 ELSE 0 END)
		FROM readings
	`
	err = db.QueryRowContext(ctx, query).Scan(&year1, &year2)
	if err != nil {
		return 0, 0, fmt.Errorf("count readings by year: %w", err)
	}
	return year1, year2, nil
}

// =============================================================================
// Combined Queries (for API responses)
// =============================================================================

// GetDailyReadings retrieves the complete readings for a position and year.
// This combines the lectionary day with its readings into a single response.
func (db *DB) GetDailyReadings(ctx context.Context, period, dayIdentifier string, yearCycle int) (*DailyReadings, error) {
	day, err := db.GetDayByPosition(ctx, period, dayIdentifier)
	if err != nil {
		return nil, err
	}

	readings, err := db.GetReadingsByDayAndYear(ctx, day.ID, yearCycle)
	if err != nil {
		return nil, err
	}

	return &DailyReadings{
		Period:        day.Period,
		DayIdentifier: day.DayIdentifier,
		SpecialName:   day.SpecialName,
		MorningPsalms: day.MorningPsalms,
		EveningPsalms: day.EveningPsalms,
		YearCycle:     yearCycle,
		Readings:      readings,
	}, nil
}

// =============================================================================
// Create Operations (for import)
// =============================================================================

// CreateDay inserts a new lectionary day.
func (db *DB) CreateDay(ctx context.Context, day *LectionaryDay) error {
	query := `
		INSERT INTO lectionary_days (period, day_identifier, period_type, special_name,
		                             morning_psalms, evening_psalms)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := db.ExecContext(ctx, query,
		day.Period,
		day.DayIdentifier,
		day.PeriodType,
		day.SpecialName,
		day.MorningPsalmsJSON(),
		day.EveningPsalmsJSON(),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return ErrDuplicate
		}
		return fmt.Errorf("insert day: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	day.ID = id
	return nil
}

// CreateReading inserts a new reading.
func (db *DB) CreateReading(ctx context.Context, reading *Reading) error {
	query := `
		INSERT INTO readings (lectionary_day_id, year_cycle, reading_type, position, reference)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := db.ExecContext(ctx, query,
		reading.LectionaryDayID,
		reading.YearCycle,
		reading.ReadingType,
		reading.Position,
		reading.Reference,
	)
	if err != nil {
		return fmt.Errorf("insert reading: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	reading.ID = id
	return nil
}

// =============================================================================
// Transaction-based Create Operations (for bulk import)
// =============================================================================

// CreateDayTx inserts a new lectionary day within a transaction.
func (tx *Tx) CreateDay(ctx context.Context, day *LectionaryDay) error {
	query := `
		INSERT INTO lectionary_days (period, day_identifier, period_type, special_name,
		                             morning_psalms, evening_psalms)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := tx.ExecContext(ctx, query,
		day.Period,
		day.DayIdentifier,
		day.PeriodType,
		day.SpecialName,
		day.MorningPsalmsJSON(),
		day.EveningPsalmsJSON(),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return ErrDuplicate
		}
		return fmt.Errorf("insert day: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	day.ID = id
	return nil
}

// CreateReading inserts a new reading within a transaction.
func (tx *Tx) CreateReading(ctx context.Context, reading *Reading) error {
	query := `
		INSERT INTO readings (lectionary_day_id, year_cycle, reading_type, position, reference)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := tx.ExecContext(ctx, query,
		reading.LectionaryDayID,
		reading.YearCycle,
		reading.ReadingType,
		reading.Position,
		reading.Reference,
	)
	if err != nil {
		return fmt.Errorf("insert reading: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	reading.ID = id
	return nil
}

// =============================================================================
// Reading Progress Queries
// =============================================================================

// GetProgressByUser retrieves a user's reading progress with pagination.
func (db *DB) GetProgressByUser(ctx context.Context, userID string, limit, offset int) ([]ReadingProgress, error) {
	query := `
		SELECT id, user_id, reading_id, notes, completed_at, created_at, updated_at
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

	var progress []ReadingProgress
	for rows.Next() {
		var p ReadingProgress
		var notes sql.NullString
		var completedAt, createdAt, updatedAt string

		if err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.ReadingID,
			&notes,
			&completedAt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan progress: %w", err)
		}

		if notes.Valid {
			p.Notes = &notes.String
		}
		p.CompletedAt, _ = time.Parse(time.DateTime, completedAt)
		p.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
		p.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)

		progress = append(progress, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate progress: %w", err)
	}

	return progress, nil
}

// GetProgressByID retrieves a progress entry by ID.
func (db *DB) GetProgressByID(ctx context.Context, id int64) (*ReadingProgress, error) {
	query := `
		SELECT id, user_id, reading_id, notes, completed_at, created_at, updated_at
		FROM reading_progress
		WHERE id = ?
	`

	var p ReadingProgress
	var notes sql.NullString
	var completedAt, createdAt, updatedAt string

	err := db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.UserID,
		&p.ReadingID,
		&notes,
		&completedAt,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query progress by id: %w", err)
	}

	if notes.Valid {
		p.Notes = &notes.String
	}
	p.CompletedAt, _ = time.Parse(time.DateTime, completedAt)
	p.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	p.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)

	return &p, nil
}

// CreateProgress inserts a new progress entry.
func (db *DB) CreateProgress(ctx context.Context, progress *ReadingProgress) error {
	query := `
		INSERT INTO reading_progress (user_id, reading_id, notes, completed_at)
		VALUES (?, ?, ?, ?)
	`

	result, err := db.ExecContext(ctx, query,
		progress.UserID,
		progress.ReadingID,
		progress.Notes,
		progress.CompletedAt.Format(time.DateTime),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return ErrDuplicate
		}
		return fmt.Errorf("insert progress: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	progress.ID = id
	return nil
}

// DeleteProgress deletes a progress entry.
func (db *DB) DeleteProgress(ctx context.Context, id int64) error {
	query := `DELETE FROM reading_progress WHERE id = ?`

	result, err := db.ExecContext(ctx, query, id)
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

// GetProgressStats calculates statistics for a user's reading progress.
func (db *DB) GetProgressStats(ctx context.Context, userID string) (*ProgressStats, error) {
	// Get total readings count
	totalQuery := `SELECT COUNT(*) FROM readings`
	var totalReadings int
	if err := db.QueryRowContext(ctx, totalQuery).Scan(&totalReadings); err != nil {
		return nil, fmt.Errorf("count total readings: %w", err)
	}

	// Get completed readings count
	completedQuery := `
		SELECT COUNT(DISTINCT reading_id)
		FROM reading_progress
		WHERE user_id = ?
	`
	var completedReadings int
	if err := db.QueryRowContext(ctx, completedQuery, userID).Scan(&completedReadings); err != nil {
		return nil, fmt.Errorf("count completed readings: %w", err)
	}

	// Calculate completion percentage
	completionPercent := 0.0
	if totalReadings > 0 {
		completionPercent = (float64(completedReadings) / float64(totalReadings)) * 100.0
	}

	// Calculate current streak (consecutive days with completed readings)
	currentStreak := 0
	streakQuery := `
		SELECT DATE(completed_at) as date, COUNT(*) as count
		FROM reading_progress
		WHERE user_id = ?
		GROUP BY DATE(completed_at)
		ORDER BY date DESC
	`
	rows, err := db.QueryContext(ctx, streakQuery, userID)
	if err == nil {
		defer rows.Close()
		expectedDate := time.Now()
		for rows.Next() {
			var dateStr string
			var count int
			if err := rows.Scan(&dateStr, &count); err != nil {
				break
			}
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				break
			}

			// Check if this date is consecutive
			daysDiff := int(expectedDate.Sub(date).Hours() / 24)
			if daysDiff == 0 || (currentStreak == 0 && daysDiff <= 1) {
				currentStreak++
				expectedDate = date.AddDate(0, 0, -1)
			} else if daysDiff > 1 {
				break
			}
		}
	}

	// Calculate longest streak (simplified - could be optimized)
	longestStreak := currentStreak // For MVP, use current streak

	stats := &ProgressStats{
		TotalReadings:     totalReadings,
		CompletedReadings: completedReadings,
		CompletionPercent: completionPercent,
		CurrentStreak:     currentStreak,
		LongestStreak:     longestStreak,
	}

	return stats, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// scanLectionaryDay scans a single lectionary day from a row.
func (db *DB) scanLectionaryDay(row *sql.Row) (*LectionaryDay, error) {
	var day LectionaryDay
	var morningPsalms, eveningPsalms string
	var createdAt, updatedAt string

	err := row.Scan(
		&day.ID,
		&day.Period,
		&day.DayIdentifier,
		&day.PeriodType,
		&day.SpecialName,
		&morningPsalms,
		&eveningPsalms,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan lectionary day: %w", err)
	}

	// Parse JSON arrays
	day.MorningPsalms, _ = ParsePsalmsJSON(morningPsalms)
	day.EveningPsalms, _ = ParsePsalmsJSON(eveningPsalms)

	// Parse timestamps
	day.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	day.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)

	return &day, nil
}

// scanLectionaryDays scans multiple lectionary days from rows.
func scanLectionaryDays(rows *sql.Rows) ([]LectionaryDay, error) {
	var days []LectionaryDay
	for rows.Next() {
		var day LectionaryDay
		var morningPsalms, eveningPsalms string
		var createdAt, updatedAt string

		if err := rows.Scan(
			&day.ID,
			&day.Period,
			&day.DayIdentifier,
			&day.PeriodType,
			&day.SpecialName,
			&morningPsalms,
			&eveningPsalms,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan lectionary day: %w", err)
		}

		day.MorningPsalms, _ = ParsePsalmsJSON(morningPsalms)
		day.EveningPsalms, _ = ParsePsalmsJSON(eveningPsalms)
		day.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
		day.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)

		days = append(days, day)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate days: %w", err)
	}

	return days, nil
}

// scanReadings scans multiple readings from rows.
func scanReadings(rows *sql.Rows) ([]Reading, error) {
	var readings []Reading
	for rows.Next() {
		var r Reading
		var createdAt, updatedAt string

		if err := rows.Scan(
			&r.ID,
			&r.LectionaryDayID,
			&r.YearCycle,
			&r.ReadingType,
			&r.Position,
			&r.Reference,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reading: %w", err)
		}

		r.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
		r.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)
		readings = append(readings, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate readings: %w", err)
	}

	return readings, nil
}
