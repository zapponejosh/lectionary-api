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

	// Create a lectionary day (position-based)
	day := &LectionaryDay{
		Period:        "1st Week of Advent",
		DayIdentifier: "Sunday",
		PeriodType:    PeriodTypeLiturgical,
		SpecialName:   strPtr("First Sunday of Advent"),
		MorningPsalms: []string{"24", "150"},
		EveningPsalms: []string{"25", "110"},
	}
	if err := db.CreateDay(ctx, day); err != nil {
		t.Fatalf("create test day: %v", err)
	}

	// Create readings for Year 1
	readings := []Reading{
		{LectionaryDayID: day.ID, YearCycle: 1, ReadingType: ReadingTypeFirst, Position: 1, Reference: "Isaiah 1:1–9"},
		{LectionaryDayID: day.ID, YearCycle: 1, ReadingType: ReadingTypeSecond, Position: 1, Reference: "Col. 2:6–12"},
		{LectionaryDayID: day.ID, YearCycle: 1, ReadingType: ReadingTypeGospel, Position: 1, Reference: "John 16:23b–30"},
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

	day := &LectionaryDay{
		Period:        "1st Week of Lent",
		DayIdentifier: "Wednesday",
		PeriodType:    PeriodTypeLiturgical,
		SpecialName:   strPtr("Ash Wednesday"),
		MorningPsalms: []string{"51"},
		EveningPsalms: []string{"32", "143"},
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

	day := &LectionaryDay{
		Period:        "1st Week of Lent",
		DayIdentifier: "Wednesday",
		PeriodType:    PeriodTypeLiturgical,
		MorningPsalms: []string{"51"},
		EveningPsalms: []string{"32"},
	}

	if err := db.CreateDay(ctx, day); err != nil {
		t.Fatalf("first CreateDay() error = %v", err)
	}

	// Try to create duplicate (same period + day_identifier)
	day2 := &LectionaryDay{
		Period:        "1st Week of Lent",
		DayIdentifier: "Wednesday",
		PeriodType:    PeriodTypeLiturgical,
		MorningPsalms: []string{"52"},
		EveningPsalms: []string{"33"},
	}

	err := db.CreateDay(ctx, day2)
	if err != ErrDuplicate {
		t.Errorf("CreateDay() duplicate error = %v, want ErrDuplicate", err)
	}
}

func TestGetDayByPosition(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	day, err := db.GetDayByPosition(ctx, "1st Week of Advent", "Sunday")
	if err != nil {
		t.Fatalf("GetDayByPosition() error = %v", err)
	}

	if day.Period != "1st Week of Advent" {
		t.Errorf("GetDayByPosition() period = %q, want %q", day.Period, "1st Week of Advent")
	}
	if day.DayIdentifier != "Sunday" {
		t.Errorf("GetDayByPosition() day_identifier = %q, want %q", day.DayIdentifier, "Sunday")
	}
	if day.PeriodType != PeriodTypeLiturgical {
		t.Errorf("GetDayByPosition() period_type = %q, want %q", day.PeriodType, PeriodTypeLiturgical)
	}
	if day.SpecialName == nil || *day.SpecialName != "First Sunday of Advent" {
		t.Errorf("GetDayByPosition() special_name = %v, want %q", day.SpecialName, "First Sunday of Advent")
	}
	if len(day.MorningPsalms) != 2 || day.MorningPsalms[0] != "24" {
		t.Errorf("GetDayByPosition() morning_psalms = %v, want [\"24\", \"150\"]", day.MorningPsalms)
	}
}

func TestGetDayByPosition_NotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	_, err := db.GetDayByPosition(ctx, "Non-existent Period", "Monday")
	if err != ErrNotFound {
		t.Errorf("GetDayByPosition() error = %v, want ErrNotFound", err)
	}
}

