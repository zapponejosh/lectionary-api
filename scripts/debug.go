package main

import (
	"fmt"
	"time"
)

// This script shows what the resolver WOULD generate for failing dates
// Run this to compare against your database

func main() {
	failingDates := []string{
		"2025-03-05", // Ash Wednesday
		"2025-03-09", // 1st Sunday of Lent
		"2025-04-13", // Palm Sunday
		"2025-04-17", // Holy Thursday
		"2025-04-18", // Good Friday
		"2025-04-20", // Easter Sunday
		"2025-06-08", // Pentecost
		"2025-06-15", // Trinity Sunday
		"2024-02-29", // Leap year (Lent 2024)
		"2030-06-15", // Future date
	}

	// Key dates for 2025
	easter2025 := calculateEaster(2025)
	ashWed2025 := easter2025.AddDate(0, 0, -46)
	pentecost2025 := easter2025.AddDate(0, 0, 49)

	fmt.Println("=== 2025 Key Dates ===")
	fmt.Printf("Ash Wednesday: %s\n", ashWed2025.Format("2006-01-02"))
	fmt.Printf("Easter:        %s\n", easter2025.Format("2006-01-02"))
	fmt.Printf("Pentecost:     %s\n", pentecost2025.Format("2006-01-02"))
	fmt.Println()

	// Key dates for 2024 (for leap year test)
	easter2024 := calculateEaster(2024)
	ashWed2024 := easter2024.AddDate(0, 0, -46)

	fmt.Println("=== 2024 Key Dates ===")
	fmt.Printf("Ash Wednesday: %s\n", ashWed2024.Format("2006-01-02"))
	fmt.Printf("Easter:        %s\n", easter2024.Format("2006-01-02"))
	fmt.Println()

	fmt.Println("=== Failing Dates Analysis ===")
	fmt.Println()

	for _, dateStr := range failingDates {
		date, _ := time.Parse("2006-01-02", dateStr)
		period, dayID := resolveDate(date)

		fmt.Printf("Date: %s (%s)\n", dateStr, date.Weekday())
		fmt.Printf("  Resolver generates: period=%q, day_identifier=%q\n", period, dayID)
		fmt.Printf("  SQL to check: SELECT * FROM lectionary_days WHERE period='%s' AND day_identifier='%s';\n", period, dayID)
		fmt.Println()
	}
}

func calculateEaster(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func resolveDate(date time.Time) (period, dayIdentifier string) {
	year := date.Year()
	easter := calculateEaster(year)
	ashWednesday := easter.AddDate(0, 0, -46)
	pentecost := easter.AddDate(0, 0, 49)
	palmSunday := easter.AddDate(0, 0, -7)

	// Ash Wednesday and Following (Wed-Sat)
	daysSinceAsh := int(date.Sub(ashWednesday).Hours() / 24)
	if daysSinceAsh >= 0 && daysSinceAsh <= 3 {
		dayNames := []string{"Wednesday", "Thursday", "Friday", "Saturday"}
		return "Ash Wednesday and Following", dayNames[daysSinceAsh]
	}

	// Lent weeks 1-5
	firstSundayOfLent := ashWednesday
	for firstSundayOfLent.Weekday() != time.Sunday {
		firstSundayOfLent = firstSundayOfLent.AddDate(0, 0, 1)
	}

	if !date.Before(firstSundayOfLent) && date.Before(palmSunday) {
		daysSinceFirstSunday := int(date.Sub(firstSundayOfLent).Hours() / 24)
		weekNum := (daysSinceFirstSunday / 7) + 1
		if weekNum >= 1 && weekNum <= 5 {
			return fmt.Sprintf("%s Week of Lent", ordinal(weekNum)), date.Weekday().String()
		}
	}

	// Holy Week
	daysSincePalm := int(date.Sub(palmSunday).Hours() / 24)
	if daysSincePalm >= 0 && daysSincePalm < 7 {
		return "Holy Week", date.Weekday().String()
	}

	// Easter weeks
	if !date.Before(easter) && date.Before(pentecost) {
		daysSinceEaster := int(date.Sub(easter).Hours() / 24)
		weekNum := (daysSinceEaster / 7) + 1
		if weekNum == 1 {
			return "Easter Week", date.Weekday().String()
		}
		return fmt.Sprintf("%s Week of Easter", ordinal(weekNum)), date.Weekday().String()
	}

	// Pentecost Sunday
	if date.Year() == pentecost.Year() && date.Month() == pentecost.Month() && date.Day() == pentecost.Day() {
		return "Pentecost", "Sunday"
	}

	// Trinity Sunday (Sunday after Pentecost)
	trinitySunday := pentecost.AddDate(0, 0, 7)
	if date.Year() == trinitySunday.Year() && date.Month() == trinitySunday.Month() && date.Day() == trinitySunday.Day() {
		return "Trinity Sunday and Following", "Sunday"
	}

	// Week 1 after Pentecost (Mon-Sat after Pentecost)
	daysSincePentecost := int(date.Sub(pentecost).Hours() / 24)
	if daysSincePentecost >= 1 && daysSincePentecost <= 6 {
		return "Week 1 after Pentecost", date.Weekday().String()
	}

	// Trinity Sunday and Following (Mon-Sat after Trinity)
	if daysSincePentecost >= 8 && daysSincePentecost <= 13 {
		return "Trinity Sunday and Following", date.Weekday().String()
	}

	// Week 2+ after Pentecost
	if daysSincePentecost >= 14 {
		weekNum := ((daysSincePentecost - 14) / 7) + 2
		return fmt.Sprintf("Week %d after Pentecost", weekNum), date.Weekday().String()
	}

	return "UNKNOWN", "UNKNOWN"
}

func ordinal(n int) string {
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
