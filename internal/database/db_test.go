package database

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"
)

// =============================================================================
// TEST SETUP HELPERS
// =============================================================================

// setupTestDB creates an in-memory database for testing.
// Returns the database and a cleanup function.
//
// WHY IN-MEMORY?
// - Fast: No disk I/O
// - Isolated: Each test gets a fresh database
// - Clean: Automatically destroyed when test ends
// - No cleanup needed: No leftover files
func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	// Create a logger that only shows errors during tests
	// This keeps test output clean
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Config for in-memory database
	// ":memory:" is SQLite's special string for in-memory databases
	cfg := Config{
		Path:            ":memory:",
		MaxOpenConns:    1, // In-memory DBs should use 1 connection
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
	}

	db, err := Open(cfg, logger)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Return database and cleanup function
	return db, func() {
		db.Close()
	}
}

// =============================================================================
// CONNECTION & HEALTH TESTS
// =============================================================================

func TestOpen_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("expected database to be non-nil")
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Try to open database in non-existent directory
	cfg := Config{
		Path:            "/nonexistent/directory/test.db",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
	}

	_, err := Open(cfg, logger)
	if err == nil {
		t.Error("expected error when opening database at invalid path")
	}
}

func TestHealth_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := db.Health(ctx); err != nil {
		t.Errorf("health check failed: %v", err)
	}
}

func TestHealth_AfterClose(t *testing.T) {
	db, cleanup := setupTestDB(t)
	cleanup() // Close immediately

	ctx := context.Background()
	if err := db.Health(ctx); err == nil {
		t.Error("expected health check to fail on closed database")
	}
}

// =============================================================================
// MIGRATION TESTS
// =============================================================================

func TestMigrate_FreshDatabase(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Run migrations
	count, err := db.Migrate(ctx)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Should apply all 3 migrations
	if count != 3 {
		t.Errorf("applied %d migrations, want 3", count)
	}

	// Verify schema_migrations table exists and has correct entries
	var migrationCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount)
	if err != nil {
		t.Fatalf("failed to query migrations: %v", err)
	}

	if migrationCount != 3 {
		t.Errorf("schema_migrations has %d entries, want 3", migrationCount)
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Run migrations twice
	count1, err := db.Migrate(ctx)
	if err != nil {
		t.Fatalf("first migration failed: %v", err)
	}

	count2, err := db.Migrate(ctx)
	if err != nil {
		t.Fatalf("second migration failed: %v", err)
	}

	// First run should apply all migrations
	if count1 != 3 {
		t.Errorf("first run applied %d migrations, want 3", count1)
	}

	// Second run should apply zero migrations
	if count2 != 0 {
		t.Errorf("second run applied %d migrations, want 0 (already applied)", count2)
	}
}

func TestMigrate_CreatesAllTables(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	_, err := db.Migrate(ctx)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Expected tables after all migrations
	expectedTables := []string{
		"schema_migrations",
		"daily_readings",
		"scrape_log",
		"reading_progress",
		"users",
		"api_keys",
	}

	for _, table := range expectedTables {
		var count int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
		err := db.QueryRowContext(ctx, query, table).Scan(&count)
		if err != nil {
			t.Errorf("failed to check for table %s: %v", table, err)
			continue
		}

		if count != 1 {
			t.Errorf("table %s not found (count=%d)", table, count)
		}
	}
}

// =============================================================================
// DAILY READINGS CRUD TESTS
// =============================================================================

func TestGetReadingByDate_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	_, err := db.GetReadingByDate(ctx, "2025-12-25")
	if !IsNotFound(err) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpsertDailyReading_Insert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create a reading
	reading := &DailyReading{
		Date:          "2025-01-01",
		MorningPsalms: []string{"1", "2"},
		EveningPsalms: []string{"3", "4"},
		FirstReading:  "Genesis 1:1-5",
		SecondReading: "Romans 1:1-7",
		GospelReading: "John 1:1-14",
		SourceURL:     "https://example.com",
	}

	err := db.UpsertDailyReading(ctx, reading)
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	// Retrieve and verify
	retrieved, err := db.GetReadingByDate(ctx, "2025-01-01")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if retrieved.Date != reading.Date {
		t.Errorf("Date = %q, want %q", retrieved.Date, reading.Date)
	}
	if retrieved.FirstReading != reading.FirstReading {
		t.Errorf("FirstReading = %q, want %q", retrieved.FirstReading, reading.FirstReading)
	}
	if len(retrieved.MorningPsalms) != 2 {
		t.Errorf("MorningPsalms length = %d, want 2", len(retrieved.MorningPsalms))
	}
}