func TestGetDaysByPeriod(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	// Create several days in the same period
	dayIdentifiers := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday"}
	for _, dayID := range dayIdentifiers {
		day := &LectionaryDay{
			Period:        "1st Week of Advent",
			DayIdentifier: dayID,
			PeriodType:    PeriodTypeLiturgical,
			MorningPsalms: []string{"24"},
			EveningPsalms: []string{"25"},
		}
		if err := db.CreateDay(ctx, day); err != nil {
			t.Fatalf("create day %s: %v", dayID, err)
		}
	}

	// Query by period
	days, err := db.GetDaysByPeriod(ctx, "1st Week of Advent")
	if err != nil {
		t.Fatalf("GetDaysByPeriod() error = %v", err)
	}

	if len(days) != 5 {
		t.Errorf("GetDaysByPeriod() returned %d days, want 5", len(days))
	}

	// Verify order (Sunday first)
	if len(days) >= 1 && days[0].DayIdentifier != "Sunday" {
		t.Errorf("GetDaysByPeriod() first day = %q, want %q", days[0].DayIdentifier, "Sunday")
	}
}

// -----------------------------------------------------------------
// Reading tests
// -----------------------------------------------------------------

func TestGetReadingsByDayAndYear(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	// Get the day first
	day, err := db.GetDayByPosition(ctx, "1st Week of Advent", "Sunday")
	if err != nil {
		t.Fatalf("get day: %v", err)
	}

	readings, err := db.GetReadingsByDayAndYear(ctx, day.ID, 1)
	if err != nil {
		t.Fatalf("GetReadingsByDayAndYear() error = %v", err)
	}

	if len(readings) != 3 {
		t.Errorf("GetReadingsByDayAndYear() returned %d readings, want 3", len(readings))
	}

	// Verify ordering (first, second, gospel)
	expectedOrder := []ReadingType{
		ReadingTypeFirst,
		ReadingTypeSecond,
		ReadingTypeGospel,
	}
	for i, reading := range readings {
		if reading.ReadingType != expectedOrder[i] {
			t.Errorf("reading[%d].ReadingType = %q, want %q", i, reading.ReadingType, expectedOrder[i])
		}
		if reading.YearCycle != 1 {
			t.Errorf("reading[%d].YearCycle = %d, want 1", i, reading.YearCycle)
		}
	}
}

func TestGetReadingsByDayID(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	// Get the day first
	day, err := db.GetDayByPosition(ctx, "1st Week of Advent", "Sunday")
	if err != nil {
		t.Fatalf("get day: %v", err)
	}

	// Add Year 2 readings
	year2Readings := []Reading{
		{LectionaryDayID: day.ID, YearCycle: 2, ReadingType: ReadingTypeFirst, Position: 1, Reference: "Jeremiah 1:1–10"},
		{LectionaryDayID: day.ID, YearCycle: 2, ReadingType: ReadingTypeSecond, Position: 1, Reference: "1 Thess. 1:1–10"},
		{LectionaryDayID: day.ID, YearCycle: 2, ReadingType: ReadingTypeGospel, Position: 1, Reference: "Matt. 3:1–12"},
	}
	for i := range year2Readings {
		if err := db.CreateReading(ctx, &year2Readings[i]); err != nil {
			t.Fatalf("create year 2 reading: %v", err)
		}
	}

	// Get all readings (both years)
	readings, err := db.GetReadingsByDayID(ctx, day.ID)
	if err != nil {
		t.Fatalf("GetReadingsByDayID() error = %v", err)
	}

	if len(readings) != 6 {
		t.Errorf("GetReadingsByDayID() returned %d readings, want 6 (3 for each year)", len(readings))
	}

	// Verify ordering: year 1 first, then year 2
	if readings[0].YearCycle != 1 {
		t.Errorf("first reading year_cycle = %d, want 1", readings[0].YearCycle)
	}
	if readings[3].YearCycle != 2 {
		t.Errorf("fourth reading year_cycle = %d, want 2", readings[3].YearCycle)
	}
}

