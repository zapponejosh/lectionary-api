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
// This allows us to use either *database.DB or *database.Tx.
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
func (dr *DateResolver) ResolveDate(ctx context.Context, date time.Time) (*ResolvedPosition, error) {
	year := date.Year()
	yearCycle := GetYearCycle(date)

	// Get key liturgical dates
	easter := CalculateEaster(year)
	advent := CalculateAdvent(year)
	ashWednesday := CalculateAshWednesday(year)
	pentecost := CalculatePentecost(year)

	// Check if date is before Advent (belongs to previous liturgical year)
	if date.Before(advent) {
		prevYear := year - 1
		prevAdvent := CalculateAdvent(prevYear)
		prevEaster := CalculateEaster(prevYear)
		prevAshWednesday := CalculateAshWednesday(prevYear)
		prevPentecost := CalculatePentecost(prevYear)

		// Use previous year's dates for calculation
		return dr.resolveDateWithContext(ctx, date, prevYear, yearCycle, prevAdvent, prevEaster, prevAshWednesday, prevPentecost)
	}

	return dr.resolveDateWithContext(ctx, date, year, yearCycle, advent, easter, ashWednesday, pentecost)
}

func (dr *DateResolver) resolveDateWithContext(
	ctx context.Context,
	date time.Time,
	year int,
	yearCycle int,
	advent time.Time,
	easter time.Time,
	ashWednesday time.Time,
	pentecost time.Time,
) (*ResolvedPosition, error) {
	// ============================================================================
	// 1. FIXED DAYS - Direct date matching
	// ============================================================================
	if fixed := dr.resolveFixedDay(ctx, date); fixed != nil {
		return &ResolvedPosition{
			Period:        fixed.Period,
			DayIdentifier: fixed.DayIdentifier,
			YearCycle:     yearCycle,
		}, nil
	}

	// ============================================================================
	// 2. ADVENT WEEKS (1st-4th Week of Advent)
	// ============================================================================
	if date.After(advent.AddDate(0, 0, -1)) && date.Before(time.Date(year, time.December, 25, 0, 0, 0, 0, time.UTC)) {
		adventWeek := dr.resolveAdventWeek(date, advent)
		if adventWeek != nil {
			return &ResolvedPosition{
				Period:        adventWeek.Period,
				DayIdentifier: adventWeek.DayIdentifier,
				YearCycle:     yearCycle,
			}, nil
		}
	}

	// ============================================================================
	// 3. CHRISTMAS SEASON (Dec 25 - Jan 5)
	// ============================================================================
	if christmas := dr.resolveChristmasSeason(date, year); christmas != nil {
		return &ResolvedPosition{
			Period:        christmas.Period,
			DayIdentifier: christmas.DayIdentifier,
			YearCycle:     yearCycle,
		}, nil
	}

	// ============================================================================
	// 4. EPIPHANY AND FOLLOWING (Jan 6-12)
	// ============================================================================
	if epiphany := dr.resolveEpiphany(date, year); epiphany != nil {
		return &ResolvedPosition{
			Period:        epiphany.Period,
			DayIdentifier: epiphany.DayIdentifier,
			YearCycle:     yearCycle,
		}, nil
	}

	// ============================================================================
	// 5. BAPTISM OF THE LORD & WEEKS AFTER
	// ============================================================================
	if baptism := dr.resolveBaptismAndFollowing(ctx, date, year, ashWednesday); baptism != nil {
		return &ResolvedPosition{
			Period:        baptism.Period,
			DayIdentifier: baptism.DayIdentifier,
			YearCycle:     yearCycle,
		}, nil
	}

	// ============================================================================
	// 6. DATED WEEKS (Week following Sun. between Feb. X and Y)
	// ============================================================================
	if dated := dr.resolveDatedWeek(ctx, date, year, ashWednesday); dated != nil {
		return &ResolvedPosition{
			Period:        dated.Period,
			DayIdentifier: dated.DayIdentifier,
			YearCycle:     yearCycle,
		}, nil
	}

	// ============================================================================
	// 7. ASH WEDNESDAY AND FOLLOWING
	// ============================================================================
	if ash := dr.resolveAshWednesday(date, ashWednesday); ash != nil {
		return &ResolvedPosition{
			Period:        ash.Period,
			DayIdentifier: ash.DayIdentifier,
			YearCycle:     yearCycle,
		}, nil
	}

	// ============================================================================
	// 8. LENT WEEKS (1st-6th Week of Lent)
	// ============================================================================
	if lent := dr.resolveLentWeek(date, ashWednesday); lent != nil {
		return &ResolvedPosition{
			Period:        lent.Period,
			DayIdentifier: lent.DayIdentifier,
			YearCycle:     yearCycle,
		}, nil
	}

	// ============================================================================
	// 9. HOLY WEEK
	// ============================================================================
	if holyWeek := dr.resolveHolyWeek(date, easter); holyWeek != nil {
		return &ResolvedPosition{
			Period:        holyWeek.Period,
			DayIdentifier: holyWeek.DayIdentifier,
			YearCycle:     yearCycle,
		}, nil
	}

	// ============================================================================
	// 10. EASTER WEEKS (1st-7th Week of Easter)
	// ============================================================================
	if easterWeek := dr.resolveEasterWeek(date, easter, pentecost); easterWeek != nil {
		return &ResolvedPosition{
			Period:        easterWeek.Period,
			DayIdentifier: easterWeek.DayIdentifier,
			YearCycle:     yearCycle,
		}, nil
	}

	// ============================================================================
	// 11. PENTECOST AND FOLLOWING
	// ============================================================================
	if pentecostWeek := dr.resolvePentecostAndFollowing(ctx, date, pentecost, advent); pentecostWeek != nil {
		return &ResolvedPosition{
			Period:        pentecostWeek.Period,
			DayIdentifier: pentecostWeek.DayIdentifier,
			YearCycle:     yearCycle,
		}, nil
	}

	return nil, fmt.Errorf("could not resolve date %s to lectionary position", date.Format("2006-01-02"))
}