func TestUpsertDailyReading_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Insert initial reading
	reading := &DailyReading{
		Date:          "2025-01-01",
		MorningPsalms: []string{"1"},
		EveningPsalms: []string{"2"},
		FirstReading:  "Genesis 1:1",
		SecondReading: "Romans 1:1",
		GospelReading: "John 1:1",
		SourceURL:     "https://example.com/v1",
	}

	db.UpsertDailyReading(ctx, reading)

	// Update with new data
	reading.FirstReading = "Genesis 1:1-10"
	reading.SourceURL = "https://example.com/v2"
	reading.MorningPsalms = []string{"1", "2", "3"}

	err := db.UpsertDailyReading(ctx, reading)
	if err != nil {
		t.Fatalf("upsert update failed: %v", err)
	}

	// Retrieve and verify update
	retrieved, err := db.GetReadingByDate(ctx, "2025-01-01")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if retrieved.FirstReading != "Genesis 1:1-10" {
		t.Errorf("FirstReading not updated: %q", retrieved.FirstReading)
	}
	if retrieved.SourceURL != "https://example.com/v2" {
		t.Errorf("SourceURL not updated: %q", retrieved.SourceURL)
	}
	if len(retrieved.MorningPsalms) != 3 {
		t.Errorf("MorningPsalms not updated: got %d psalms, want 3", len(retrieved.MorningPsalms))
	}
}

func TestGetReadingsByDateRange(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Insert multiple readings
	dates := []string{
		"2025-01-01",
		"2025-01-02",
		"2025-01-03",
		"2025-01-05", // Gap on 01-04
		"2025-01-10",
	}

	for _, date := range dates {
		reading := &DailyReading{
			Date:          date,
			MorningPsalms: []string{"1"},
			EveningPsalms: []string{"2"},
			FirstReading:  "Genesis 1:1",
			SecondReading: "Romans 1:1",
			GospelReading: "John 1:1",
			SourceURL:     "https://example.com",
		}
		db.UpsertDailyReading(ctx, reading)
	}

	// Test range query
	readings, err := db.GetReadingsByDateRange(ctx, "2025-01-01", "2025-01-05")
	if err != nil {
		t.Fatalf("get range failed: %v", err)
	}

	// Should get 4 readings (01-01, 01-02, 01-03, 01-05)
	if len(readings) != 4 {
		t.Errorf("got %d readings, want 4", len(readings))
	}

	// Verify order (should be ascending)
	if len(readings) > 0 && readings[0].Date != "2025-01-01" {
		t.Errorf("first reading date = %q, want 2025-01-01", readings[0].Date)
	}
}

func TestDeleteDailyReading_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Insert a reading
	reading := &DailyReading{
		Date:          "2025-01-01",
		MorningPsalms: []string{"1"},
		EveningPsalms: []string{"2"},
		FirstReading:  "Genesis 1:1",
		SecondReading: "Romans 1:1",
		GospelReading: "John 1:1",
		SourceURL:     "https://example.com",
	}
	db.UpsertDailyReading(ctx, reading)

	// Delete it
	err := db.DeleteDailyReading(ctx, "2025-01-01")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify it's gone
	_, err = db.GetReadingByDate(ctx, "2025-01-01")
	if !IsNotFound(err) {
		t.Error("reading still exists after delete")
	}
}

