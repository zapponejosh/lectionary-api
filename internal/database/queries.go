package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// -----------------------------------------------------------------
// LiturgicalDay queries
// -----------------------------------------------------------------

// GetDayByDate retrieves a liturgical day by its date.
// Returns ErrNotFound if no day exists for the given date.
func (db *DB) GetDayByDate(ctx context.Context, date string) (*LiturgicalDay, error) {
	query := `
		SELECT id, date, weekday, year_cycle, season, special_day_name, 
		       created_at, updated_at
		FROM liturgical_days
		WHERE date = ?
	`

	var day LiturgicalDay
	var createdAt, updatedAt string

	err := db.QueryRowContext(ctx, query, date).Scan(
		&day.ID,
		&day.Date,
		&day.Weekday,
		&day.YearCycle,
		&day.Season,
		&day.SpecialDayName,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query day by date: %w", err)
	}

	// Parse timestamps
	day.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	day.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)

	return &day, nil
}

// GetDayByID retrieves a liturgical day by its ID.
func (db *DB) GetDayByID(ctx context.Context, id int64) (*LiturgicalDay, error) {
	query := `
		SELECT id, date, weekday, year_cycle, season, special_day_name,
		       created_at, updated_at
		FROM liturgical_days
		WHERE id = ?
	`

	var day LiturgicalDay
	var createdAt, updatedAt string

	err := db.QueryRowContext(ctx, query, id).Scan(
		&day.ID,
		&day.Date,
		&day.Weekday,
		&day.YearCycle,
		&day.Season,
		&day.SpecialDayName,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query day by id: %w", err)
	}

	day.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	day.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)

	return &day, nil
}

// GetDaysInRange retrieves all liturgical days within a date range (inclusive).
func (db *DB) GetDaysInRange(ctx context.Context, startDate, endDate string) ([]LiturgicalDay, error) {
	query := `
		SELECT id, date, weekday, year_cycle, season, special_day_name,
		       created_at, updated_at
		FROM liturgical_days
		WHERE date >= ? AND date <= ?
		ORDER BY date ASC
	`

	rows, err := db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query days in range: %w", err)
	}
	defer rows.Close()

	var days []LiturgicalDay
	for rows.Next() {
		var day LiturgicalDay
		var createdAt, updatedAt string

		if err := rows.Scan(
			&day.ID,
			&day.Date,
			&day.Weekday,
			&day.YearCycle,
			&day.Season,
			&day.SpecialDayName,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan day: %w", err)
		}

		day.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
		day.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)
		days = append(days, day)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate days: %w", err)
	}

	return days, nil
}

