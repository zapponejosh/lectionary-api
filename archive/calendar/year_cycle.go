package calendar

import "time"

// Year cycle constants
const (
	// Cycle1 represents Year 1 of the two-year lectionary cycle.
	Cycle1 = 1

	// Cycle2 represents Year 2 of the two-year lectionary cycle.
	Cycle2 = 2

	// ReferenceYear is the liturgical year we use as a baseline for cycle calculation.
	// The liturgical year starting with Advent 2024 is Cycle 1.
	ReferenceYear = 2024

	// ReferenceCycle is the cycle for the reference year.
	ReferenceCycle = Cycle1
)

// GetYearCycle determines which year cycle (1 or 2) applies to a given date.
//
// The lectionary operates on a two-year cycle. The liturgical year begins
// on the first Sunday of Advent (late November/early December), not January 1.
//
// Cycle determination:
//   - The liturgical year starting Advent 2024 is Cycle 1
//   - The liturgical year starting Advent 2025 is Cycle 2
//   - The pattern alternates each liturgical year
//
// Examples:
//   - December 1, 2024 (after Advent 2024): Cycle 1
//   - November 15, 2024 (before Advent 2024): Cycle 2 (still in previous liturgical year)
//   - March 15, 2025: Cycle 1 (between Advent 2024 and Advent 2025)
//   - December 15, 2025 (after Advent 2025): Cycle 2
func GetYearCycle(date time.Time) int {
	year := date.Year()
	advent := CalculateAdvent(year)

	// Determine which liturgical year this date belongs to.
	// If the date is before Advent of its calendar year, it belongs
	// to the liturgical year that started the previous Advent.
	liturgicalYear := year
	if date.Before(advent) {
		liturgicalYear = year - 1
	}

	// Calculate offset from reference year
	yearsSinceReference := liturgicalYear - ReferenceYear

	// Determine cycle based on whether offset is even or odd
	// Even offset (0, 2, 4, ...): same as reference cycle
	// Odd offset (1, 3, 5, ...): opposite of reference cycle
	if yearsSinceReference%2 == 0 {
		return ReferenceCycle
	}

	// Return the opposite cycle
	if ReferenceCycle == Cycle1 {
		return Cycle2
	}
	return Cycle1
}

// GetLiturgicalYear returns the starting year of the liturgical year
// that contains the given date.
//
// The liturgical year is identified by the year in which its Advent begins.
// For example, the liturgical year "2024" runs from Advent 2024 through
// the Saturday before Advent 2025.
func GetLiturgicalYear(date time.Time) int {
	year := date.Year()
	advent := CalculateAdvent(year)

	if date.Before(advent) {
		return year - 1
	}
	return year
}