func TestDeleteDailyReading_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	err := db.DeleteDailyReading(ctx, "2099-12-31")
	if !IsNotFound(err) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetReadingStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Empty database
	stats, err := db.GetReadingStats(ctx)
	if err != nil {
		t.Fatalf("get stats failed: %v", err)
	}

	if stats.TotalDays != 0 {
		t.Errorf("empty db: TotalDays = %d, want 0", stats.TotalDays)
	}

	// Insert some readings
	for i := 1; i <= 5; i++ {
		reading := &DailyReading{
			Date:          "2025-01-0" + string(rune('0'+i)),
			MorningPsalms: []string{"1"},
			EveningPsalms: []string{"2"},
			FirstReading:  "Genesis 1:1",
			SecondReading: "Romans 1:1",
			GospelReading: "John 1:1",
			SourceURL:     "https://example.com",
		}
		db.UpsertDailyReading(ctx, reading)
	}

	// Check stats again
	stats, err = db.GetReadingStats(ctx)
	if err != nil {
		t.Fatalf("get stats failed: %v", err)
	}

	if stats.TotalDays != 5 {
		t.Errorf("TotalDays = %d, want 5", stats.TotalDays)
	}
	if stats.EarliestDate != "2025-01-01" {
		t.Errorf("EarliestDate = %q, want 2025-01-01", stats.EarliestDate)
	}
	if stats.LatestDate != "2025-01-05" {
		t.Errorf("LatestDate = %q, want 2025-01-05", stats.LatestDate)
	}
}

// =============================================================================
// USER CRUD TESTS
// =============================================================================

func TestCreateUser_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	email := "test@example.com"
	fullName := "Test User"

	user, err := db.CreateUser(ctx, "testuser", &email, &fullName)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	if user.ID == 0 {
		t.Error("user ID is 0")
	}
	if user.Username != "testuser" {
		t.Errorf("username = %q, want testuser", user.Username)
	}
	if user.Email == nil || *user.Email != email {
		t.Errorf("email not set correctly")
	}
	if !user.Active {
		t.Error("user should be active by default")
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create first user
	email1 := "user1@example.com"
	_, err := db.CreateUser(ctx, "duplicate", &email1, nil)
	if err != nil {
		t.Fatalf("first user creation failed: %v", err)
	}

	// Try to create second user with same username
	email2 := "user2@example.com"
	_, err = db.CreateUser(ctx, "duplicate", &email2, nil)
	if err != ErrDuplicate {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestGetUserByID_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user
	email := "test@example.com"
	created, _ := db.CreateUser(ctx, "testuser", &email, nil)

	// Retrieve by ID
	user, err := db.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}

	if user.ID != created.ID {
		t.Errorf("ID = %d, want %d", user.ID, created.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("username = %q, want testuser", user.Username)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	_, err := db.GetUserByID(ctx, 99999)
	if !IsNotFound(err) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetUserByUsername_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	email := "test@example.com"
	db.CreateUser(ctx, "testuser", &email, nil)

	user, err := db.GetUserByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("username = %q, want testuser", user.Username)
	}
}

func TestListUsers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create multiple users
	for i := 1; i <= 3; i++ {
		username := "user" + string(rune('0'+i))
		email := username + "@example.com"
		db.CreateUser(ctx, username, &email, nil)
	}

	users, err := db.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users failed: %v", err)
	}

	if len(users) != 3 {
		t.Errorf("got %d users, want 3", len(users))
	}
}

func TestUpdateUserLastLogin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user
	email := "test@example.com"
	user, _ := db.CreateUser(ctx, "testuser", &email, nil)

	// Initially no last login
	if user.LastLoginAt != nil {
		t.Error("new user should have nil LastLoginAt")
	}

	// Update last login
	time.Sleep(10 * time.Millisecond) // Small delay to ensure timestamp is different
	err := db.UpdateUserLastLogin(ctx, user.ID)
	if err != nil {
		t.Fatalf("update last login failed: %v", err)
	}

	// Verify update
	updated, _ := db.GetUserByID(ctx, user.ID)
	if updated.LastLoginAt == nil {
		t.Error("LastLoginAt should be set after update")
	}
}

// =============================================================================
// API KEY CRUD TESTS
// =============================================================================

func TestCreateAPIKey_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user first
	email := "test@example.com"
	user, _ := db.CreateUser(ctx, "testuser", &email, nil)

	// Create API key
	keyWithPlaintext, err := db.CreateAPIKey(ctx, user.ID, "Test Device")
	if err != nil {
		t.Fatalf("create api key failed: %v", err)
	}

	// Verify plaintext key is returned
	if keyWithPlaintext.PlaintextKey == "" {
		t.Error("plaintext key should not be empty")
	}

	// Key should start with "key_"
	if len(keyWithPlaintext.PlaintextKey) < 40 {
		t.Error("plaintext key seems too short")
	}

	// Verify key metadata
	if keyWithPlaintext.Name != "Test Device" {
		t.Errorf("name = %q, want 'Test Device'", keyWithPlaintext.Name)
	}
	if !keyWithPlaintext.Active {
		t.Error("new key should be active")
	}
}

