package main

import (
	"flag"
	"fmt"
	"sort"
	"time"
)

// This script generates sample dates for every liturgical period
// to help identify which periods are missing from your database.

func main() {
	year := flag.Int("year", 2025, "Year to generate dates for")
	flag.Parse()

	fmt.Printf("=== Liturgical Date Generator for %d ===\n\n", *year)

	// Calculate key dates
	easter := calculateEaster(*year)
	advent := calculateAdvent(*year)
	ashWednesday := easter.AddDate(0, 0, -46)
	pentecost := easter.AddDate(0, 0, 49)

	fmt.Println("Key Dates:")
	fmt.Printf("  Advent Start:    %s\n", formatDate(advent))
	fmt.Printf("  Ash Wednesday:   %s\n", formatDate(ashWednesday))
	fmt.Printf("  Easter:          %s\n", formatDate(easter))
	fmt.Printf("  Pentecost:       %s\n", formatDate(pentecost))
	fmt.Println()

	// Collect all test dates with expected periods
	type testDate struct {
		date           time.Time
		expectedPeriod string
		dayIdentifier  string
	}

	var dates []testDate

	// ==========================================================================
	// ADVENT (previous year's Advent starts current liturgical year)
	// ==========================================================================
	prevAdvent := calculateAdvent(*year - 1)
	for week := 1; week <= 4; week++ {
		weekStart := prevAdvent.AddDate(0, 0, (week-1)*7)
		for day := 0; day < 7; day++ {
			d := weekStart.AddDate(0, 0, day)
			// Stop before Christmas
			if d.Month() == time.December && d.Day() >= 25 {
				break
			}
			period := fmt.Sprintf("%s Week of Advent", ordinal(week))
			dayID := d.Weekday().String()
			if week == 4 {
				dayID = fmt.Sprintf("%s %d", d.Month().String(), d.Day())
			}
			dates = append(dates, testDate{d, period, dayID})
		}
	}

	// ==========================================================================
	// CHRISTMAS SEASON (Dec 25 - Jan 5)
	// ==========================================================================
	christmas := time.Date(*year-1, time.December, 25, 0, 0, 0, 0, time.UTC)
	dates = append(dates, testDate{christmas, "Christmas", "December 25"})

	for day := 26; day <= 31; day++ {
		d := time.Date(*year-1, time.December, day, 0, 0, 0, 0, time.UTC)
		dates = append(dates, testDate{d, "Christmas Season", fmt.Sprintf("December %d", day)})
	}
	for day := 1; day <= 5; day++ {
		d := time.Date(*year, time.January, day, 0, 0, 0, 0, time.UTC)
		dates = append(dates, testDate{d, "Christmas Season", fmt.Sprintf("January %d", day)})
	}

	// ==========================================================================
	// EPIPHANY AND FOLLOWING (Jan 6-12)
	// ==========================================================================
	for day := 6; day <= 12; day++ {
		d := time.Date(*year, time.January, day, 0, 0, 0, 0, time.UTC)
		dates = append(dates, testDate{d, "Epiphany and Following", fmt.Sprintf("January %d", day)})
	}

	// ==========================================================================
	// BAPTISM OF THE LORD (Sunday between Jan 7-13)
	// ==========================================================================
	baptismSunday := findSundayBetween(*year, 1, 7, 1, 13)
	if baptismSunday != nil {
		dates = append(dates, testDate{*baptismSunday, "Baptism of the Lord", "Sunday"})

		// Weeks after Baptism
		for week := 1; week <= 8; week++ {
			weekStart := baptismSunday.AddDate(0, 0, week*7)
			if weekStart.After(ashWednesday) || weekStart.Equal(ashWednesday) {
				break
			}
			for day := 0; day < 7; day++ {
				d := weekStart.AddDate(0, 0, day)
				if d.After(ashWednesday) || d.Equal(ashWednesday) {
					break
				}
				period := fmt.Sprintf("Week %d after Baptism of the Lord", week)
				dates = append(dates, testDate{d, period, d.Weekday().String()})
			}
		}
	}

	// ==========================================================================
	// ASH WEDNESDAY AND FOLLOWING
	// ==========================================================================
	ashDays := []string{"Wednesday", "Thursday", "Friday", "Saturday"}
	for i, dayName := range ashDays {
		d := ashWednesday.AddDate(0, 0, i)
		dates = append(dates, testDate{d, "Ash Wednesday and Following", dayName})
	}

	// ==========================================================================
	// LENT (Weeks 1-6)
	// ==========================================================================
	firstSundayOfLent := ashWednesday
	for firstSundayOfLent.Weekday() != time.Sunday {
		firstSundayOfLent = firstSundayOfLent.AddDate(0, 0, 1)
	}

	for week := 1; week <= 6; week++ {
		weekStart := firstSundayOfLent.AddDate(0, 0, (week-1)*7)
		for day := 0; day < 7; day++ {
			d := weekStart.AddDate(0, 0, day)
			palmSunday := easter.AddDate(0, 0, -7)
			if d.Equal(palmSunday) || d.After(palmSunday) {
				break
			}
			period := fmt.Sprintf("%s Week of Lent", ordinal(week))
			dates = append(dates, testDate{d, period, d.Weekday().String()})
		}
	}

	// ==========================================================================
	// HOLY WEEK
	// ==========================================================================
	palmSunday := easter.AddDate(0, 0, -7)
	holyWeekDays := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	for i, dayName := range holyWeekDays {
		d := palmSunday.AddDate(0, 0, i)
		dates = append(dates, testDate{d, "Holy Week", dayName})
	}

	// ==========================================================================
	// EASTER (Weeks 1-7)
	// ==========================================================================
	for week := 1; week <= 7; week++ {
		weekStart := easter.AddDate(0, 0, (week-1)*7)
		for day := 0; day < 7; day++ {
			d := weekStart.AddDate(0, 0, day)
			if d.Equal(pentecost) || d.After(pentecost) {
				break
			}
			period := fmt.Sprintf("%s Week of Easter", ordinal(week))
			dates = append(dates, testDate{d, period, d.Weekday().String()})
		}
	}

	// ==========================================================================
	// PENTECOST AND WEEKS AFTER
	// ==========================================================================
	dates = append(dates, testDate{pentecost, "Pentecost", "Sunday"})

	// Weeks after Pentecost (up to 27 weeks possible)
	for week := 1; week <= 27; week++ {
		weekStart := pentecost.AddDate(0, 0, week*7)
		if weekStart.After(advent) || weekStart.Equal(advent) {
			break
		}
		for day := 0; day < 7; day++ {
			d := weekStart.AddDate(0, 0, day)
			if d.After(advent) || d.Equal(advent) {
				break
			}
			period := fmt.Sprintf("Week %d after Pentecost", week)
			dates = append(dates, testDate{d, period, d.Weekday().String()})
		}
	}

	// Sort by date
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].date.Before(dates[j].date)
	})

	// ==========================================================================
	// OUTPUT
	// ==========================================================================

	// Group by season for summary
	seasonCounts := make(map[string]int)
	for _, d := range dates {
		season := categorize(d.expectedPeriod)
		seasonCounts[season]++
	}

	fmt.Println("Expected entries by season:")
	seasons := []string{"Advent", "Christmas", "Epiphany", "Baptism", "Lent", "Holy Week", "Easter", "Pentecost", "Ordinary Time"}
	for _, s := range seasons {
		if count, ok := seasonCounts[s]; ok {
			fmt.Printf("  %-15s %d days\n", s+":", count)
		}
	}
	fmt.Printf("  %-15s %d days\n", "TOTAL:", len(dates))
	fmt.Println()

	// Output all dates as test cases
	fmt.Println("=== All Test Dates ===")
	fmt.Println("Date,Expected Period,Day Identifier")
	for _, d := range dates {
		fmt.Printf("%s,%s,%s\n", formatDate(d.date), d.expectedPeriod, d.dayIdentifier)
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

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func categorize(period string) string {
	switch {
	case contains(period, "Advent"):
		return "Advent"
	case contains(period, "Christmas"):
		return "Christmas"
	case contains(period, "Epiphany"):
		return "Epiphany"
	case contains(period, "Baptism"):
		return "Baptism"
	case contains(period, "Lent"):
		return "Lent"
	case contains(period, "Holy Week"):
		return "Holy Week"
	case contains(period, "Easter"):
		return "Easter"
	case period == "Pentecost":
		return "Pentecost"
	case contains(period, "after Pentecost"):
		return "Ordinary Time"
	default:
		return "Other"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
