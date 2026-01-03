// Command import loads scraped lectionary readings into the SQLite database.
//
// Usage:
//
//	go run ./cmd/import -json data/lectionary-scraper/scraped_readings.json -db data/lectionary.db
//
// This tool:
// 1. Creates/opens the SQLite database
// 2. Runs migrations to ensure schema is current
// 3. Parses the scraped JSON file
// 4. Imports all readings using idempotent upserts
//
// The import is idempotent - running it multiple times is safe.
// Existing readings will be updated if data has changed.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/zapponejosh/lectionary-api/internal/database"
)

func main() {
	// Parse command line flags
	jsonPath := flag.String("json", "data/lectionary-scraper/scraped_readings.json", "Path to scraped JSON file")
	dbPath := flag.String("db", "data/lectionary.db", "Path to SQLite database")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	// Setup logger
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Run import
	if err := run(*jsonPath, *dbPath, logger); err != nil {
		logger.Error("import failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("import complete")
}

// =============================================================================
// Scraper JSON Format
// =============================================================================

// ScraperReading represents a single reading from the scraper output.
type ScraperReading struct {
	Morning       string `json:"Morning"`
	FirstReading  string `json:"First Reading"`
	SecondReading string `json:"Second Reading"`
	GospelReading string `json:"Gospel"`
	Evening       string `json:"Evening"`
}

// ScraperDateEntry represents one date's data from the scraper.
type ScraperDateEntry struct {
	Date      string         `json:"date"`
	URL       string         `json:"url"`
	Readings  ScraperReading `json:"readings"`
	ScrapedAt string         `json:"scraped_at"`
}

// ScraperMetadata contains scraper metadata.
type ScraperMetadata struct {
	ExportedAt string `json:"exported_at"`
	TotalDates int    `json:"total_dates"`
	Source     string `json:"source"`
	DateRange  *struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"date_range"`
}

// ScraperData represents the complete scraper output file.
type ScraperData struct {
	Metadata       ScraperMetadata             `json:"metadata"`
	ReadingsByDate map[string]ScraperDateEntry `json:"readings_by_date"`
}

// =============================================================================
// Import Functions
// =============================================================================

func run(jsonPath, dbPath string, logger *slog.Logger) error {
	ctx := context.Background()
	startTime := time.Now()

	// =========================================================================
	// Step 1: Read and parse JSON
	// =========================================================================
	logger.Info("reading JSON file", slog.String("path", jsonPath))

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read JSON file: %w", err)
	}

	var scraperData ScraperData
	if err := json.Unmarshal(data, &scraperData); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}

	logger.Info("parsed JSON",
		slog.Int("dates", len(scraperData.ReadingsByDate)),
		slog.String("source", scraperData.Metadata.Source),
	)

	if scraperData.Metadata.DateRange != nil {
		logger.Info("date range",
			slog.String("start", scraperData.Metadata.DateRange.Start),
			slog.String("end", scraperData.Metadata.DateRange.End),
		)
	}

	// =========================================================================
	// Step 2: Open database and run migrations
	// =========================================================================
	logger.Info("opening database", slog.String("path", dbPath))

	db, err := database.Open(database.DefaultConfig(dbPath), logger)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	migrated, err := db.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	logger.Info("migrations complete", slog.Int("applied", migrated))

	// =========================================================================
	// Step 3: Import readings
	// =========================================================================
	logger.Info("starting import")

	stats := &ImportStats{}

	for date, entry := range scraperData.ReadingsByDate {
		if err := importReading(ctx, db, entry, logger, stats); err != nil {
			logger.Warn("failed to import reading",
				slog.String("date", date),
				slog.String("error", err.Error()),
			)
			stats.Failed++
		}
	}

	// =========================================================================
	// Step 4: Get final statistics
	// =========================================================================
	dbStats, err := db.GetReadingStats(ctx)
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	elapsed := time.Since(startTime)

	logger.Info("import verified",
		slog.Int("total_days", dbStats.TotalDays),
		slog.String("earliest_date", dbStats.EarliestDate),
		slog.String("latest_date", dbStats.LatestDate),
		slog.Duration("elapsed", elapsed),
	)

	// Print summary
	fmt.Println()
	fmt.Println("=== Import Summary ===")
	fmt.Printf("Imported:          %d readings\n", stats.Imported)
	fmt.Printf("Updated:           %d readings\n", stats.Updated)
	fmt.Printf("Failed:            %d readings\n", stats.Failed)
	fmt.Printf("Total in database: %d readings\n", dbStats.TotalDays)
	fmt.Printf("Date range:        %s to %s\n", dbStats.EarliestDate, dbStats.LatestDate)
	fmt.Printf("Time elapsed:      %v\n", elapsed.Round(time.Millisecond))
	fmt.Println()

	return nil
}

// ImportStats tracks import statistics.
type ImportStats struct {
	Imported int
	Updated  int
	Failed   int
}

// parsePsalms converts "Psalm 111; 149" to []string{"111", "149"}
func parsePsalms(raw string) []string {
	if raw == "" {
		return []string{}
	}

	// Remove "Psalm" or "Psalms" prefix if present
	raw = trimPrefix(raw, "Psalm ")
	raw = trimPrefix(raw, "Psalms ")
	raw = trimPrefix(raw, "Ps. ")
	raw = trimPrefix(raw, "Pss. ")

	// Split on semicolon
	parts := splitAndTrim(raw, ";")

	return parts
}

// trimPrefix removes prefix from string (case-insensitive)
func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// splitAndTrim splits string on separator and trims whitespace
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	var result []string
	for _, part := range split(s, sep) {
		trimmed := trim(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// split is a simple string split
func split(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	var result []string
	current := ""

	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, current)
			current = ""
			i += len(sep) - 1
		} else {
			current += string(s[i])
		}
	}
	result = append(result, current)

	return result
}

// trim removes leading/trailing whitespace
func trim(s string) string {
	start := 0
	end := len(s)

	// Trim leading
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

// importReading imports a single date's reading into the database.
func importReading(ctx context.Context, db *database.DB, entry ScraperDateEntry, logger *slog.Logger, stats *ImportStats) error {
	// Parse scraped_at timestamp
	// Python's datetime.isoformat() outputs: "2026-01-03T12:04:24.723240"
	var scrapedAt time.Time
	var err error

	// Try parsing with microseconds (Python's isoformat)
	scrapedAt, err = time.Parse("2006-01-02T15:04:05.999999", entry.ScrapedAt)
	if err != nil {
		// Try RFC3339 format
		scrapedAt, err = time.Parse(time.RFC3339, entry.ScrapedAt)
		if err != nil {
			logger.Debug("could not parse scraped_at timestamp",
				slog.String("date", entry.Date),
				slog.String("scraped_at", entry.ScrapedAt),
				slog.String("error", err.Error()),
			)
			scrapedAt = time.Now() // fallback to now
		}
	}

	// Create DailyReading struct
	reading := &database.DailyReading{
		Date:          entry.Date,
		MorningPsalms: parsePsalms(entry.Readings.Morning),
		EveningPsalms: parsePsalms(entry.Readings.Evening),
		FirstReading:  entry.Readings.FirstReading,
		SecondReading: entry.Readings.SecondReading,
		GospelReading: entry.Readings.GospelReading,
		SourceURL:     entry.URL,
		ScrapedAt:     &scrapedAt,
	}

	// Check if it already exists (for stats)
	existing, err := db.GetReadingByDate(ctx, entry.Date)
	if err != nil && !database.IsNotFound(err) {
		return fmt.Errorf("check existing reading: %w", err)
	}

	// Upsert (insert or update)
	if err := db.UpsertDailyReading(ctx, reading); err != nil {
		return fmt.Errorf("upsert reading: %w", err)
	}

	if existing != nil {
		stats.Updated++
		logger.Debug("updated reading", slog.String("date", entry.Date))
	} else {
		stats.Imported++
		logger.Debug("imported reading", slog.String("date", entry.Date))
	}

	return nil
}
