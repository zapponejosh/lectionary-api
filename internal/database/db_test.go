package database

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

// testDB creates a temporary in-memory database for testing.
func testDB(t *testing.T) *DB {
	t.Helper()

	// Use in-memory database for tests
	cfg := Config{
		Path:            ":memory:",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
	}

	// Quiet logger for tests
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	db, err := Open(cfg, logger)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	// Run migrations
	ctx := context.Background()
	if _, err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// seedTestData inserts sample data for testing.
func seedTestData(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()

	// Create a liturgical day
	day := &LiturgicalDay{
		Date:           "2025-01-01",
		Weekday:        3, // Wednesday
		YearCycle:      1,
		Season:         string(SeasonChristmas),
		SpecialDayName: strPtr("New Year's Day"),
	}
	if err := db.CreateDay(ctx, day); err != nil {
		t.Fatalf("create test day: %v", err)
	}

	// Create readings for the day
	readings := []Reading{
		{DayID: day.ID, ReadingType: ReadingTypeMorningPsalm, Position: 1, Reference: "Pss. 98; 147:1–11"},
		{DayID: day.ID, ReadingType: ReadingTypeEveningPsalm, Position: 1, Reference: "Pss. 99; 8"},
		{DayID: day.ID, ReadingType: ReadingTypeOldTestament, Position: 1, Reference: "Gen. 17:1–12a, 15–16"},
		{DayID: day.ID, ReadingType: ReadingTypeEpistle, Position: 1, Reference: "Col. 2:6–12"},
		{DayID: day.ID, ReadingType: ReadingTypeGospel, Position: 1, Reference: "John 16:23b–30"},
	}

	for i := range readings {
		if err := db.CreateReading(ctx, &readings[i]); err != nil {
			t.Fatalf("create test reading: %v", err)
		}
	}
}

func strPtr(s string) *string {
	return &s
}

// -----------------------------------------------------------------
// DB tests
// -----------------------------------------------------------------

func TestOpen(t *testing.T) {
	db := testDB(t)

	// Verify connection works
	ctx := context.Background()
	if err := db.Health(ctx); err != nil {
		t.Errorf("Health() error = %v", err)
	}
}

func TestMigrate(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	// Migrations should have run (in testDB)
	// Running again should be a no-op
	count, err := db.Migrate(ctx)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Migrate() count = %d, want 0 (already applied)", count)
	}
}

// -----------------------------------------------------------------
// LiturgicalDay tests
// -----------------------------------------------------------------

func TestCreateDay(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	day := &LiturgicalDay{
		Date:           "2025-03-05",
		Weekday:        3,
		YearCycle:      1,
		Season:         string(SeasonLent),
		SpecialDayName: strPtr("Ash Wednesday"),
	}

	err := db.CreateDay(ctx, day)
	if err != nil {
		t.Fatalf("CreateDay() error = %v", err)
	}

	if day.ID == 0 {
		t.Error("CreateDay() did not set ID")
	}
}

func TestCreateDay_Duplicate(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	day := &LiturgicalDay{
		Date:      "2025-03-05",
		Weekday:   3,
		YearCycle: 1,
		Season:    string(SeasonLent),
	}

	if err := db.CreateDay(ctx, day); err != nil {
		t.Fatalf("first CreateDay() error = %v", err)
	}

	// Try to create duplicate
	day2 := &LiturgicalDay{
		Date:      "2025-03-05",
		Weekday:   3,
		YearCycle: 1,
		Season:    string(SeasonLent),
	}

	err := db.CreateDay(ctx, day2)
	if err != ErrDuplicate {
		t.Errorf("CreateDay() duplicate error = %v, want ErrDuplicate", err)
	}
}

func TestGetDayByDate(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	day, err := db.GetDayByDate(ctx, "2025-01-01")
	if err != nil {
		t.Fatalf("GetDayByDate() error = %v", err)
	}

	if day.Date != "2025-01-01" {
		t.Errorf("GetDayByDate() date = %q, want %q", day.Date, "2025-01-01")
	}
	if day.Season != string(SeasonChristmas) {
		t.Errorf("GetDayByDate() season = %q, want %q", day.Season, SeasonChristmas)
	}
	if day.SpecialDayName == nil || *day.SpecialDayName != "New Year's Day" {
		t.Errorf("GetDayByDate() special_day_name = %v, want %q", day.SpecialDayName, "New Year's Day")
	}
}

