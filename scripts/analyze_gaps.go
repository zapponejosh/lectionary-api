package main

import (
	"fmt"
	"time"
)

func main() {
	failingDates := []string{
		// 2024 Feb
		"2024-02-04", "2024-02-05", "2024-02-06", "2024-02-07", "2024-02-08",
		"2024-02-09", "2024-02-10", "2024-02-11", "2024-02-12", "2024-02-13",
		// 2024 May/Jun
		"2024-05-27", "2024-05-28", "2024-05-29", "2024-05-30", "2024-05-31",
		"2024-06-01", "2024-06-02",
		// 2024 Nov (Week 27)
		"2024-11-24", "2024-11-25", "2024-11-26", "2024-11-27", "2024-11-28", "2024-11-29", "2024-11-30",
		// 2025 Feb/Mar
		"2025-02-09", "2025-02-10", "2025-02-11", "2025-02-12", "2025-02-13",
		"2025-02-14", "2025-02-15", "2025-02-16", "2025-02-17", "2025-02-18",
		"2025-02-19", "2025-02-20", "2025-02-21", "2025-02-22", "2025-02-23",
		"2025-02-24", "2025-02-25", "2025-02-26", "2025-02-27", "2025-02-28",
		"2025-03-01", "2025-03-02", "2025-03-03", "2025-03-04",
		// 2025 Jun
		"2025-06-16", "2025-06-17", "2025-06-18", "2025-06-19", "2025-06-20", "2025-06-21", "2025-06-22",
	}

	fmt.Println("=== Analysis of Failing Dates ===\n")

	// Group by year
	years := []int{2024, 2025, 2026, 2027}

	for _, year := range years {
		easter := calculateEaster(year)
		ashWed := easter.AddDate(0, 0, -46)
		pentecost := easter.AddDate(0, 0, 49)
		baptism := findSundayBetween(year, 1, 7, 1, 13)
		advent := calculateAdvent(year)

		fmt.Printf("=== %d ===\n", year)
		fmt.Printf("Baptism of Lord: %s\n", baptism.Format("2006-01-02"))
		fmt.Printf("Ash Wednesday:   %s\n", ashWed.Format("2006-01-02"))
		fmt.Printf("Easter:          %s\n", easter.Format("2006-01-02"))
		fmt.Printf("Pentecost:       %s\n", pentecost.Format("2006-01-02"))
		fmt.Printf("Advent:          %s\n", advent.Format("2006-01-02"))
		fmt.Println()

		// Show what weeks exist between Baptism and Ash Wednesday
		fmt.Println("Weeks between Baptism and Ash Wednesday:")
		current := *baptism
		weekNum := 0
		for current.Before(ashWed) {
			if current.Weekday() == time.Sunday && !current.Equal(*baptism) {
				weekNum++
				fmt.Printf("  Week %d after Baptism starts: %s\n", weekNum, current.Format("2006-01-02"))
			}
			current = current.AddDate(0, 0, 1)
		}
		fmt.Printf("  Total weeks needed: %d\n", weekNum)
		fmt.Println()

		// Show weeks between Pentecost and Advent
		fmt.Println("Weeks between Pentecost and Advent:")
		current = pentecost
		weekNum = 0
		for current.Before(advent) {
			if current.Weekday() == time.Sunday {
				weekNum++
				if weekNum == 1 {
					fmt.Printf("  Pentecost Sunday: %s\n", current.Format("2006-01-02"))
				} else if weekNum == 2 {
					fmt.Printf("  Trinity Sunday: %s\n", current.Format("2006-01-02"))
				} else {
					fmt.Printf("  Week %d after Pentecost starts: %s\n", weekNum-1, current.Format("2006-01-02"))
				}
			}
			current = current.AddDate(0, 0, 1)
		}
		fmt.Printf("  Total weeks after Pentecost needed: %d\n", weekNum-1) // -1 because Pentecost itself doesn't count
		fmt.Println()
	}

	fmt.Println("\n=== Failing Dates Detail ===\n")

	for _, dateStr := range failingDates {
		date, _ := time.Parse("2006-01-02", dateStr)
		year := date.Year()

		easter := calculateEaster(year)
		ashWed := easter.AddDate(0, 0, -46)
		pentecost := easter.AddDate(0, 0, 49)
		baptism := findSundayBetween(year, 1, 7, 1, 13)

		period, dayID := resolveDate(date, *baptism, ashWed, easter, pentecost)
		fmt.Printf("%s (%s): %s / %s\n", dateStr, date.Weekday(), period, dayID)
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

func calculateAdvent(year int) time.Time {
	christmas := time.Date(year, time.December, 25, 0, 0, 0, 0, time.UTC)
	daysToSubtract := int(christmas.Weekday())
	if daysToSubtract == 0 {
		daysToSubtract = 7
	}
	return christmas.AddDate(0, 0, -daysToSubtract-21)
}

func findSundayBetween(year, startMonth, startDay, endMonth, endDay int) *time.Time {
	start := time.Date(year, time.Month(startMonth), startDay, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, time.Month(endMonth), endDay, 0, 0, 0, 0, time.UTC)
	current := start
	for !current.After(end) {
		if current.Weekday() == time.Sunday {
			return &current
		}
		current = current.AddDate(0, 0, 1)
	}
	return nil
}

func resolveDate(date time.Time, baptism, ashWed, easter, pentecost time.Time) (string, string) {
	// Weeks after Baptism
	if date.After(baptism) && date.Before(ashWed) {
		daysAfter := int(date.Sub(baptism).Hours() / 24)
		weekNum := (daysAfter / 7) + 1
		return fmt.Sprintf("Week %d after Baptism of the Lord", weekNum), date.Weekday().String()
	}

	// Post-Pentecost
	if date.After(pentecost) {
		daysSince := int(date.Sub(pentecost).Hours() / 24)

		// Week 1 after Pentecost (days 1-6)
		if daysSince >= 1 && daysSince <= 6 {
			return "Week 1 after Pentecost", date.Weekday().String()
		}

		// Trinity Sunday (day 7)
		if daysSince == 7 {
			return "Trinity Sunday and Following", "Sunday"
		}

		// Trinity week (days 8-13)
		if daysSince >= 8 && daysSince <= 13 {
			return "Trinity Sunday and Following", date.Weekday().String()
		}

		// Week 2+ after Pentecost (day 14+)
		if daysSince >= 14 {
			weekNum := ((daysSince - 14) / 7) + 2
			return fmt.Sprintf("Week %d after Pentecost", weekNum), date.Weekday().String()
		}
	}

	return "UNKNOWN", "UNKNOWN"
}