// resolveFixedDay handles fixed calendar dates like Christmas, Epiphany, etc.
func (dr *DateResolver) resolveFixedDay(ctx context.Context, date time.Time) *ResolvedPosition {
	month := date.Month()
	day := date.Day()

	// Christmas Day - December 25
	if month == time.December && day == 25 {
		dayName := DayName(date)
		// Check if it's stored as "December 25" or "Sunday"
		day, err := dr.db.GetDayByPosition(ctx, "Christmas", "December 25")
		if err == nil && day != nil {
			return &ResolvedPosition{
				Period:        day.Period,
				DayIdentifier: day.DayIdentifier,
			}
		}
		// Fallback to day name
		day, err = dr.db.GetDayByPosition(ctx, "Christmas", dayName)
		if err == nil && day != nil {
			return &ResolvedPosition{
				Period:        day.Period,
				DayIdentifier: day.DayIdentifier,
			}
		}
	}

	// Epiphany - January 6
	if month == time.January && day == 6 {
		day, err := dr.db.GetDayByPosition(ctx, "Epiphany and Following", "January 6")
		if err == nil && day != nil {
			return &ResolvedPosition{
				Period:        day.Period,
				DayIdentifier: day.DayIdentifier,
			}
		}
	}

	return nil
}

// resolveAdventWeek resolves dates in Advent (weeks 1-4)
func (dr *DateResolver) resolveAdventWeek(date time.Time, advent time.Time) *ResolvedPosition {
	daysSinceAdvent := int(date.Sub(advent).Hours() / 24)
	weekNum := (daysSinceAdvent / 7) + 1

	if weekNum >= 1 && weekNum <= 4 {
		period := fmt.Sprintf("%s Week of Advent", Ordinal(weekNum))
		
		// 4th Week of Advent uses date-based identifiers (December 17-24)
		// instead of day names
		var dayIdentifier string
		if weekNum == 4 {
			monthName := date.Month().String()
			dayIdentifier = fmt.Sprintf("%s %d", monthName, date.Day())
		} else {
			dayIdentifier = DayName(date)
		}
		
		return &ResolvedPosition{
			Period:        period,
			DayIdentifier: dayIdentifier,
		}
	}

	return nil
}