func TestGetDayByDate_NotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	_, err := db.GetDayByDate(ctx, "9999-12-31")
	if err != ErrNotFound {
		t.Errorf("GetDayByDate() error = %v, want ErrNotFound", err)
	}
}

func TestGetDaysInRange(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	// Create several days
	dates := []string{"2025-01-01", "2025-01-02", "2025-01-03", "2025-01-04", "2025-01-05"}
	for i, date := range dates {
		day := &LiturgicalDay{
			Date:      date,
			Weekday:   i % 7,
			YearCycle: 1,
			Season:    string(SeasonChristmas),
		}
		if err := db.CreateDay(ctx, day); err != nil {
			t.Fatalf("create day %s: %v", date, err)
		}
	}

	// Query range
	days, err := db.GetDaysInRange(ctx, "2025-01-02", "2025-01-04")
	if err != nil {
		t.Fatalf("GetDaysInRange() error = %v", err)
	}

	if len(days) != 3 {
		t.Errorf("GetDaysInRange() returned %d days, want 3", len(days))
	}

	// Verify order
	if len(days) >= 2 && days[0].Date > days[1].Date {
		t.Error("GetDaysInRange() days not in ascending order")
	}
}

// -----------------------------------------------------------------
// Reading tests
// -----------------------------------------------------------------

func TestGetReadingsByDayID(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	// Get the day first
	day, err := db.GetDayByDate(ctx, "2025-01-01")
	if err != nil {
		t.Fatalf("get day: %v", err)
	}

	readings, err := db.GetReadingsByDayID(ctx, day.ID)
	if err != nil {
		t.Fatalf("GetReadingsByDayID() error = %v", err)
	}

	if len(readings) != 5 {
		t.Errorf("GetReadingsByDayID() returned %d readings, want 5", len(readings))
	}

	// Verify ordering (morning psalms first, then evening, OT, epistle, gospel)
	expectedOrder := []ReadingType{
		ReadingTypeMorningPsalm,
		ReadingTypeEveningPsalm,
		ReadingTypeOldTestament,
		ReadingTypeEpistle,
		ReadingTypeGospel,
	}
	for i, reading := range readings {
		if reading.ReadingType != expectedOrder[i] {
			t.Errorf("reading[%d].ReadingType = %q, want %q", i, reading.ReadingType, expectedOrder[i])
		}
	}
}

func TestCreateReading(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	// Create a day first
	day := &LiturgicalDay{
		Date:      "2025-01-10",
		Weekday:   5,
		YearCycle: 1,
		Season:    string(SeasonEpiphany),
	}
	if err := db.CreateDay(ctx, day); err != nil {
		t.Fatalf("create day: %v", err)
	}

	// Create a reading
	reading := &Reading{
		DayID:         day.ID,
		ReadingType:   ReadingTypeMorningPsalm,
		Position:      1,
		Reference:     "Pss. 46; 47",
		IsAlternative: false,
	}

	err := db.CreateReading(ctx, reading)
	if err != nil {
		t.Fatalf("CreateReading() error = %v", err)
	}

	if reading.ID == 0 {
		t.Error("CreateReading() did not set ID")
	}

	// Verify we can retrieve it
	retrieved, err := db.GetReadingByID(ctx, reading.ID)
	if err != nil {
		t.Fatalf("GetReadingByID() error = %v", err)
	}

	if retrieved.Reference != "Pss. 46; 47" {
		t.Errorf("retrieved reading reference = %q, want %q", retrieved.Reference, "Pss. 46; 47")
	}
}

// -----------------------------------------------------------------
// DailyReadings combined tests
// -----------------------------------------------------------------

func TestGetDailyReadings(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	daily, err := db.GetDailyReadings(ctx, "2025-01-01")
	if err != nil {
		t.Fatalf("GetDailyReadings() error = %v", err)
	}

	if daily.Day.Date != "2025-01-01" {
		t.Errorf("GetDailyReadings() day.date = %q, want %q", daily.Day.Date, "2025-01-01")
	}

	if len(daily.Readings) != 5 {
		t.Errorf("GetDailyReadings() returned %d readings, want 5", len(daily.Readings))
	}
}

