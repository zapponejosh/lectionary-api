package calendar

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Liturgical week limits
const (
	// MaxWeeksAfterBaptism is the maximum number of weeks between
	// Baptism of the Lord and Ash Wednesday (typically 4-9 weeks).
	// Note: Database only has weeks 1-4; weeks 5+ use dated weeks.
	MaxWeeksAfterBaptism = 4

	// MaxWeeksAfterPentecost is the maximum number of weeks between
	// Pentecost and Advent (typically 22-27 weeks).
	// Note: Database has weeks 2-27; weeks beyond 27 fall back to 27.
	MaxWeeksAfterPentecost = 27

	// AdventWeeks is the number of weeks in Advent.
	AdventWeeks = 4

	// LentWeeks is the number of weeks in Lent (not counting Holy Week).
	LentWeeks = 5 // Changed from 6 - week 6 is Holy Week

	// EasterWeeks is the number of weeks in the Easter season.
	EasterWeeks = 7
)

// DayName returns the day of week name (Sunday, Monday, etc.)
func DayName(date time.Time) string {
	return date.Weekday().String()
}

// Ordinal returns the ordinal form of a number (1st, 2nd, 3rd, 4th, etc.)
func Ordinal(n int) string {
	suffix := "th"
	switch n % 10 {
	case 1:
		if n%100 != 11 {
			suffix = "st"
		}
	case 2:
		if n%100 != 12 {
			suffix = "nd"
		}
	case 3:
		if n%100 != 13 {
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

// FindSundayBetween finds the Sunday within a date range (inclusive).
// Returns nil if no Sunday exists in the range.
func FindSundayBetween(year int, startMonth, startDay, endMonth, endDay int) *time.Time {
	start := time.Date(year, time.Month(startMonth), startDay, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, time.Month(endMonth), endDay, 0, 0, 0, 0, time.UTC)

	current := start
	for !current.After(end) {
		if current.Weekday() == time.Sunday {
			result := current // Create a copy to return pointer to
			return &result
		}
		current = current.AddDate(0, 0, 1)
	}

	return nil
}

// DatedWeekPeriodPattern matches strings like "Week following Sun. between Feb. 11 and 17"
var DatedWeekPeriodPattern = regexp.MustCompile(`Week following Sun\. between (\w+)\. (\d+) and (\d+)`)

// monthAbbreviations maps month abbreviations and full names to month numbers.
var monthAbbreviations = map[string]int{
	"Jan": 1, "January": 1,
	"Feb": 2, "February": 2,
	"Mar": 3, "March": 3,
	"Apr": 4, "April": 4,
	"May": 5,
	"Jun": 6, "June": 6,
	"Jul": 7, "July": 7,
	"Aug": 8, "August": 8,
	"Sep": 9, "September": 9,
	"Oct": 10, "October": 10,
	"Nov": 11, "November": 11,
	"Dec": 12, "December": 12,
}

// ParseDatedWeekPeriod parses a dated week period string like
// "Week following Sun. between Feb. 11 and 17" and extracts the month/day range.
//
// Returns: startMonth, startDay, endMonth, endDay, error
func ParseDatedWeekPeriod(period string) (int, int, int, int, error) {
	matches := DatedWeekPeriodPattern.FindStringSubmatch(period)

	if len(matches) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("invalid dated week period format: %s", period)
	}

	monthName := matches[1]
	startDay, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid start day: %w", err)
	}

	endDay, err := strconv.Atoi(matches[3])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid end day: %w", err)
	}

	month, ok := monthAbbreviations[monthName]
	if !ok {
		return 0, 0, 0, 0, fmt.Errorf("unknown month: %s", monthName)
	}

	return month, startDay, month, endDay, nil
}

// GetLiturgicalWeekNumber calculates which week of a liturgical season a date falls in.
// Week numbering starts at 1.
//
// For Advent: weeks 1-4
// For Lent: weeks 1-6 (first Sunday of Lent starts week 1)
// For Easter: weeks 1-7 (Easter Sunday starts week 1)
func GetLiturgicalWeekNumber(date time.Time, seasonStart time.Time) int {
	daysDiff := int(date.Sub(seasonStart).Hours() / 24)
	weekNum := (daysDiff / 7) + 1
	return weekNum
}

// DaysBetween calculates the number of days between two dates.
// Returns a positive number if end is after start.
func DaysBetween(start, end time.Time) int {
	return int(end.Sub(start).Hours() / 24)
}

// IsSameDay returns true if two times represent the same calendar day.
func IsSameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// NormalizeToMidnight returns the date at midnight UTC.
// This is useful for consistent date comparisons.
func NormalizeToMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