// resolveChristmasSeason resolves dates in Christmas season (Dec 25 - Jan 5)
func (dr *DateResolver) resolveChristmasSeason(date time.Time, year int) *ResolvedPosition {
	month := date.Month()
	day := date.Day()

	// December 25-31
	if month == time.December && day >= 25 {
		dayIdentifier := fmt.Sprintf("December %d", day)
		return &ResolvedPosition{
			Period:        "Christmas Season",
			DayIdentifier: dayIdentifier,
		}
	}

	// January 1-5
	if month == time.January && day <= 5 {
		dayIdentifier := fmt.Sprintf("January %d", day)
		return &ResolvedPosition{
			Period:        "Christmas Season",
			DayIdentifier: dayIdentifier,
		}
	}

	return nil
}

// resolveEpiphany resolves Epiphany and Following (Jan 6-12)
func (dr *DateResolver) resolveEpiphany(date time.Time, year int) *ResolvedPosition {
	month := date.Month()
	day := date.Day()

	if month == time.January && day >= 6 && day <= 12 {
		dayIdentifier := fmt.Sprintf("January %d", day)
		return &ResolvedPosition{
			Period:        "Epiphany and Following",
			DayIdentifier: dayIdentifier,
		}
	}

	return nil
}

// resolveBaptismAndFollowing resolves Baptism of the Lord and weeks after
func (dr *DateResolver) resolveBaptismAndFollowing(ctx context.Context, date time.Time, year int, ashWednesday time.Time) *ResolvedPosition {
	// Baptism of the Lord is the Sunday between Jan 7-13
	baptismSunday := FindSundayBetween(year, 1, 7, 1, 13)
	if baptismSunday == nil {
		return nil
	}

	// Check if date is in Baptism of the Lord week or weeks after
	if date.Before(ashWednesday) {
		// Check if it's the Baptism Sunday itself
		if date.Year() == baptismSunday.Year() && date.Month() == baptismSunday.Month() && date.Day() == baptismSunday.Day() {
			return &ResolvedPosition{
				Period:        "Baptism of the Lord",
				DayIdentifier: "Sunday",
			}
		}

		// Check for "Week N after Baptism of the Lord"
		daysAfterBaptism := int(date.Sub(*baptismSunday).Hours() / 24)
		if daysAfterBaptism > 0 && daysAfterBaptism < 7*10 { // Up to 10 weeks
			weekNum := (daysAfterBaptism / 7) + 1
			dayName := DayName(date)
			period := fmt.Sprintf("Week %d after Baptism of the Lord", weekNum)
			return &ResolvedPosition{
				Period:        period,
				DayIdentifier: dayName,
			}
		}
	}

	return nil
}

// resolveDatedWeek resolves "Week following Sun. between Feb. X and Y" periods
func (dr *DateResolver) resolveDatedWeek(ctx context.Context, date time.Time, year int, ashWednesday time.Time) *ResolvedPosition {
	// Get all dated week periods from database
	days, err := dr.db.GetDaysByPeriodType(ctx, database.PeriodTypeDated)
	if err != nil {
		return nil
	}

	for _, day := range days {
		startMonth, startDay, endMonth, endDay, err := ParseDatedWeekPeriod(day.Period)
		if err != nil {
			continue
		}

		sunday := FindSundayBetween(year, startMonth, startDay, endMonth, endDay)
		if sunday == nil {
			continue
		}

		// The week following starts the day after the Sunday
		weekStart := sunday.AddDate(0, 0, 1)
		weekEnd := weekStart.AddDate(0, 0, 7)

		// Check if date falls in this week and before Ash Wednesday
		if (date.After(weekStart) || date.Equal(weekStart)) && date.Before(weekEnd) && date.Before(ashWednesday) {
			dayName := DayName(date)
			return &ResolvedPosition{
				Period:        day.Period,
				DayIdentifier: dayName,
			}
		}
	}

	return nil
}