func TestGetDailyReadings_NotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	_, err := db.GetDailyReadings(ctx, "9999-12-31")
	if err != ErrNotFound {
		t.Errorf("GetDailyReadings() error = %v, want ErrNotFound", err)
	}
}

// -----------------------------------------------------------------
// Progress tests
// -----------------------------------------------------------------

func TestMarkReadingComplete(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	// Get a reading
	day, _ := db.GetDayByDate(ctx, "2025-01-01")
	readings, _ := db.GetReadingsByDayID(ctx, day.ID)
	reading := readings[0]

	// Mark it complete
	notes := "Great psalm!"
	progress, err := db.MarkReadingComplete(ctx, "test-user", reading.ID, &notes)
	if err != nil {
		t.Fatalf("MarkReadingComplete() error = %v", err)
	}

	if progress.ID == 0 {
		t.Error("MarkReadingComplete() did not set ID")
	}
	if progress.UserID != "test-user" {
		t.Errorf("progress.UserID = %q, want %q", progress.UserID, "test-user")
	}
	if progress.Notes == nil || *progress.Notes != "Great psalm!" {
		t.Errorf("progress.Notes = %v, want %q", progress.Notes, "Great psalm!")
	}
}

func TestMarkReadingComplete_Duplicate(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	day, _ := db.GetDayByDate(ctx, "2025-01-01")
	readings, _ := db.GetReadingsByDayID(ctx, day.ID)
	reading := readings[0]

	// Mark complete first time
	_, err := db.MarkReadingComplete(ctx, "test-user", reading.ID, nil)
	if err != nil {
		t.Fatalf("first MarkReadingComplete() error = %v", err)
	}

	// Try to mark again
	_, err = db.MarkReadingComplete(ctx, "test-user", reading.ID, nil)
	if err != ErrDuplicate {
		t.Errorf("second MarkReadingComplete() error = %v, want ErrDuplicate", err)
	}
}

func TestUnmarkReadingComplete(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	day, _ := db.GetDayByDate(ctx, "2025-01-01")
	readings, _ := db.GetReadingsByDayID(ctx, day.ID)
	reading := readings[0]

	// Mark then unmark
	db.MarkReadingComplete(ctx, "test-user", reading.ID, nil)

	err := db.UnmarkReadingComplete(ctx, "test-user", reading.ID)
	if err != nil {
		t.Fatalf("UnmarkReadingComplete() error = %v", err)
	}

	// Verify it's gone
	progress, _ := db.GetProgressForReadings(ctx, "test-user", []int64{reading.ID})
	if _, exists := progress[reading.ID]; exists {
		t.Error("UnmarkReadingComplete() did not remove progress")
	}
}

func TestUnmarkReadingComplete_NotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	err := db.UnmarkReadingComplete(ctx, "test-user", 99999)
	if err != ErrNotFound {
		t.Errorf("UnmarkReadingComplete() error = %v, want ErrNotFound", err)
	}
}

func TestGetUserProgress(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	day, _ := db.GetDayByDate(ctx, "2025-01-01")
	readings, _ := db.GetReadingsByDayID(ctx, day.ID)

	// Mark several readings
	for i := 0; i < 3; i++ {
		db.MarkReadingComplete(ctx, "test-user", readings[i].ID, nil)
	}

	// Get progress
	progress, err := db.GetUserProgress(ctx, "test-user", 10, 0)
	if err != nil {
		t.Fatalf("GetUserProgress() error = %v", err)
	}

	if len(progress) != 3 {
		t.Errorf("GetUserProgress() returned %d records, want 3", len(progress))
	}
}

