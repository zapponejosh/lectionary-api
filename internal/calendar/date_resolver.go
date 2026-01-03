package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/zapponejosh/lectionary-api/internal/database"
)

// ResolvedPosition represents a lectionary position resolved from a calendar date.
type ResolvedPosition struct {
	Period        string
	DayIdentifier string
	YearCycle     int
}

// DateResolver resolves calendar dates to lectionary positions.
type DateResolver struct {
	db Queryable
}

// Queryable is an interface for database queries.
// This allows us to use either *database.DB or *database.Tx, and enables
// easy mocking in tests.
type Queryable interface {
	GetDaysByPeriodType(ctx context.Context, periodType database.PeriodType) ([]database.LectionaryDay, error)
	GetDayByPosition(ctx context.Context, period, dayIdentifier string) (*database.LectionaryDay, error)
}

// NewDateResolver creates a new date resolver.
func NewDateResolver(db Queryable) *DateResolver {
	return &DateResolver{db: db}
}

// ResolveDate converts a calendar date to a lectionary position.
// Returns the period, day identifier, and year cycle for the given date.
//
// The resolution follows this priority:
//  1. Fixed days (Christmas, Epiphany)
//  2. Advent weeks
//  3. Christmas season
//  4. Epiphany and following
//  5. Baptism of the Lord and weeks after
//  6. Dated weeks (variable Epiphany-Lent transition)
//  7. Ash Wednesday and following
//  8. Lent weeks
//  9. Holy Week
//  10. Easter weeks
//  11. Pentecost and weeks after (including Trinity Sunday)
func (dr *DateResolver) ResolveDate(ctx context.Context, date time.Time) (*ResolvedPosition, error) {
	// Normalize the date to midnight UTC for consistent comparisons
	date = NormalizeToMidnight(date)

	year := date.Year()
	yearCycle := GetYearCycle(date)

	// Get key liturgical dates for this CALENDAR year.
	// Important: Easter, Ash Wednesday, Pentecost are based on the calendar year
	// of the date being resolved, NOT the liturgical year.
	// The liturgical year only affects which year cycle (1 or 2) we use.
	easter := CalculateEaster(year)
	ashWednesday := CalculateAshWednesday(year)
	pentecost := CalculatePentecost(year)

	// Advent for this calendar year (used as boundary for Pentecost season)
	advent := CalculateAdvent(year)

	// For dates in Advent/Christmas (after this year's Advent), we need
	// to check against THIS year's Advent, not next year's
	// For dates before Advent (Jan-Nov), Advent serves as the end boundary
	// for the Pentecost season

	return dr.resolveDateWithContext(ctx, date, year, yearCycle, advent, easter, ashWednesday, pentecost)
}

// resolveDateWithContext resolves a date using pre-calculated liturgical dates.
func (dr *DateResolver) resolveDateWithContext(
	ctx context.Context,
	date time.Time,
	year int,
	yearCycle int,
	advent, easter, ashWednesday, pentecost time.Time,
) (*ResolvedPosition, error) {

	// Resolution chain - order matters!
	resolvers := []func() *ResolvedPosition{
		func() *ResolvedPosition { return dr.resolveFixedDay(ctx, date) },
		func() *ResolvedPosition { return dr.resolveAdventWeek(date, advent, year) },
		func() *ResolvedPosition { return dr.resolveChristmasSeason(date, year) },
		func() *ResolvedPosition { return dr.resolveEpiphany(date) },
		func() *ResolvedPosition { return dr.resolveBaptismAndFollowing(ctx, date, year, ashWednesday) },
		func() *ResolvedPosition { return dr.resolveDatedWeek(ctx, date, year, ashWednesday) },
		func() *ResolvedPosition { return dr.resolveAshWednesday(date, ashWednesday) },
		func() *ResolvedPosition { return dr.resolveLentWeek(date, ashWednesday, easter) },
		func() *ResolvedPosition { return dr.resolveHolyWeek(date, easter) },
		func() *ResolvedPosition { return dr.resolveEasterWeek(date, easter, pentecost) },
		func() *ResolvedPosition { return dr.resolvePentecostAndFollowing(date, pentecost, advent) },
	}

	for _, resolve := range resolvers {
		if pos := resolve(); pos != nil {
			pos.YearCycle = yearCycle
			return pos, nil
		}
	}

	return nil, fmt.Errorf("could not resolve date %s to lectionary position", FormatDate(date))
}

