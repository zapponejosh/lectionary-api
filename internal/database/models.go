// Package database provides database access for the lectionary API.
package database

import (
	"encoding/json"
	"time"
)

// =============================================================================
// Core Domain Types
// =============================================================================

// PeriodType categorizes how a lectionary period relates to the calendar.
// This affects how we compute dates for a given position.
type PeriodType string

const (
	// PeriodTypeLiturgical represents weeks relative to moveable feasts.
	// Examples: "1st Week of Advent", "Holy Week", "3rd Week of Easter"
	// Date computation requires knowing Easter date for the year.
	PeriodTypeLiturgical PeriodType = "liturgical_week"

	// PeriodTypeDated represents weeks anchored to calendar date ranges.
	// Examples: "Week following Sun. between Feb. 11 and 17"
	// These occur in the Epiphany→Lent and Pentecost→Advent transitions.
	PeriodTypeDated PeriodType = "dated_week"

	// PeriodTypeFixed represents specific calendar dates.
	// Examples: Christmas Day (Dec 25), Epiphany (Jan 6)
	// These always fall on the same date regardless of Easter.
	PeriodTypeFixed PeriodType = "fixed_days"
)

// ValidPeriodTypes returns all valid period types.
func ValidPeriodTypes() []PeriodType {
	return []PeriodType{
		PeriodTypeLiturgical,
		PeriodTypeDated,
		PeriodTypeFixed,
	}
}

// IsValid checks if a period type is valid.
func (pt PeriodType) IsValid() bool {
	for _, valid := range ValidPeriodTypes() {
		if pt == valid {
			return true
		}
	}
	return false
}

// ReadingType defines the category of a scripture reading.
type ReadingType string

const (
	// ReadingTypeFirst is the first scripture reading (typically OT).
	ReadingTypeFirst ReadingType = "first"

	// ReadingTypeSecond is the second scripture reading (typically Epistle).
	ReadingTypeSecond ReadingType = "second"

	// ReadingTypeGospel is the gospel reading.
	ReadingTypeGospel ReadingType = "gospel"
)

// ValidReadingTypes returns all valid reading types.
func ValidReadingTypes() []ReadingType {
	return []ReadingType{
		ReadingTypeFirst,
		ReadingTypeSecond,
		ReadingTypeGospel,
	}
}

// IsValid checks if a reading type is valid.
func (rt ReadingType) IsValid() bool {
	for _, valid := range ValidReadingTypes() {
		if rt == valid {
			return true
		}
	}
	return false
}

// =============================================================================
// Database Models
// =============================================================================