// CreateDay inserts a new liturgical day.
func (db *DB) CreateDay(ctx context.Context, day *LiturgicalDay) error {
	query := `
		INSERT INTO liturgical_days (date, weekday, year_cycle, season, special_day_name)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := db.ExecContext(ctx, query,
		day.Date,
		day.Weekday,
		day.YearCycle,
		day.Season,
		day.SpecialDayName,
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

// -----------------------------------------------------------------
// Reading queries
// -----------------------------------------------------------------

// GetReadingsByDayID retrieves all readings for a specific day.
func (db *DB) GetReadingsByDayID(ctx context.Context, dayID int64) ([]Reading, error) {
	query := `
		SELECT id, day_id, reading_type, position, reference, is_alternative,
		       created_at, updated_at
		FROM readings
		WHERE day_id = ?
		ORDER BY 
			CASE reading_type
				WHEN 'morning_psalm' THEN 1
				WHEN 'evening_psalm' THEN 2
				WHEN 'old_testament' THEN 3
				WHEN 'epistle' THEN 4
				WHEN 'gospel' THEN 5
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
		SELECT id, day_id, reading_type, position, reference, is_alternative,
		       created_at, updated_at
		FROM readings
		WHERE id = ?
	`

	var r Reading
	var createdAt, updatedAt string

	err := db.QueryRowContext(ctx, query, id).Scan(
		&r.ID,
		&r.DayID,
		&r.ReadingType,
		&r.Position,
		&r.Reference,
		&r.IsAlternative,
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

// CreateReading inserts a new reading.
func (db *DB) CreateReading(ctx context.Context, r *Reading) error {
	query := `
		INSERT INTO readings (day_id, reading_type, position, reference, is_alternative)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := db.ExecContext(ctx, query,
		r.DayID,
		r.ReadingType,
		r.Position,
		r.Reference,
		r.IsAlternative,
	)
	if err != nil {
		return fmt.Errorf("insert reading: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	r.ID = id
	return nil
}

// CreateReadingTx inserts a new reading within a transaction.
func (tx *Tx) CreateReading(ctx context.Context, r *Reading) error {
	query := `
		INSERT INTO readings (day_id, reading_type, position, reference, is_alternative)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := tx.ExecContext(ctx, query,
		r.DayID,
		r.ReadingType,
		r.Position,
		r.Reference,
		r.IsAlternative,
	)
	if err != nil {
		return fmt.Errorf("insert reading: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	r.ID = id
	return nil
}

// -----------------------------------------------------------------
// Combined day + readings queries
// -----------------------------------------------------------------

// GetDailyReadings retrieves a day and all its readings by date.
// This is the primary method for the API endpoints.
func (db *DB) GetDailyReadings(ctx context.Context, date string) (*DailyReadings, error) {
	day, err := db.GetDayByDate(ctx, date)
	if err != nil {
		return nil, err
	}

	readings, err := db.GetReadingsByDayID(ctx, day.ID)
	if err != nil {
		return nil, err
	}

	return &DailyReadings{
		Day:      *day,
		Readings: readings,
	}, nil
}

// GetDailyReadingsRange retrieves days and readings for a date range.
func (db *DB) GetDailyReadingsRange(ctx context.Context, startDate, endDate string) ([]DailyReadings, error) {
	days, err := db.GetDaysInRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	result := make([]DailyReadings, len(days))
	for i, day := range days {
		readings, err := db.GetReadingsByDayID(ctx, day.ID)
		if err != nil {
			return nil, fmt.Errorf("get readings for %s: %w", day.Date, err)
		}
		result[i] = DailyReadings{
			Day:      day,
			Readings: readings,
		}
	}

	return result, nil
}

// -----------------------------------------------------------------
// Reading Progress queries
// -----------------------------------------------------------------

// MarkReadingComplete creates a progress record for a reading.
// Returns ErrDuplicate if already marked complete.
func (db *DB) MarkReadingComplete(ctx context.Context, userID string, readingID int64, notes *string) (*ReadingProgress, error) {
	query := `
		INSERT INTO reading_progress (user_id, reading_id, notes)
		VALUES (?, ?, ?)
	`

	result, err := db.ExecContext(ctx, query, userID, readingID, notes)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("insert progress: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	// Retrieve the created record to get timestamps
	return db.GetProgressByID(ctx, id)
}

// UnmarkReadingComplete removes a progress record.
func (db *DB) UnmarkReadingComplete(ctx context.Context, userID string, readingID int64) error {
	query := `DELETE FROM reading_progress WHERE user_id = ? AND reading_id = ?`

	result, err := db.ExecContext(ctx, query, userID, readingID)
	if err != nil {
		return fmt.Errorf("delete progress: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// GetProgressByID retrieves a progress record by its ID.
func (db *DB) GetProgressByID(ctx context.Context, id int64) (*ReadingProgress, error) {
	query := `
		SELECT id, user_id, reading_id, notes, completed_at, created_at, updated_at
		FROM reading_progress
		WHERE id = ?
	`

	var p ReadingProgress
	var completedAt, createdAt, updatedAt string

	err := db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.UserID,
		&p.ReadingID,
		&p.Notes,
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

	p.CompletedAt, _ = time.Parse(time.DateTime, completedAt)
	p.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	p.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)

	return &p, nil
}

// GetUserProgress retrieves all progress records for a user.
func (db *DB) GetUserProgress(ctx context.Context, userID string, limit, offset int) ([]ReadingProgress, error) {
	query := `
		SELECT id, user_id, reading_id, notes, completed_at, created_at, updated_at
		FROM reading_progress
		WHERE user_id = ?
		ORDER BY completed_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query user progress: %w", err)
	}
	defer rows.Close()

	var progress []ReadingProgress
	for rows.Next() {
		var p ReadingProgress
		var completedAt, createdAt, updatedAt string

		if err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.ReadingID,
			&p.Notes,
			&completedAt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan progress: %w", err)
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

// GetProgressForReadings retrieves progress for specific reading IDs.
// Returns a map of reading_id -> ReadingProgress for easy lookup.
func (db *DB) GetProgressForReadings(ctx context.Context, userID string, readingIDs []int64) (map[int64]*ReadingProgress, error) {
	if len(readingIDs) == 0 {
		return make(map[int64]*ReadingProgress), nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(readingIDs))
	args := make([]interface{}, len(readingIDs)+1)
	args[0] = userID
	for i, id := range readingIDs {
		placeholders[i] = "?"
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, reading_id, notes, completed_at, created_at, updated_at
		FROM reading_progress
		WHERE user_id = ? AND reading_id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query progress for readings: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]*ReadingProgress)
	for rows.Next() {
		var p ReadingProgress
		var completedAt, createdAt, updatedAt string

		if err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.ReadingID,
			&p.Notes,
			&completedAt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan progress: %w", err)
		}

		p.CompletedAt, _ = time.Parse(time.DateTime, completedAt)
		p.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
		p.UpdatedAt, _ = time.Parse(time.DateTime, updatedAt)
		result[p.ReadingID] = &p
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate progress: %w", err)
	}

	return result, nil
}

// GetUserStats calculates reading statistics for a user.
func (db *DB) GetUserStats(ctx context.Context, userID string) (*ProgressStats, error) {
	// Get total readings count
	var total int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM readings").Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count total readings: %w", err)
	}

	// Get completed readings count
	var completed int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM reading_progress WHERE user_id = ?",
		userID,
	).Scan(&completed)
	if err != nil {
		return nil, fmt.Errorf("count completed readings: %w", err)
	}

	// Calculate completion percentage
	var percent float64
	if total > 0 {
		percent = float64(completed) / float64(total) * 100
	}

	// Calculate current streak (consecutive days with at least one reading)
	currentStreak, longestStreak, err := db.calculateStreaks(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &ProgressStats{
		TotalReadings:     total,
		CompletedReadings: completed,
		CompletionPercent: percent,
		CurrentStreak:     currentStreak,
		LongestStreak:     longestStreak,
	}, nil
}