func TestValidateAPIKey_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user and key
	email := "test@example.com"
	user, _ := db.CreateUser(ctx, "testuser", &email, nil)
	keyWithPlaintext, _ := db.CreateAPIKey(ctx, user.ID, "Test Device")

	// Validate the key
	validatedUser, err := db.ValidateAPIKey(ctx, keyWithPlaintext.PlaintextKey)
	if err != nil {
		t.Fatalf("validate api key failed: %v", err)
	}

	if validatedUser.ID != user.ID {
		t.Errorf("validated user ID = %d, want %d", validatedUser.ID, user.ID)
	}
}

func TestValidateAPIKey_Invalid(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Try to validate non-existent key
	_, err := db.ValidateAPIKey(ctx, "key_invalid_does_not_exist_1234567890")
	if !IsNotFound(err) {
		t.Errorf("expected ErrNotFound for invalid key, got %v", err)
	}
}

func TestValidateAPIKey_InactiveUser(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user and key
	email := "test@example.com"
	user, _ := db.CreateUser(ctx, "testuser", &email, nil)
	keyWithPlaintext, _ := db.CreateAPIKey(ctx, user.ID, "Test Device")

	// Deactivate user
	_, err := db.ExecContext(ctx, "UPDATE users SET active = 0 WHERE id = ?", user.ID)
	if err != nil {
		t.Fatalf("failed to deactivate user: %v", err)
	}

	// Try to validate key
	_, err = db.ValidateAPIKey(ctx, keyWithPlaintext.PlaintextKey)
	if !IsNotFound(err) {
		t.Error("inactive user's key should not validate")
	}
}

func TestListUserAPIKeys(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user
	email := "test@example.com"
	user, _ := db.CreateUser(ctx, "testuser", &email, nil)

	// Create multiple keys
	db.CreateAPIKey(ctx, user.ID, "Device 1")
	db.CreateAPIKey(ctx, user.ID, "Device 2")
	db.CreateAPIKey(ctx, user.ID, "Device 3")

	// List keys
	keys, err := db.ListUserAPIKeys(ctx, user.ID)
	if err != nil {
		t.Fatalf("list keys failed: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("got %d keys, want 3", len(keys))
	}
}

func TestRevokeAPIKey_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user and key
	email := "test@example.com"
	user, _ := db.CreateUser(ctx, "testuser", &email, nil)
	keyWithPlaintext, _ := db.CreateAPIKey(ctx, user.ID, "Test Device")

	// Revoke the key
	err := db.RevokeAPIKey(ctx, keyWithPlaintext.ID, user.ID)
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	// Try to validate revoked key
	_, err = db.ValidateAPIKey(ctx, keyWithPlaintext.PlaintextKey)
	if !IsNotFound(err) {
		t.Error("revoked key should not validate")
	}

	// Verify key is marked inactive
	keys, _ := db.ListUserAPIKeys(ctx, user.ID)
	if len(keys) != 1 {
		t.Fatal("should still have 1 key (revoked)")
	}
	if keys[0].Active {
		t.Error("key should be inactive after revoke")
	}
	if keys[0].RevokedAt == nil {
		t.Error("key should have RevokedAt timestamp")
	}
}

// =============================================================================
// PROGRESS TRACKING TESTS
// =============================================================================

