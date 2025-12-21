// Package calendar provides liturgical calendar calculations.
package calendar

import (
	"time"
)

// CalculateEaster calculates the date of Easter Sunday for a given year
// using the computus algorithm for the Gregorian calendar.
//
// The algorithm is based on the method described by J.M. Oudin (1940)
// and is valid for all years in the Gregorian calendar.
func CalculateEaster(year int) time.Time {
	// Computus algorithm for Gregorian calendar
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

// CalculateAdvent calculates the date of the first Sunday of Advent
// (the 4th Sunday before Christmas) for a given year.
//
// Advent Sunday is the Sunday closest to November 30, which means
// it falls between November 27 and December 3.
func CalculateAdvent(year int) time.Time {
	// Start with November 30 of the year
	nov30 := time.Date(year, time.November, 30, 0, 0, 0, 0, time.UTC)

	// Find the Sunday on or before November 30
	// Sunday is weekday 0, so we subtract the weekday
	daysToSubtract := int(nov30.Weekday())
	advent := nov30.AddDate(0, 0, -daysToSubtract)

	return advent
}

// CalculateAshWednesday calculates Ash Wednesday for a given year.
// Ash Wednesday is 46 days before Easter (40 days of Lent + 6 days of Holy Week).
func CalculateAshWednesday(year int) time.Time {
	easter := CalculateEaster(year)
	return easter.AddDate(0, 0, -46)
}

// CalculateAscension calculates Ascension Day for a given year.
// Ascension is 39 days after Easter (always on a Thursday).
func CalculateAscension(year int) time.Time {
	easter := CalculateEaster(year)
	return easter.AddDate(0, 0, 39)
}

// CalculatePentecost calculates Pentecost Sunday for a given year.
// Pentecost is 49 days after Easter (7 weeks).
func CalculatePentecost(year int) time.Time {
	easter := CalculateEaster(year)
	return easter.AddDate(0, 0, 49)
}
