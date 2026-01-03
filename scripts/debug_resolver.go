package main

import (
	"fmt"
	"time"
)

func main() {
	// Test the failing dates
	testDates := []string{
		"2024-11-24", // Christ the King Sunday - should work
		"2024-11-25", // Monday after - FAILING
		"2024-11-30", // Saturday before Advent - FAILING
		"2027-11-21", // Christ the King Sunday 2027
		"2027-11-22", // Monday after - FAILING
	}

	for _, dateStr := range testDates {
		date, _ := time.Parse("2006-01-02", dateStr)
		year := date.Year()

		pentecost := calculatePentecost(year)
		advent := calculateAdvent(year)

		fmt.Printf("\n=== %s (%s) ===\n", dateStr, date.Weekday())
		fmt.Printf("Year: %d\n", year)
		fmt.Printf("Pentecost: %s\n", pentecost.Format("2006-01-02"))
		fmt.Printf("Advent: %s\n", advent.Format("2006-01-02"))

		// Check conditions
		fmt.Printf("date.Before(pentecost): %v\n", date.Before(pentecost))
		fmt.Printf("date.Before(advent): %v\n", date.Before(advent))
		fmt.Printf("!date.Before(advent): %v\n", !date.Before(advent))

		// If in Pentecost range
		if !date.Before(pentecost) && date.Before(advent) {
			fmt.Println("✓ In Pentecost range")

			daysSincePentecost := int(date.Sub(pentecost).Hours() / 24)
			fmt.Printf("daysSincePentecost: %d\n", daysSincePentecost)

			// Christ the King check
			christTheKing := advent.AddDate(0, 0, -7)
			fmt.Printf("Christ the King Sunday: %s\n", christTheKing.Format("2006-01-02"))

			if date.Weekday() == time.Sunday && isSameDay(date, christTheKing) {
				fmt.Println("→ Resolves to: Christ the King / Sunday")
			} else if daysSincePentecost >= 8 {
				daysAfterTrinity := daysSincePentecost - 7
				weekNum := (daysAfterTrinity / 7) + 2

				fmt.Printf("daysAfterTrinity: %d\n", daysAfterTrinity)
				fmt.Printf("weekNum (before cap): %d\n", weekNum)

				if weekNum > 27 {
					weekNum = 27
				}
				fmt.Printf("weekNum (after cap): %d\n", weekNum)
				fmt.Printf("→ Resolves to: Week %d after Pentecost / %s\n", weekNum, date.Weekday())
			}
		} else {
			fmt.Println("✗ NOT in Pentecost range - this is the problem!")
		}
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

func calculatePentecost(year int) time.Time {
	return calculateEaster(year).AddDate(0, 0, 49)
}

func calculateAdvent(year int) time.Time {
	christmas := time.Date(year, time.December, 25, 0, 0, 0, 0, time.UTC)
	daysToSubtract := int(christmas.Weekday())
	if daysToSubtract == 0 {
		daysToSubtract = 7
	}
	return christmas.AddDate(0, 0, -daysToSubtract-21)
}

func isSameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