// resolveFixedDay handles fixed calendar dates like Christmas, Epiphany, etc.
func (dr *DateResolver) resolveFixedDay(ctx context.Context, date time.Time) *ResolvedPosition {
	month := date.Month()
	dayOfMonth := date.Day()

	// Christmas Day - December 25
	if month == time.December && dayOfMonth == 25 {
		// Try "December 25" identifier first
		if lday, err := dr.db.GetDayByPosition(ctx, "Christmas", "December 25"); err == nil && lday != nil {
			return &ResolvedPosition{
				Period:        lday.Period,
				DayIdentifier: lday.DayIdentifier,
			}
		}
		// Fallback to day name (some lectionaries use the day name for Christmas)
		dayName := DayName(date)
		if lday, err := dr.db.GetDayByPosition(ctx, "Christmas", dayName); err == nil && lday != nil {
			return &ResolvedPosition{
				Period:        lday.Period,
				DayIdentifier: lday.DayIdentifier,
			}
		}
	}

	// Epiphany - January 6
	if month == time.January && dayOfMonth == 6 {
		if lday, err := dr.db.GetDayByPosition(ctx, "Epiphany and Following", "January 6"); err == nil && lday != nil {
			return &ResolvedPosition{
				Period:        lday.Period,
				DayIdentifier: lday.DayIdentifier,
			}
		}
	}

	return nil
}

// resolveAdventWeek resolves dates in Advent (weeks 1-4).
func (dr *DateResolver) resolveAdventWeek(date time.Time, advent time.Time, year int) *ResolvedPosition {
	christmas := time.Date(year, time.December, 25, 0, 0, 0, 0, time.UTC)

	// Must be between Advent Sunday (inclusive) and Christmas (exclusive)
	if date.Before(advent) || !date.Before(christmas) {
		return nil
	}

	daysSinceAdvent := DaysBetween(advent, date)
	weekNum := (daysSinceAdvent / 7) + 1

	if weekNum < 1 || weekNum > AdventWeeks {
		return nil
	}

	period := fmt.Sprintf("%s Week of Advent", Ordinal(weekNum))

	// 4th Week of Advent uses date-based identifiers (December 17-24)
	// because it has variable length depending on when Christmas falls
	var dayIdentifier string
	if weekNum == 4 {
		dayIdentifier = fmt.Sprintf("%s %d", date.Month().String(), date.Day())
	} else {
		dayIdentifier = DayName(date)
	}

	return &ResolvedPosition{
		Period:        period,
		DayIdentifier: dayIdentifier,
	}
}

// resolveChristmasSeason resolves dates in Christmas season (Dec 25 - Jan 5).
func (dr *DateResolver) resolveChristmasSeason(date time.Time, year int) *ResolvedPosition {
	month := date.Month()
	dayOfMonth := date.Day()

	// December 25-31 of current year
	if month == time.December && dayOfMonth >= 25 {
		return &ResolvedPosition{
			Period:        "Christmas Season",
			DayIdentifier: fmt.Sprintf("December %d", dayOfMonth),
		}
	}

	// January 1-5 (next calendar year but same liturgical season)
	// We need to check if this is the January following Christmas
	if month == time.January && dayOfMonth >= 1 && dayOfMonth <= 5 {
		return &ResolvedPosition{
			Period:        "Christmas Season",
			DayIdentifier: fmt.Sprintf("January %d", dayOfMonth),
		}
	}

	return nil
}

// resolveEpiphany resolves Epiphany and Following (Jan 6-12).
func (dr *DateResolver) resolveEpiphany(date time.Time) *ResolvedPosition {
	month := date.Month()
	dayOfMonth := date.Day()

	if month == time.January && dayOfMonth >= 6 && dayOfMonth <= 12 {
		return &ResolvedPosition{
			Period:        "Epiphany and Following",
			DayIdentifier: fmt.Sprintf("January %d", dayOfMonth),
		}
	}

	return nil
}

// resolveBaptismAndFollowing resolves Baptism of the Lord and weeks 1-4 after.
// Weeks 5+ are handled by resolveDatedWeek using "Week following Sun. between..." periods.
func (dr *DateResolver) resolveBaptismAndFollowing(ctx context.Context, date time.Time, year int, ashWednesday time.Time) *ResolvedPosition {
	// Don't process if we're past Ash Wednesday
	if !date.Before(ashWednesday) {
		return nil
	}

	// Baptism of the Lord is the Sunday between Jan 7-13
	baptismSunday := FindSundayBetween(year, 1, 7, 1, 13)
	if baptismSunday == nil {
		return nil
	}

	// Check if it's the Baptism Sunday itself
	if IsSameDay(date, *baptismSunday) {
		return &ResolvedPosition{
			Period:        "Baptism of the Lord",
			DayIdentifier: "Sunday",
		}
	}

	// Check for weeks 1-4 after Baptism of the Lord
	// The database only has weeks 1-4; weeks 5+ use dated weeks
	daysAfterBaptism := DaysBetween(*baptismSunday, date)
	if daysAfterBaptism > 0 {
		weekNum := (daysAfterBaptism / 7) + 1
		// Only handle weeks 1-4; let resolveDatedWeek handle weeks 5+
		if weekNum >= 1 && weekNum <= 4 {
			return &ResolvedPosition{
				Period:        fmt.Sprintf("Week %d after Baptism of the Lord", weekNum),
				DayIdentifier: DayName(date),
			}
		}
	}

	return nil
}