func TestCreateProgress_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user and reading
	email := "test@example.com"
	db.CreateUser(ctx, "testuser", &email, nil)
	userID := "1" // String representation

	reading := &DailyReading{
		Date:          "2025-01-01",
		MorningPsalms: []string{"1"},
		EveningPsalms: []string{"2"},
		FirstReading:  "Genesis 1:1",
		SecondReading: "Romans 1:1",
		GospelReading: "John 1:1",
		SourceURL:     "https://example.com",
	}
	db.UpsertDailyReading(ctx, reading)

	// Mark complete
	notes := "Great reading!"
	progress := &ReadingProgress{
		UserID:      userID,
		ReadingDate: "2025-01-01",
		Notes:       &notes,
		CompletedAt: time.Now(),
	}

	err := db.CreateProgress(ctx, progress)
	if err != nil {
		t.Fatalf("create progress failed: %v", err)
	}

	// Verify completion
	retrieved, err := db.GetProgressByDate(ctx, userID, "2025-01-01")
	if err != nil {
		t.Fatalf("get progress failed: %v", err)
	}

	if retrieved.ReadingDate != "2025-01-01" {
		t.Errorf("date = %q, want 2025-01-01", retrieved.ReadingDate)
	}
	if retrieved.Notes == nil || *retrieved.Notes != notes {
		t.Error("notes not saved correctly")
	}
}

func TestCreateProgress_Duplicate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user and reading
	email := "test@example.com"
	db.CreateUser(ctx, "testuser", &email, nil)
	userID := "1"

	reading := &DailyReading{
		Date:          "2025-01-01",
		MorningPsalms: []string{"1"},
		EveningPsalms: []string{"2"},
		FirstReading:  "Genesis 1:1",
		SecondReading: "Romans 1:1",
		GospelReading: "John 1:1",
		SourceURL:     "https://example.com",
	}
	db.UpsertDailyReading(ctx, reading)

	// Mark complete first time
	progress := &ReadingProgress{
		UserID:      userID,
		ReadingDate: "2025-01-01",
		CompletedAt: time.Now(),
	}
	db.CreateProgress(ctx, progress)

	// Try to mark complete again
	progress2 := &ReadingProgress{
		UserID:      userID,
		ReadingDate: "2025-01-01",
		CompletedAt: time.Now(),
	}
	err := db.CreateProgress(ctx, progress2)

	// Should get duplicate error
	if err != ErrDuplicate {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestDeleteProgress_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user and reading with progress
	email := "test@example.com"
	db.CreateUser(ctx, "testuser", &email, nil)
	userID := "1"

	reading := &DailyReading{
		Date:          "2025-01-01",
		MorningPsalms: []string{"1"},
		EveningPsalms: []string{"2"},
		FirstReading:  "Genesis 1:1",
		SecondReading: "Romans 1:1",
		GospelReading: "John 1:1",
		SourceURL:     "https://example.com",
	}
	db.UpsertDailyReading(ctx, reading)

	progress := &ReadingProgress{
		UserID:      userID,
		ReadingDate: "2025-01-01",
		CompletedAt: time.Now(),
	}
	db.CreateProgress(ctx, progress)

	// Delete progress
	err := db.DeleteProgress(ctx, userID, "2025-01-01")
	if err != nil {
		t.Fatalf("delete progress failed: %v", err)
	}

	// Verify deletion
	_, err = db.GetProgressByDate(ctx, userID, "2025-01-01")
	if !IsNotFound(err) {
		t.Error("progress should be deleted")
	}
}

func TestGetProgressStats_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user with no progress
	email := "test@example.com"
	db.CreateUser(ctx, "testuser", &email, nil)
	userID := "1"

	stats, err := db.GetProgressStats(ctx, userID)
	if err != nil {
		t.Fatalf("get stats failed: %v", err)
	}

	if stats.CompletedDays != 0 {
		t.Errorf("CompletedDays = %d, want 0", stats.CompletedDays)
	}
	if stats.CurrentStreak != 0 {
		t.Errorf("CurrentStreak = %d, want 0", stats.CurrentStreak)
	}
}

// =============================================================================
// SCRAPE LOG TESTS
// =============================================================================

func TestLogScrapeAttempt_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	duration := int64(1234)
	entry := &ScrapeLogEntry{
		Date:       "2025-01-01",
		SourceURL:  "https://example.com",
		Success:    true,
		DurationMs: &duration,
	}

	err := db.LogScrapeAttempt(ctx, entry)
	if err != nil {
		t.Fatalf("log scrape failed: %v", err)
	}

	// Verify it was logged
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM scrape_log WHERE date = ?", "2025-01-01").Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if count != 1 {
		t.Errorf("scrape_log has %d entries, want 1", count)
	}
}

