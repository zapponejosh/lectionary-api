// Package database provides database access for the lectionary API.
package database

import (
	"time"
)

// LiturgicalDay represents a single day in the lectionary calendar.
// Each day belongs to a liturgical season and may have special observances.
type LiturgicalDay struct {
	ID             int64     `json:"id"`
	Date           string    `json:"date"`             // ISO 8601 format: YYYY-MM-DD
	Weekday        int       `json:"weekday"`          // 0=Sunday through 6=Saturday
	YearCycle      int       `json:"year_cycle"`       // 1 or 2 (two-year cycle)
	Season         string    `json:"season"`           // advent, christmas, lent, etc.
	SpecialDayName *string   `json:"special_day_name"` // nullable: "Easter", "Ash Wednesday", etc.
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ReadingType defines the category of a scripture reading.
type ReadingType string

const (
	ReadingTypeMorningPsalm ReadingType = "morning_psalm"
	ReadingTypeEveningPsalm ReadingType = "evening_psalm"
	ReadingTypeOldTestament ReadingType = "old_testament"
	ReadingTypeEpistle      ReadingType = "epistle"
	ReadingTypeGospel       ReadingType = "gospel"
)

// ValidReadingTypes returns all valid reading types.
func ValidReadingTypes() []ReadingType {
	return []ReadingType{
		ReadingTypeMorningPsalm,
		ReadingTypeEveningPsalm,
		ReadingTypeOldTestament,
		ReadingTypeEpistle,
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

// Reading represents a single scripture reading assigned to a day.
// Each day has multiple readings (psalms, OT, epistle, gospel).
type Reading struct {
	ID            int64       `json:"id"`
	DayID         int64       `json:"day_id"`
	ReadingType   ReadingType `json:"reading_type"`
	Position      int         `json:"position"`       // Order within type (1, 2, etc.)
	Reference     string      `json:"reference"`      // e.g., "Gen. 17:1–12a"
	IsAlternative bool        `json:"is_alternative"` // True if marked with "or"
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
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

// -----------------------------------------------------------------
// Composite types for API responses
// -----------------------------------------------------------------

// DailyReadings combines a liturgical day with all its readings.
// This is the primary response type for the readings endpoints.
type DailyReadings struct {
	Day      LiturgicalDay `json:"day"`
	Readings []Reading     `json:"readings"`
}

// ReadingWithProgress combines a reading with its completion status.
type ReadingWithProgress struct {
	Reading   Reading          `json:"reading"`
	Progress  *ReadingProgress `json:"progress,omitempty"` // nil if not completed
	Completed bool             `json:"completed"`
}

// DailyReadingsWithProgress combines daily readings with progress info.
type DailyReadingsWithProgress struct {
	Day      LiturgicalDay         `json:"day"`
	Readings []ReadingWithProgress `json:"readings"`
}

// ProgressStats contains statistics about a user's reading progress.
type ProgressStats struct {
	TotalReadings     int     `json:"total_readings"`
	CompletedReadings int     `json:"completed_readings"`
	CompletionPercent float64 `json:"completion_percent"`
	CurrentStreak     int     `json:"current_streak"` // Consecutive days
	LongestStreak     int     `json:"longest_streak"`
}

// -----------------------------------------------------------------
// Season constants and helpers
// -----------------------------------------------------------------

// Season represents a liturgical season.
type Season string

const (
	SeasonAdvent    Season = "advent"
	SeasonChristmas Season = "christmas"
	SeasonEpiphany  Season = "epiphany"
	SeasonLent      Season = "lent"
	SeasonHolyWeek  Season = "holy_week"
	SeasonEaster    Season = "easter"
	SeasonPentecost Season = "pentecost"
	SeasonOrdinary  Season = "ordinary"
)

// ValidSeasons returns all valid liturgical seasons.
func ValidSeasons() []Season {
	return []Season{
		SeasonAdvent,
		SeasonChristmas,
		SeasonEpiphany,
		SeasonLent,
		SeasonHolyWeek,
		SeasonEaster,
		SeasonPentecost,
		SeasonOrdinary,
	}
}

// IsValid checks if a season is valid.
func (s Season) IsValid() bool {
	for _, valid := range ValidSeasons() {
		if s == valid {
			return true
		}
	}
	return false
}