func TestCreateReading(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	// Create a day first
	day := &LectionaryDay{
		Period:        "2nd Week of Epiphany",
		DayIdentifier: "Monday",
		PeriodType:    PeriodTypeLiturgical,
		MorningPsalms: []string{"46"},
		EveningPsalms: []string{"47"},
	}
	if err := db.CreateDay(ctx, day); err != nil {
		t.Fatalf("create day: %v", err)
	}

	// Create a reading
	reading := &Reading{
		LectionaryDayID: day.ID,
		YearCycle:       1,
		ReadingType:     ReadingTypeFirst,
		Position:        1,
		Reference:       "Isaiah 42:1–9",
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

	if retrieved.Reference != "Isaiah 42:1–9" {
		t.Errorf("retrieved reading reference = %q, want %q", retrieved.Reference, "Isaiah 42:1–9")
	}
	if retrieved.YearCycle != 1 {
		t.Errorf("retrieved reading year_cycle = %d, want 1", retrieved.YearCycle)
	}
}

// -----------------------------------------------------------------
// DailyReadings combined tests
// -----------------------------------------------------------------

func TestGetDailyReadings(t *testing.T) {
	db := testDB(t)
	seedTestData(t, db)
	ctx := context.Background()

	daily, err := db.GetDailyReadings(ctx, "1st Week of Advent", "Sunday", 1)
	if err != nil {
		t.Fatalf("GetDailyReadings() error = %v", err)
	}

	if daily.Period != "1st Week of Advent" {
		t.Errorf("GetDailyReadings() period = %q, want %q", daily.Period, "1st Week of Advent")
	}
	if daily.DayIdentifier != "Sunday" {
		t.Errorf("GetDailyReadings() day_identifier = %q, want %q", daily.DayIdentifier, "Sunday")
	}
	if daily.YearCycle != 1 {
		t.Errorf("GetDailyReadings() year_cycle = %d, want 1", daily.YearCycle)
	}

	if len(daily.Readings) != 3 {
		t.Errorf("GetDailyReadings() returned %d readings, want 3", len(daily.Readings))
	}
	if len(daily.MorningPsalms) != 2 {
		t.Errorf("GetDailyReadings() morning_psalms = %v, want 2 psalms", daily.MorningPsalms)
	}
}

func TestGetDailyReadings_NotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	_, err := db.GetDailyReadings(ctx, "Non-existent Period", "Monday", 1)
	if err != ErrNotFound {
		t.Errorf("GetDailyReadings() error = %v, want ErrNotFound", err)
	}
}

// -----------------------------------------------------------------
// Progress tests
// -----------------------------------------------------------------
// Note: Progress tracking methods have been removed from the current
// implementation. These tests are commented out but kept for reference
// in case progress tracking is re-implemented in the future.
//
// func TestMarkReadingComplete(t *testing.T) { ... }
// func TestUnmarkReadingComplete(t *testing.T) { ... }
// func TestGetUserProgress(t *testing.T) { ... }
// func TestGetProgressForReadings(t *testing.T) { ... }
// func TestGetUserStats(t *testing.T) { ... }

// -----------------------------------------------------------------
// Model validation tests
// -----------------------------------------------------------------

func TestReadingType_IsValid(t *testing.T) {
	tests := []struct {
		rt   ReadingType
		want bool
	}{
		{ReadingTypeFirst, true},
		{ReadingTypeSecond, true},
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

func TestPeriodType_IsValid(t *testing.T) {
	tests := []struct {
		pt   PeriodType
		want bool
	}{
		{PeriodTypeLiturgical, true},
		{PeriodTypeDated, true},
		{PeriodTypeFixed, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.pt), func(t *testing.T) {
			if got := tt.pt.IsValid(); got != tt.want {
				t.Errorf("PeriodType(%q).IsValid() = %v, want %v", tt.pt, got, tt.want)
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
		day := &LectionaryDay{
			Period:        "10th Week of Ordinary Time",
			DayIdentifier: "Sunday",
			PeriodType:    PeriodTypeLiturgical,
			MorningPsalms: []string{"24"},
			EveningPsalms: []string{"25"},
		}
		return tx.CreateDay(ctx, day)
	})
	if err != nil {
		t.Fatalf("WithTx() success case error = %v", err)
	}

	// Verify day was created
	day, err := db.GetDayByPosition(ctx, "10th Week of Ordinary Time", "Sunday")
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
		day := &LectionaryDay{
			Period:        "11th Week of Ordinary Time",
			DayIdentifier: "Monday",
			PeriodType:    PeriodTypeLiturgical,
			MorningPsalms: []string{"24"},
			EveningPsalms: []string{"25"},
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
	_, err = db.GetDayByPosition(ctx, "11th Week of Ordinary Time", "Monday")
	if err != ErrNotFound {
		t.Errorf("day should not exist after rollback, got error: %v", err)
	}
}
