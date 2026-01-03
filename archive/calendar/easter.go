// Package calendar provides liturgical calendar calculations.
package calendar

import (
	"time"
)

// Liturgical calendar constants
const (
	// DaysFromEasterToAshWednesday is the number of days before Easter that Ash Wednesday falls.
	// This is 46 days: 40 days of Lent + 6 Sundays (which aren't counted in Lent).
	DaysFromEasterToAshWednesday = 46

	// DaysFromEasterToAscension is the number of days after Easter for Ascension Thursday.
	DaysFromEasterToAscension = 39

	// DaysFromEasterToPentecost is the number of days after Easter for Pentecost Sunday.
	// This is 7 weeks (49 days).
	DaysFromEasterToPentecost = 49

	// DaysFromEasterToPalmSunday is the number of days before Easter that Palm Sunday falls.
	DaysFromEasterToPalmSunday = 7
)

// CalculateEaster calculates the date of Easter Sunday for a given year
// using the computus algorithm for the Gregorian calendar.
//
// The algorithm is based on the method described by J.M. Oudin (1940)
// and is valid for all years in the Gregorian calendar (1583 onwards).
//
// Easter falls on the first Sunday after the first full moon occurring
// on or after the spring equinox (March 21).
func CalculateEaster(year int) time.Time {
	// Computus algorithm for Gregorian calendar
	// See: https://en.wikipedia.org/wiki/Computus#Anonymous_Gregorian_algorithm
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
// for a given year.
//
// Advent Sunday is the fourth Sunday before Christmas Day, which means
// it's the Sunday nearest to November 30 (St. Andrew's Day). This places
// Advent Sunday between November 27 and December 3 inclusive.
func CalculateAdvent(year int) time.Time {
	// Christmas is December 25
	christmas := time.Date(year, time.December, 25, 0, 0, 0, 0, time.UTC)

	// Find the 4th Sunday before Christmas
	// First, find the Sunday on or before Christmas
	daysToSubtract := int(christmas.Weekday())
	if daysToSubtract == 0 {
		daysToSubtract = 7 // If Christmas is Sunday, go back a full week
	}

	// That gives us the Sunday before Christmas, now go back 3 more weeks
	fourthSundayBefore := christmas.AddDate(0, 0, -daysToSubtract-21)

	return fourthSundayBefore
}

// CalculateAshWednesday calculates Ash Wednesday for a given year.
// Ash Wednesday marks the beginning of Lent, occurring 46 days before Easter.
func CalculateAshWednesday(year int) time.Time {
	easter := CalculateEaster(year)
	return easter.AddDate(0, 0, -DaysFromEasterToAshWednesday)
}

// CalculateAscension calculates Ascension Day for a given year.
// Ascension is 39 days after Easter (always on a Thursday).
func CalculateAscension(year int) time.Time {
	easter := CalculateEaster(year)
	return easter.AddDate(0, 0, DaysFromEasterToAscension)
}

// CalculatePentecost calculates Pentecost Sunday for a given year.
// Pentecost is 49 days after Easter (7 weeks).
func CalculatePentecost(year int) time.Time {
	easter := CalculateEaster(year)
	return easter.AddDate(0, 0, DaysFromEasterToPentecost)
}

// CalculatePalmSunday calculates Palm Sunday for a given year.
// Palm Sunday is the Sunday before Easter, beginning Holy Week.
func CalculatePalmSunday(year int) time.Time {
	easter := CalculateEaster(year)
	return easter.AddDate(0, 0, -DaysFromEasterToPalmSunday)
}