func TestLogScrapeAttempt_Failure(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	errMsg := "Connection timeout"
	entry := &ScrapeLogEntry{
		Date:         "2025-01-01",
		SourceURL:    "https://example.com",
		Success:      false,
		ErrorMessage: &errMsg,
	}

	err := db.LogScrapeAttempt(ctx, entry)
	if err != nil {
		t.Fatalf("log scrape failed: %v", err)
	}

	// Query the logged entry
	var success bool
	var errorMessage sql.NullString
	err = db.QueryRowContext(ctx,
		"SELECT success, error_message FROM scrape_log WHERE date = ?",
		"2025-01-01",
	).Scan(&success, &errorMessage)

	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if success {
		t.Error("success should be false for failed scrape")
	}
	if !errorMessage.Valid || errorMessage.String != errMsg {
		t.Errorf("error message = %q, want %q", errorMessage.String, errMsg)
	}
}

// =============================================================================
// TRANSACTION TESTS
// =============================================================================

func TestWithTx_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Use transaction to create user
	var userID int64
	err := db.WithTx(ctx, func(tx *Tx) error {
		email := "test@example.com"
		user, err := db.CreateUser(ctx, "testuser", &email, nil)
		if err != nil {
			return err
		}
		userID = user.ID
		return nil
	})

	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	// Verify user was created
	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		t.Error("user should exist after successful transaction")
	}
	if user.Username != "testuser" {
		t.Errorf("username = %q, want testuser", user.Username)
	}
}

func TestWithTx_Rollback(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Transaction that returns error
	err := db.WithTx(ctx, func(tx *Tx) error {
		email := "test@example.com"
		_, err := db.CreateUser(ctx, "testuser", &email, nil)
		if err != nil {
			return err
		}

		// Force rollback by returning error
		return sql.ErrTxDone
	})

	if err == nil {
		t.Fatal("expected error from transaction")
	}

	// Verify user was NOT created (transaction rolled back)
	_, err = db.GetUserByUsername(ctx, "testuser")
	if !IsNotFound(err) {
		t.Error("user should not exist after rolled back transaction")
	}
}

// =============================================================================
// ERROR HANDLING TESTS
// =============================================================================

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrNotFound",
			err:  ErrNotFound,
			want: true,
		},
		{
			name: "sql.ErrNoRows",
			err:  sql.ErrNoRows,
			want: true,
		},
		{
			name: "other error",
			err:  sql.ErrTxDone,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// =============================================================================
// CONSTRAINT TESTS
// =============================================================================

func TestDailyReading_UniqueDate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	reading := &DailyReading{
		Date:          "2025-01-01",
		MorningPsalms: []string{"1"},
		EveningPsalms: []string{"2"},
		FirstReading:  "Genesis 1:1",
		SecondReading: "Romans 1:1",
		GospelReading: "John 1:1",
		SourceURL:     "https://example.com",
	}

	// First insert should succeed
	err := db.UpsertDailyReading(ctx, reading)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// Second insert with same date should update (not error)
	reading.FirstReading = "Different reading"
	err = db.UpsertDailyReading(ctx, reading)
	if err != nil {
		t.Fatalf("upsert should handle duplicates: %v", err)
	}
}

func TestReadingProgress_UniqueUserDate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Migrate(ctx)

	// Create user and reading
	email := "test@example.com"
	db.CreateUser(ctx, "testuser", &email, nil)
	userID := "1"

	reading := &DailyReading{
		Date:          "2025-01-01",
		MorningPsalms: []string{"1"},
		EveningPsalms: []string{"2"},
		FirstReading:  "Genesis 1:1",
		SecondReading: "Romans 1:1",
		GospelReading: "John 1:1",
		SourceURL:     "https://example.com",
	}
	db.UpsertDailyReading(ctx, reading)

	// First completion
	progress := &ReadingProgress{
		UserID:      userID,
		ReadingDate: "2025-01-01",
		CompletedAt: time.Now(),
	}
	db.CreateProgress(ctx, progress)

	// Second completion should fail with duplicate error
	progress2 := &ReadingProgress{
		UserID:      userID,
		ReadingDate: "2025-01-01",
		CompletedAt: time.Now(),
	}
	err := db.CreateProgress(ctx, progress2)

	if err != ErrDuplicate {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}
