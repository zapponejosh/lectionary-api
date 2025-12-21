package calendar

import "time"

// GetYearCycle determines which year cycle (1 or 2) applies to a given date.
//
// The liturgical year starts on the first Sunday of Advent. The cycle
// alternates each liturgical year. Based on the reference data:
// - 2024: Cycle 2 (Advent 2023)
// - 2025: Cycle 1 (Advent 2024)
//
// We use a reference point: Advent 2024 starts Cycle 1.
func GetYearCycle(date time.Time) int {
	year := date.Year()
	advent := CalculateAdvent(year)

	// If the date is before Advent, it belongs to the previous liturgical year
	if date.Before(advent) {
		advent = CalculateAdvent(year - 1)
		year = year - 1
	}

	// Reference: 2024 liturgical year (starting Advent 2024) is Cycle 1
	// Calculate offset from 2024
	referenceYear := 2024
	referenceCycle := 1

	yearsSinceReference := year - referenceYear
	cycleOffset := yearsSinceReference % 2

	// If offset is 0, same cycle as reference; if 1, opposite
	if cycleOffset == 0 {
		return referenceCycle
	}
	return 3 - referenceCycle // Flip 1->2 or 2->1
}
