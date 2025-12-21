package calendar

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// DayName returns the day of week name (Sunday, Monday, etc.)
func DayName(date time.Time) string {
	days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	return days[date.Weekday()]
}

// Ordinal returns the ordinal form of a number (1st, 2nd, 3rd, 4th, etc.)
func Ordinal(n int) string {
	if n == 1 {
		return "1st"
	}
	if n == 2 {
		return "2nd"
	}
	if n == 3 {
		return "3rd"
	}
	return fmt.Sprintf("%dth", n)
}

// FindSundayBetween finds the Sunday within a date range.
// Returns nil if no Sunday exists in the range.
func FindSundayBetween(year int, startMonth, startDay, endMonth, endDay int) *time.Time {
	start := time.Date(year, time.Month(startMonth), startDay, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, time.Month(endMonth), endDay, 0, 0, 0, 0, time.UTC)

	current := start
	for current.Before(end) || current.Equal(end) {
		if current.Weekday() == time.Sunday {
			return &current
		}
		current = current.AddDate(0, 0, 1)
	}

	return nil
}

// ParseDatedWeekPeriod parses a dated week period string like
// "Week following Sun. between Feb. 11 and 17" and extracts the month/day range.
//
// Returns: startMonth, startDay, endMonth, endDay, error
func ParseDatedWeekPeriod(period string) (int, int, int, int, error) {
	// Pattern: "Week following Sun. between Feb. 11 and 17"
	// or "Week following Sun. between Feb. 25 and 29"
	pattern := regexp.MustCompile(`Week following Sun\. between (\w+)\. (\d+) and (\d+)`)
	matches := pattern.FindStringSubmatch(period)

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

	monthMap := map[string]int{
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

	month, ok := monthMap[monthName]
	if !ok {
		return 0, 0, 0, 0, fmt.Errorf("unknown month: %s", monthName)
	}

	return month, startDay, month, endDay, nil
}

// GetLiturgicalWeekNumber calculates which week of a liturgical season a date falls in.
// For Advent: weeks 1-4
// For Lent: weeks 1-6 (Ash Wednesday starts Lent)
// For Easter: weeks 1-7 (Easter Sunday starts week 1)
func GetLiturgicalWeekNumber(date time.Time, seasonStart time.Time) int {
	daysDiff := int(date.Sub(seasonStart).Hours() / 24)
	weekNum := (daysDiff / 7) + 1
	return weekNum
}