// resolveDatedWeek resolves "Week following Sun. between Feb. X and Y" periods.
// These are transitional weeks between Epiphany season and Lent.
// Note: Despite the name "Week following", these periods include the Sunday itself.
func (dr *DateResolver) resolveDatedWeek(ctx context.Context, date time.Time, year int, ashWednesday time.Time) *ResolvedPosition {
	// Must be before Ash Wednesday
	if !date.Before(ashWednesday) {
		return nil
	}

	// Get all dated week periods from database
	days, err := dr.db.GetDaysByPeriodType(ctx, database.PeriodTypeDated)
	if err != nil {
		return nil
	}

	// Build a unique set of periods (we get multiple rows per period)
	seenPeriods := make(map[string]bool)
	var periods []string
	for _, lday := range days {
		if !seenPeriods[lday.Period] {
			seenPeriods[lday.Period] = true
			periods = append(periods, lday.Period)
		}
	}

	for _, period := range periods {
		startMonth, startDay, endMonth, endDay, err := ParseDatedWeekPeriod(period)
		if err != nil {
			continue
		}

		sunday := FindSundayBetween(year, startMonth, startDay, endMonth, endDay)
		if sunday == nil {
			continue
		}

		// The week includes Sunday through Saturday (7 days)
		weekStart := *sunday
		weekEnd := sunday.AddDate(0, 0, 6) // Saturday

		// Check if date falls in this week (Sunday through Saturday)
		if (date.Equal(weekStart) || date.After(weekStart)) && !date.After(weekEnd) {
			return &ResolvedPosition{
				Period:        period,
				DayIdentifier: DayName(date),
			}
		}
	}

	return nil
}

// resolveAshWednesday resolves Ash Wednesday and following days (Wed-Sat).
func (dr *DateResolver) resolveAshWednesday(date time.Time, ashWednesday time.Time) *ResolvedPosition {
	daysSinceAsh := DaysBetween(ashWednesday, date)

	// Ash Wednesday through Saturday (4 days: Wed, Thu, Fri, Sat)
	if daysSinceAsh >= 0 && daysSinceAsh <= 3 {
		dayNames := []string{"Wednesday", "Thursday", "Friday", "Saturday"}
		return &ResolvedPosition{
			Period:        "Ash Wednesday and Following",
			DayIdentifier: dayNames[daysSinceAsh],
		}
	}

	return nil
}

// resolveLentWeek resolves Lent weeks (1st-5th Week of Lent).
// Note: There is no 6th Week of Lent - that becomes Holy Week.
func (dr *DateResolver) resolveLentWeek(date time.Time, ashWednesday, easter time.Time) *ResolvedPosition {
	// First Sunday of Lent is the Sunday after Ash Wednesday
	firstSundayOfLent := ashWednesday
	for firstSundayOfLent.Weekday() != time.Sunday {
		firstSundayOfLent = firstSundayOfLent.AddDate(0, 0, 1)
	}

	// Holy Week starts on Palm Sunday (7 days before Easter)
	palmSunday := easter.AddDate(0, 0, -7)

	if date.Before(firstSundayOfLent) || !date.Before(palmSunday) {
		return nil
	}

	daysSinceFirstSunday := DaysBetween(firstSundayOfLent, date)
	weekNum := (daysSinceFirstSunday / 7) + 1

	// Only weeks 1-5 of Lent (week 6 is Holy Week)
	if weekNum >= 1 && weekNum <= 5 {
		return &ResolvedPosition{
			Period:        fmt.Sprintf("%s Week of Lent", Ordinal(weekNum)),
			DayIdentifier: DayName(date),
		}
	}

	return nil
}