func TestGetProgressForReadings(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	day, _ := db.GetDayByDate(ctx, "2025-01-01")
	readings, _ := db.GetReadingsByDayID(ctx, day.ID)

	// Mark first reading complete
	db.MarkReadingComplete(ctx, "test-user", readings[0].ID, nil)

	// Get progress for first two readings
	readingIDs := []int64{readings[0].ID, readings[1].ID}
	progress, err := db.GetProgressForReadings(ctx, "test-user", readingIDs)
	if err != nil {
		t.Fatalf("GetProgressForReadings() error = %v", err)
	}

	// First should exist, second should not
	if _, exists := progress[readings[0].ID]; !exists {
		t.Error("GetProgressForReadings() missing progress for reading 0")
	}
	if _, exists := progress[readings[1].ID]; exists {
		t.Error("GetProgressForReadings() has unexpected progress for reading 1")
	}
}

func TestGetUserStats(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	day, _ := db.GetDayByDate(ctx, "2025-01-01")
	readings, _ := db.GetReadingsByDayID(ctx, day.ID)

	// Mark 2 of 5 readings complete
	db.MarkReadingComplete(ctx, "test-user", readings[0].ID, nil)
	db.MarkReadingComplete(ctx, "test-user", readings[1].ID, nil)

	stats, err := db.GetUserStats(ctx, "test-user")
	if err != nil {
		t.Fatalf("GetUserStats() error = %v", err)
	}

	if stats.TotalReadings != 5 {
		t.Errorf("stats.TotalReadings = %d, want 5", stats.TotalReadings)
	}
	if stats.CompletedReadings != 2 {
		t.Errorf("stats.CompletedReadings = %d, want 2", stats.CompletedReadings)
	}
	if stats.CompletionPercent != 40.0 {
		t.Errorf("stats.CompletionPercent = %f, want 40.0", stats.CompletionPercent)
	}
}

// -----------------------------------------------------------------
// Model validation tests
// -----------------------------------------------------------------

func TestReadingType_IsValid(t *testing.T) {
	tests := []struct {
		rt   ReadingType
		want bool
	}{
		{ReadingTypeMorningPsalm, true},
		{ReadingTypeEveningPsalm, true},
		{ReadingTypeOldTestament, true},
		{ReadingTypeEpistle, true},
		{ReadingTypeGospel, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.rt), func(t *testing.T) {
			if got := tt.rt.IsValid(); got != tt.want {
				t.Errorf("ReadingType(%q).IsValid() = %v, want %v", tt.rt, got, tt.want)
			}
		})
	}
}

func TestSeason_IsValid(t *testing.T) {
	tests := []struct {
		s    Season
		want bool
	}{
		{SeasonAdvent, true},
		{SeasonChristmas, true},
		{SeasonEpiphany, true},
		{SeasonLent, true},
		{SeasonHolyWeek, true},
		{SeasonEaster, true},
		{SeasonPentecost, true},
		{SeasonOrdinary, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.s), func(t *testing.T) {
			if got := tt.s.IsValid(); got != tt.want {
				t.Errorf("Season(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------
// Transaction tests
// -----------------------------------------------------------------

func TestWithTx(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	// Successful transaction
	err := db.WithTx(ctx, func(tx *Tx) error {
		day := &LiturgicalDay{
			Date:      "2025-06-01",
			Weekday:   0,
			YearCycle: 1,
			Season:    string(SeasonOrdinary),
		}
		return tx.CreateDay(ctx, day)
	})
	if err != nil {
		t.Fatalf("WithTx() success case error = %v", err)
	}

	// Verify day was created
	day, err := db.GetDayByDate(ctx, "2025-06-01")
	if err != nil {
		t.Errorf("day not created: %v", err)
	}
	if day == nil {
		t.Error("day is nil")
	}
}

func TestWithTx_Rollback(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	// Failed transaction should rollback
	err := db.WithTx(ctx, func(tx *Tx) error {
		day := &LiturgicalDay{
			Date:      "2025-06-02",
			Weekday:   1,
			YearCycle: 1,
			Season:    string(SeasonOrdinary),
		}
		if err := tx.CreateDay(ctx, day); err != nil {
			return err
		}
		// Force error to trigger rollback
		return ErrNotFound
	})
	if err != ErrNotFound {
		t.Fatalf("WithTx() rollback case error = %v, want ErrNotFound", err)
	}

	// Verify day was NOT created
	_, err = db.GetDayByDate(ctx, "2025-06-02")
	if err != ErrNotFound {
		t.Errorf("day should not exist after rollback, got error: %v", err)
	}
}