// LectionaryDay represents a position in the 2-year lectionary cycle.
// This is NOT tied to specific calendar dates - dates are computed at runtime.
//
// The combination of Period + DayIdentifier uniquely identifies a position.
// Examples:
//   - Period="1st Week of Advent", DayIdentifier="Sunday"
//   - Period="Christmas Season", DayIdentifier="December 25"
type LectionaryDay struct {
	ID            int64      `json:"id"`
	Period        string     `json:"period"`         // "1st Week of Advent", "Holy Week", etc.
	DayIdentifier string     `json:"day_identifier"` // "Sunday", "Monday", or "December 25"
	PeriodType    PeriodType `json:"period_type"`    // liturgical_week, dated_week, fixed_days
	SpecialName   *string    `json:"special_name"`   // "Christmas Day", "Epiphany", etc. (nullable)
	MorningPsalms []string   `json:"morning_psalms"` // ["24", "150"]
	EveningPsalms []string   `json:"evening_psalms"` // ["25", "110"]
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// MorningPsalmsJSON returns the morning psalms as a JSON string for database storage.
func (ld *LectionaryDay) MorningPsalmsJSON() string {
	if ld.MorningPsalms == nil {
		return "[]"
	}
	b, _ := json.Marshal(ld.MorningPsalms)
	return string(b)
}

// EveningPsalmsJSON returns the evening psalms as a JSON string for database storage.
func (ld *LectionaryDay) EveningPsalmsJSON() string {
	if ld.EveningPsalms == nil {
		return "[]"
	}
	b, _ := json.Marshal(ld.EveningPsalms)
	return string(b)
}

// ParsePsalmsJSON parses a JSON array string into a slice of psalm references.
func ParsePsalmsJSON(jsonStr string) ([]string, error) {
	if jsonStr == "" || jsonStr == "[]" {
		return []string{}, nil
	}
	var psalms []string
	err := json.Unmarshal([]byte(jsonStr), &psalms)
	return psalms, err
}

// Reading represents a single scripture reading assigned to a lectionary position.
// Readings are specific to a year cycle (1 or 2).
type Reading struct {
	ID              int64       `json:"id"`
	LectionaryDayID int64       `json:"lectionary_day_id"`
	YearCycle       int         `json:"year_cycle"`   // 1 or 2
	ReadingType     ReadingType `json:"reading_type"` // first, second, gospel
	Position        int         `json:"position"`     // Order within type (usually 1)
	Reference       string      `json:"reference"`    // "Isaiah 1:1-9"
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// ReadingProgress tracks a user's completion of a specific reading.
type ReadingProgress struct {
	ID          int64     `json:"id"`
	UserID      string    `json:"user_id"`
	ReadingID   int64     `json:"reading_id"`
	Notes       *string   `json:"notes"` // nullable
	CompletedAt time.Time `json:"completed_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// =============================================================================
// API Response Types
// =============================================================================

// DailyReadings is the primary response for "give me readings for a date".
// It combines the position information with the appropriate year's readings.
type DailyReadings struct {
	// Position information (from lectionary_days)
	Period        string   `json:"period"`
	DayIdentifier string   `json:"day_identifier"`
	SpecialName   *string  `json:"special_name,omitempty"`
	MorningPsalms []string `json:"morning_psalms"`
	EveningPsalms []string `json:"evening_psalms"`

	// Which year cycle was used for this response
	YearCycle int `json:"year_cycle"`

	// The scripture readings for this year cycle
	Readings []Reading `json:"readings"`
}

// ReadingWithProgress combines a reading with its completion status.
type ReadingWithProgress struct {
	Reading   Reading          `json:"reading"`
	Progress  *ReadingProgress `json:"progress,omitempty"`
	Completed bool             `json:"completed"`
}

// DailyReadingsWithProgress extends DailyReadings with progress information.
type DailyReadingsWithProgress struct {
	DailyReadings
	ReadingsWithProgress []ReadingWithProgress `json:"readings_with_progress"`
}

// ProgressStats contains statistics about a user's reading progress.
type ProgressStats struct {
	TotalReadings     int     `json:"total_readings"`
	CompletedReadings int     `json:"completed_readings"`
	CompletionPercent float64 `json:"completion_percent"`
	CurrentStreak     int     `json:"current_streak"`
	LongestStreak     int     `json:"longest_streak"`
}

// =============================================================================
// Import Types (for loading JSON data)
// =============================================================================

// ImportReading represents a reading in the import JSON format.
type ImportReading struct {
	Position   int      `json:"position"`
	Label      string   `json:"label"` // "first", "second", "gospel"
	References []string `json:"references"`
}

// ImportYearData represents readings for a single year in the import JSON.
type ImportYearData struct {
	Readings []ImportReading `json:"readings"`
}

// ImportPsalms represents the psalms structure in import JSON.
type ImportPsalms struct {
	Morning []string `json:"morning"`
	Evening []string `json:"evening"`
}

// ImportPosition represents a single position in the import JSON.
type ImportPosition struct {
	Period        string          `json:"period"`
	DayIdentifier string          `json:"day_identifier"`
	SpecialName   *string         `json:"special_name"`
	PeriodType    string          `json:"period_type"`
	Psalms        ImportPsalms    `json:"psalms"`
	Year1         *ImportYearData `json:"year_1"`
	Year2         *ImportYearData `json:"year_2"`
}

// ImportData represents the full import JSON structure.
type ImportData struct {
	Metadata struct {
		GeneratedAt    string `json:"generated_at"`
		Source         string `json:"source"`
		SchemaVersion  string `json:"schema_version"`
		TotalPositions int    `json:"total_positions"`
		Year1Complete  int    `json:"year_1_complete"`
		Year2Complete  int    `json:"year_2_complete"`
	} `json:"metadata"`
	DailyLectionary []ImportPosition `json:"daily_lectionary"`
}