// calculateStreaks calculates current and longest reading streaks for a user.
// A streak is counted as consecutive days where the user completed at least one reading.
func (db *DB) calculateStreaks(ctx context.Context, userID string) (current, longest int, err error) {
	// Get distinct dates when user completed readings, ordered by date descending
	query := `
		SELECT DISTINCT date(rp.completed_at) as read_date
		FROM reading_progress rp
		WHERE rp.user_id = ?
		ORDER BY read_date DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("query reading dates: %w", err)
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return 0, 0, fmt.Errorf("scan date: %w", err)
		}
		t, _ := time.Parse("2006-01-02", dateStr)
		dates = append(dates, t)
	}

	if len(dates) == 0 {
		return 0, 0, nil
	}

	// Calculate streaks
	today := time.Now().Truncate(24 * time.Hour)

	// Check if current streak is active (includes today or yesterday)
	if dates[0].Equal(today) || dates[0].Equal(today.AddDate(0, 0, -1)) {
		current = 1
		for i := 1; i < len(dates); i++ {
			expected := dates[i-1].AddDate(0, 0, -1)
			if dates[i].Equal(expected) {
				current++
			} else {
				break
			}
		}
	}

	// Calculate longest streak
	longest = 1
	streak := 1
	for i := 1; i < len(dates); i++ {
		expected := dates[i-1].AddDate(0, 0, -1)
		if dates[i].Equal(expected) {
			streak++
			if streak > longest {
				longest = streak
			}
		} else {
			streak = 1
		}
	}

	// Ensure current is counted in longest
	if current > longest {
		longest = current
	}

	return current, longest, nil
}

// -----------------------------------------------------------------
// Helper functions
// -----------------------------------------------------------------

// scanReadings scans multiple reading rows into a slice.
func scanReadings(rows *sql.Rows) ([]Reading, error) {
	var readings []Reading
	for rows.Next() {
		var r Reading
		var createdAt, updatedAt string

		if err := rows.Scan(
			&r.ID,
			&r.DayID,
			&r.ReadingType,
			&r.Position,
			&r.Reference,
			&r.IsAlternative,
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

// -----------------------------------------------------------------
// Bulk import helpers (for PDF import tool)
// -----------------------------------------------------------------

// CreateDayTx creates a liturgical day within a transaction.
func (tx *Tx) CreateDay(ctx context.Context, day *LiturgicalDay) error {
	query := `
		INSERT INTO liturgical_days (date, weekday, year_cycle, season, special_day_name)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := tx.ExecContext(ctx, query,
		day.Date,
		day.Weekday,
		day.YearCycle,
		day.Season,
		day.SpecialDayName,
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

// ClearAllData removes all data from the tables (for reimport).
// Use with caution!
func (db *DB) ClearAllData(ctx context.Context) error {
	tables := []string{"reading_progress", "readings", "liturgical_days"}

	for _, table := range tables {
		if _, err := db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}

	return nil
}