// resolveAshWednesday resolves Ash Wednesday and following days
func (dr *DateResolver) resolveAshWednesday(date time.Time, ashWednesday time.Time) *ResolvedPosition {
	daysSinceAsh := int(date.Sub(ashWednesday).Hours() / 24)

	if daysSinceAsh >= 0 && daysSinceAsh <= 3 {
		dayNames := []string{"Wednesday", "Thursday", "Friday", "Saturday"}
		if daysSinceAsh < len(dayNames) {
			return &ResolvedPosition{
				Period:        "Ash Wednesday and Following",
				DayIdentifier: dayNames[daysSinceAsh],
			}
		}
	}

	return nil
}

// resolveLentWeek resolves Lent weeks (1st-6th Week of Lent)
func (dr *DateResolver) resolveLentWeek(date time.Time, ashWednesday time.Time) *ResolvedPosition {
	// First Sunday of Lent is the Sunday after Ash Wednesday
	firstSundayOfLent := ashWednesday
	for firstSundayOfLent.Weekday() != time.Sunday {
		firstSundayOfLent = firstSundayOfLent.AddDate(0, 0, 1)
	}

	if date.Before(firstSundayOfLent) {
		return nil
	}

	daysSinceFirstSunday := int(date.Sub(firstSundayOfLent).Hours() / 24)
	weekNum := (daysSinceFirstSunday / 7) + 1

	if weekNum >= 1 && weekNum <= 6 {
		dayName := DayName(date)
		period := fmt.Sprintf("%s Week of Lent", Ordinal(weekNum))
		return &ResolvedPosition{
			Period:        period,
			DayIdentifier: dayName,
		}
	}

	return nil
}

// resolveHolyWeek resolves Holy Week (Palm Sunday through Holy Saturday)
func (dr *DateResolver) resolveHolyWeek(date time.Time, easter time.Time) *ResolvedPosition {
	palmSunday := easter.AddDate(0, 0, -7)
	daysSincePalm := int(date.Sub(palmSunday).Hours() / 24)

	if daysSincePalm >= 0 && daysSincePalm < 7 {
		dayName := DayName(date)
		return &ResolvedPosition{
			Period:        "Holy Week",
			DayIdentifier: dayName,
		}
	}

	return nil
}

// resolveEasterWeek resolves Easter weeks (1st-7th Week of Easter)
func (dr *DateResolver) resolveEasterWeek(date time.Time, easter time.Time, pentecost time.Time) *ResolvedPosition {
	if date.Before(easter) || date.After(pentecost) || date.Equal(pentecost) {
		return nil
	}

	daysSinceEaster := int(date.Sub(easter).Hours() / 24)
	weekNum := (daysSinceEaster / 7) + 1

	if weekNum >= 1 && weekNum <= 7 {
		dayName := DayName(date)
		period := fmt.Sprintf("%s Week of Easter", Ordinal(weekNum))
		return &ResolvedPosition{
			Period:        period,
			DayIdentifier: dayName,
		}
	}

	return nil
}

// resolvePentecostAndFollowing resolves Pentecost and weeks after until Advent
func (dr *DateResolver) resolvePentecostAndFollowing(ctx context.Context, date time.Time, pentecost time.Time, nextAdvent time.Time) *ResolvedPosition {
	if date.Before(pentecost) {
		return nil
	}

	// Pentecost Sunday itself
	if date.Year() == pentecost.Year() && date.Month() == pentecost.Month() && date.Day() == pentecost.Day() {
		return &ResolvedPosition{
			Period:        "Pentecost",
			DayIdentifier: "Sunday",
		}
	}

	// Weeks after Pentecost
	daysSincePentecost := int(date.Sub(pentecost).Hours() / 24)
	weekNum := (daysSincePentecost / 7) + 1

	// Limit to reasonable number of weeks (up to Advent)
	if date.Before(nextAdvent) && weekNum >= 1 && weekNum <= 30 {
		dayName := DayName(date)
		period := fmt.Sprintf("Week %d after Pentecost", weekNum)
		return &ResolvedPosition{
			Period:        period,
			DayIdentifier: dayName,
		}
	}

	return nil
}

// ParseDateString parses a date string in YYYY-MM-DD format
func ParseDateString(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// FormatDate formats a date as YYYY-MM-DD
func FormatDate(date time.Time) string {
	return date.Format("2006-01-02")
}