// resolveHolyWeek resolves Holy Week (Palm Sunday through Holy Saturday).
func (dr *DateResolver) resolveHolyWeek(date time.Time, easter time.Time) *ResolvedPosition {
	palmSunday := easter.AddDate(0, 0, -DaysFromEasterToPalmSunday)
	daysSincePalm := DaysBetween(palmSunday, date)

	// Holy Week is Palm Sunday (day 0) through Holy Saturday (day 6)
	if daysSincePalm >= 0 && daysSincePalm < 7 {
		return &ResolvedPosition{
			Period:        "Holy Week",
			DayIdentifier: DayName(date),
		}
	}

	return nil
}

// resolveEasterWeek resolves Easter weeks.
// Note: The first week is called "Easter Week", subsequent weeks are "2nd Week of Easter", etc.
func (dr *DateResolver) resolveEasterWeek(date time.Time, easter, pentecost time.Time) *ResolvedPosition {
	// Easter season is from Easter Sunday up to (but not including) Pentecost
	if date.Before(easter) || !date.Before(pentecost) {
		return nil
	}

	daysSinceEaster := DaysBetween(easter, date)
	weekNum := (daysSinceEaster / 7) + 1

	if weekNum >= 1 && weekNum <= EasterWeeks {
		// First week is "Easter Week", not "1st Week of Easter"
		var period string
		if weekNum == 1 {
			period = "Easter Week"
		} else {
			period = fmt.Sprintf("%s Week of Easter", Ordinal(weekNum))
		}

		return &ResolvedPosition{
			Period:        period,
			DayIdentifier: DayName(date),
		}
	}

	return nil
}

// resolvePentecostAndFollowing resolves Pentecost and weeks after until Advent.
//
// Database structure:
// - Pentecost (Sunday only)
// - Week 1 after Pentecost (Mon-Sat after Pentecost)
// - Trinity Sunday and Following (Sunday only - no weekdays in DB)
// - Week 2 after Pentecost (includes 2nd Sunday after Pentecost + Mon-Sat)
// - Week 3-27 after Pentecost (Sunday + Mon-Sat each)
// - Christ the King (Sunday only - last Sunday before Advent)
func (dr *DateResolver) resolvePentecostAndFollowing(date time.Time, pentecost, nextAdvent time.Time) *ResolvedPosition {
	if date.Before(pentecost) || !date.Before(nextAdvent) {
		return nil
	}

	// Pentecost Sunday itself
	if IsSameDay(date, pentecost) {
		return &ResolvedPosition{
			Period:        "Pentecost",
			DayIdentifier: "Sunday",
		}
	}

	daysSincePentecost := DaysBetween(pentecost, date)

	// Week 1 after Pentecost is Mon-Sat after Pentecost (days 1-6)
	if daysSincePentecost >= 1 && daysSincePentecost <= 6 {
		return &ResolvedPosition{
			Period:        "Week 1 after Pentecost",
			DayIdentifier: DayName(date),
		}
	}

	// Trinity Sunday is the Sunday after Pentecost (day 7)
	// In the database, Trinity Sunday only has Sunday - no weekdays
	if daysSincePentecost == 7 {
		return &ResolvedPosition{
			Period:        "Trinity Sunday and Following",
			DayIdentifier: "Sunday",
		}
	}

	// Check for Christ the King Sunday (last Sunday before Advent)
	// This takes precedence over week numbering
	christTheKing := nextAdvent.AddDate(0, 0, -7) // Sunday before Advent
	if date.Weekday() == time.Sunday && IsSameDay(date, christTheKing) {
		return &ResolvedPosition{
			Period:        "Christ the King",
			DayIdentifier: "Sunday",
		}
	}

	// Days 8+ after Pentecost: Week 2-27 after Pentecost
	if daysSincePentecost >= 8 {
		// Calculate which week we're in
		// Days 8-14: Week 2 (8-14 = days after Trinity week)
		// Days 15-21: Week 3
		// etc.
		daysAfterTrinity := daysSincePentecost - 7 // Day 8 = day 1 after Trinity
		weekNum := (daysAfterTrinity / 7) + 2      // Week 2 = first week after Trinity

		// Cap at MaxWeeksAfterPentecost (27)
		// Weeks beyond 27 still use Week 27 readings (the last available)
		if weekNum > MaxWeeksAfterPentecost {
			weekNum = MaxWeeksAfterPentecost
		}

		if weekNum >= 2 {
			return &ResolvedPosition{
				Period:        fmt.Sprintf("Week %d after Pentecost", weekNum),
				DayIdentifier: DayName(date),
			}
		}
	}

	return nil
}

// ParseDateString parses a date string in YYYY-MM-DD format.
func ParseDateString(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// FormatDate formats a date as YYYY-MM-DD.
func FormatDate(date time.Time) string {
	return date.Format("2006-01-02")
}
