// Command import loads the merged lectionary JSON into the SQLite database.
//
// Usage:
//
//	go run ./cmd/import -json data/merge-data/daily_lectionary_merged.json -db data/lectionary.db
//
// This tool:
// 1. Creates/opens the SQLite database
// 2. Runs migrations to ensure schema is current
// 3. Parses the merged JSON file
// 4. Imports all positions and readings in a single transaction
//
// The import is idempotent - running it twice will fail on duplicate positions.
// To reimport, delete the database file first.
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
	jsonPath := flag.String("json", "data/merge-data/daily_lectionary_merged.json", "Path to merged JSON file")
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

	var importData database.ImportData
	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}

	logger.Info("parsed JSON",
		slog.Int("positions", len(importData.DailyLectionary)),
		slog.String("source", importData.Metadata.Source),
		slog.String("generated_at", importData.Metadata.GeneratedAt),
	)

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
	// Step 3: Import data in a transaction
	// =========================================================================
	logger.Info("starting import")

	var stats ImportStats
	err = db.WithTx(ctx, func(tx *database.Tx) error {
		return importPositions(ctx, tx, importData.DailyLectionary, logger, &stats)
	})
	if err != nil {
		return fmt.Errorf("import data: %w", err)
	}

	// =========================================================================
	// Step 4: Verify import
	// =========================================================================
	dayCount, err := db.CountDays(ctx)
	if err != nil {
		return fmt.Errorf("count days: %w", err)
	}

	readingCount, err := db.CountReadings(ctx)
	if err != nil {
		return fmt.Errorf("count readings: %w", err)
	}

	year1Count, year2Count, err := db.CountReadingsByYear(ctx)
	if err != nil {
		return fmt.Errorf("count readings by year: %w", err)
	}

	elapsed := time.Since(startTime)

	logger.Info("import verified",
		slog.Int("days", dayCount),
		slog.Int("total_readings", readingCount),
		slog.Int("year_1_readings", year1Count),
		slog.Int("year_2_readings", year2Count),
		slog.Duration("elapsed", elapsed),
	)

	// Print summary
	fmt.Println()
	fmt.Println("=== Import Summary ===")
	fmt.Printf("Positions imported:  %d\n", stats.Positions)
	fmt.Printf("Year 1 readings:     %d\n", stats.Year1Readings)
	fmt.Printf("Year 2 readings:     %d\n", stats.Year2Readings)
	fmt.Printf("Positions w/o Y2:    %d\n", stats.MissingYear2)
	fmt.Printf("Time elapsed:        %v\n", elapsed.Round(time.Millisecond))

	return nil
}

// ImportStats tracks import statistics.
type ImportStats struct {
	Positions     int
	Year1Readings int
	Year2Readings int
	MissingYear2  int
}

// importPositions imports all positions and their readings.
func importPositions(ctx context.Context, tx *database.Tx, positions []database.ImportPosition, logger *slog.Logger, stats *ImportStats) error {
	for i, pos := range positions {
		// Create the lectionary day (position)
		day := &database.LectionaryDay{
			Period:        pos.Period,
			DayIdentifier: pos.DayIdentifier,
			PeriodType:    database.PeriodType(pos.PeriodType),
			SpecialName:   pos.SpecialName,
			MorningPsalms: pos.Psalms.Morning,
			EveningPsalms: pos.Psalms.Evening,
		}

		if err := tx.CreateDay(ctx, day); err != nil {
			return fmt.Errorf("create day %d (%s/%s): %w",
				i+1, pos.Period, pos.DayIdentifier, err)
		}

		stats.Positions++

		// Import Year 1 readings
		if pos.Year1 != nil {
			for _, r := range pos.Year1.Readings {
				// Handle multiple references per reading
				for refIdx, ref := range r.References {
					reading := &database.Reading{
						LectionaryDayID: day.ID,
						YearCycle:       1,
						ReadingType:     database.ReadingType(r.Label),
						Position:        r.Position*10 + refIdx, // Preserve order with multi-ref
						Reference:       ref,
					}

					if err := tx.CreateReading(ctx, reading); err != nil {
						return fmt.Errorf("create year 1 reading for day %d: %w", i+1, err)
					}

					stats.Year1Readings++
				}
			}
		}

		// Import Year 2 readings
		if pos.Year2 != nil {
			for _, r := range pos.Year2.Readings {
				for refIdx, ref := range r.References {
					reading := &database.Reading{
						LectionaryDayID: day.ID,
						YearCycle:       2,
						ReadingType:     database.ReadingType(r.Label),
						Position:        r.Position*10 + refIdx,
						Reference:       ref,
					}

					if err := tx.CreateReading(ctx, reading); err != nil {
						return fmt.Errorf("create year 2 reading for day %d: %w", i+1, err)
					}

					stats.Year2Readings++
				}
			}
		} else {
			stats.MissingYear2++
		}

		// Progress logging every 50 positions
		if (i+1)%50 == 0 {
			logger.Debug("import progress",
				slog.Int("position", i+1),
				slog.Int("total", len(positions)),
			)
		}
	}

	return nil
}
