#!/usr/bin/env python3
"""
Calculate dates to scrape for the PCUSA daily lectionary - Version 4

CRITICAL INSIGHT: The dated weeks in Ordinary Time (before Lent) don't exist
every year! Easter moves by up to 35 days, which means Ash Wednesday (and thus
the end of Ordinary Time before Lent) varies significantly.

For example:
- "Week following Sun. between Feb. 25 and 29" only exists when Ash Wednesday
  is in March. In the 2023-2035 range, this only happens in:
  - Year 1 (cycle 1): 2033
  - Year 2 (cycle 2): 2028

This script identifies which years contain each dated week and builds a
complete scraping plan that pulls from multiple years to ensure full coverage.
"""

import json
import sys
from datetime import datetime, timedelta
from typing import Optional, List, Dict, Set

def parse_date(s: str) -> datetime:
    return datetime.strptime(s, "%Y-%m-%d")

def format_date(d: datetime) -> str:
    return d.strftime("%Y-%m-%d")

def day_name(d: datetime) -> str:
    return d.strftime("%A")

def is_leap_year(year: int) -> bool:
    return year % 4 == 0 and (year % 100 != 0 or year % 400 == 0)

def find_sunday_between(year: int, start_month: int, start_day: int, 
                        end_month: int, end_day: int) -> Optional[datetime]:
    if end_month == 2 and end_day > 28 and not is_leap_year(year):
        end_day = 28
    if start_month == 2 and start_day > 28 and not is_leap_year(year):
        return None
    try:
        start = datetime(year, start_month, start_day)
        end = datetime(year, end_month, end_day)
    except ValueError:
        return None
    current = start
    while current <= end:
        if current.weekday() == 6:
            return current
        current += timedelta(days=1)
    return None

def ordinal(n: int) -> str:
    if n == 1: return "1st"
    if n == 2: return "2nd"
    if n == 3: return "3rd"
    return f"{n}th"


# Complete liturgical calendar 1993-2040
LITURGICAL_CALENDAR = {
    1993: {"cycle": 1, "advent": "1992-11-29", "ash_wed": "1993-02-24", "easter": "1993-04-11"},
    1994: {"cycle": 2, "advent": "1993-11-28", "ash_wed": "1994-02-16", "easter": "1994-04-03"},
    1995: {"cycle": 1, "advent": "1994-11-27", "ash_wed": "1995-03-01", "easter": "1995-04-16"},
    1996: {"cycle": 2, "advent": "1995-12-03", "ash_wed": "1996-02-21", "easter": "1996-04-07"},
    1997: {"cycle": 1, "advent": "1996-12-01", "ash_wed": "1997-02-12", "easter": "1997-03-30"},
    1998: {"cycle": 2, "advent": "1997-11-30", "ash_wed": "1998-02-25", "easter": "1998-04-12"},
    1999: {"cycle": 1, "advent": "1998-11-29", "ash_wed": "1999-02-17", "easter": "1999-04-04"},
    2000: {"cycle": 2, "advent": "1999-11-28", "ash_wed": "2000-03-08", "easter": "2000-04-23"},
    2001: {"cycle": 1, "advent": "2000-12-03", "ash_wed": "2001-02-28", "easter": "2001-04-15"},
    2002: {"cycle": 2, "advent": "2001-12-02", "ash_wed": "2002-02-13", "easter": "2002-03-31"},
    2003: {"cycle": 1, "advent": "2002-12-01", "ash_wed": "2003-03-05", "easter": "2003-04-20"},
    2004: {"cycle": 2, "advent": "2003-11-30", "ash_wed": "2004-02-25", "easter": "2004-04-11"},
    2005: {"cycle": 1, "advent": "2004-11-28", "ash_wed": "2005-02-09", "easter": "2005-03-27"},
    2006: {"cycle": 2, "advent": "2005-11-27", "ash_wed": "2006-03-01", "easter": "2006-04-16"},
    2007: {"cycle": 1, "advent": "2006-12-03", "ash_wed": "2007-02-21", "easter": "2007-04-08"},
    2008: {"cycle": 2, "advent": "2007-12-02", "ash_wed": "2008-02-06", "easter": "2008-03-23"},
    2009: {"cycle": 1, "advent": "2008-11-30", "ash_wed": "2009-02-25", "easter": "2009-04-12"},
    2010: {"cycle": 2, "advent": "2009-11-29", "ash_wed": "2010-02-17", "easter": "2010-04-04"},
    2011: {"cycle": 1, "advent": "2010-11-28", "ash_wed": "2011-03-09", "easter": "2011-04-24"},
    2012: {"cycle": 2, "advent": "2011-11-27", "ash_wed": "2012-02-22", "easter": "2012-04-08"},
    2013: {"cycle": 1, "advent": "2012-12-02", "ash_wed": "2013-02-13", "easter": "2013-03-31"},
    2014: {"cycle": 2, "advent": "2013-12-01", "ash_wed": "2014-03-05", "easter": "2014-04-20"},
    2015: {"cycle": 1, "advent": "2014-11-30", "ash_wed": "2015-02-18", "easter": "2015-04-05"},
    2016: {"cycle": 2, "advent": "2015-11-29", "ash_wed": "2016-02-10", "easter": "2016-03-27"},
    2017: {"cycle": 1, "advent": "2016-11-27", "ash_wed": "2017-03-01", "easter": "2017-04-16"},
    2018: {"cycle": 2, "advent": "2017-12-03", "ash_wed": "2018-02-14", "easter": "2018-04-01"},
    2019: {"cycle": 1, "advent": "2018-12-02", "ash_wed": "2019-03-06", "easter": "2019-04-21"},
    2020: {"cycle": 2, "advent": "2019-12-01", "ash_wed": "2020-02-26", "easter": "2020-04-12"},
    2021: {"cycle": 1, "advent": "2020-11-29", "ash_wed": "2021-02-17", "easter": "2021-04-04"},
    2022: {"cycle": 2, "advent": "2021-11-28", "ash_wed": "2022-03-02", "easter": "2022-04-17"},
    2023: {"cycle": 1, "advent": "2022-11-27", "ash_wed": "2023-02-22", "easter": "2023-04-09"},
    2024: {"cycle": 2, "advent": "2023-12-03", "ash_wed": "2024-02-14", "easter": "2024-03-31"},
    2025: {"cycle": 1, "advent": "2024-12-01", "ash_wed": "2025-03-05", "easter": "2025-04-20"},
    2026: {"cycle": 2, "advent": "2025-11-30", "ash_wed": "2026-02-18", "easter": "2026-04-05"},
    2027: {"cycle": 1, "advent": "2026-11-29", "ash_wed": "2027-02-10", "easter": "2027-03-28"},
    2028: {"cycle": 2, "advent": "2027-11-28", "ash_wed": "2028-03-01", "easter": "2028-04-16"},
    2029: {"cycle": 1, "advent": "2028-12-03", "ash_wed": "2029-02-14", "easter": "2029-04-01"},
    2030: {"cycle": 2, "advent": "2029-12-02", "ash_wed": "2030-03-06", "easter": "2030-04-21"},
    2031: {"cycle": 1, "advent": "2030-12-01", "ash_wed": "2031-02-26", "easter": "2031-04-13"},
    2032: {"cycle": 2, "advent": "2031-11-30", "ash_wed": "2032-02-11", "easter": "2032-03-28"},
    2033: {"cycle": 1, "advent": "2032-11-28", "ash_wed": "2033-03-02", "easter": "2033-04-17"},
    2034: {"cycle": 2, "advent": "2033-11-27", "ash_wed": "2034-02-22", "easter": "2034-04-09"},
    2035: {"cycle": 1, "advent": "2034-12-03", "ash_wed": "2035-02-07", "easter": "2035-03-25"},
    2036: {"cycle": 2, "advent": "2035-12-02", "ash_wed": "2036-02-27", "easter": "2036-04-13"},
    2037: {"cycle": 1, "advent": "2036-11-30", "ash_wed": "2037-02-18", "easter": "2037-04-05"},
    2038: {"cycle": 2, "advent": "2037-11-29", "ash_wed": "2038-03-10", "easter": "2038-04-25"},
    2039: {"cycle": 1, "advent": "2038-11-28", "ash_wed": "2039-02-23", "easter": "2039-04-10"},
    2040: {"cycle": 2, "advent": "2039-11-27", "ash_wed": "2040-02-15", "easter": "2040-04-01"},
}


# The dated week ranges that vary by year
DATED_WEEK_RANGES = [
    (2, 4, 2, 10, "Week following Sun. between Feb. 4 and 10"),
    (2, 11, 2, 17, "Week following Sun. between Feb. 11 and 17"),
    (2, 18, 2, 24, "Week following Sun. between Feb. 18 and 24"),
    (2, 25, 2, 29, "Week following Sun. between Feb. 25 and 29"),
]


def find_years_with_dated_week(start_m, start_d, end_m, end_d, target_cycle):
    """Find all years where a dated week exists for a given cycle."""
    results = []
    for year, data in LITURGICAL_CALENDAR.items():
        if data['cycle'] != target_cycle:
            continue
        ash_wed = parse_date(data['ash_wed'])
        sunday = find_sunday_between(year, start_m, start_d, end_m, end_d)
        if sunday and sunday < ash_wed:
            results.append(year)
    return results


def generate_dated_week_dates(year: int, start_m: int, start_d: int, 
                               end_m: int, end_d: int, period_name: str) -> List[Dict]:
    """Generate dates for a specific dated week in a specific year."""
    dates = []
    data = LITURGICAL_CALENDAR[year]
    ash_wed = parse_date(data['ash_wed'])
    sunday = find_sunday_between(year, start_m, start_d, end_m, end_d)
    
    if not sunday or sunday >= ash_wed:
        return dates
    
    # Generate days from Sunday until Ash Wednesday (exclusive)
    current = sunday
    while current < ash_wed:
        dates.append({
            "period": period_name,
            "day_identifier": day_name(current),
            "special_name": None,
            "date": format_date(current),
            "source_year": year,
        })
        current += timedelta(days=1)
    
    return dates


def generate_base_year_dates(year: int) -> List[Dict]:
    """
    Generate dates for a liturgical year, EXCLUDING the variable dated weeks.
    (Those will be filled in separately from appropriate years.)
    """
    data = LITURGICAL_CALENDAR[year]
    advent = parse_date(data['advent'])
    ash_wed = parse_date(data['ash_wed'])
    easter = parse_date(data['easter'])
    ascension = easter + timedelta(days=39)
    pentecost = easter + timedelta(days=49)
    
    # Get next year's advent for end boundary
    next_year = year + 1
    if next_year in LITURGICAL_CALENDAR:
        next_advent = parse_date(LITURGICAL_CALENDAR[next_year]['advent'])
    else:
        # Estimate
        next_advent = advent + timedelta(days=364)
    
    dates = []
    
    # ==========================================================================
    # ADVENT (Weeks 1-3)
    # ==========================================================================
    for week in range(1, 4):
        start = advent + timedelta(days=(week - 1) * 7)
        for i, day in enumerate(["Sunday", "Monday", "Tuesday", "Wednesday", 
                                  "Thursday", "Friday", "Saturday"]):
            dates.append({
                "period": f"{ordinal(week)} Week of Advent",
                "day_identifier": day,
                "special_name": None,
                "date": format_date(start + timedelta(days=i)),
                "source_year": year,
            })
    
    # ==========================================================================
    # 4th WEEK OF ADVENT & CHRISTMAS (Fixed dates)
    # ==========================================================================
    dec_year = advent.year
    jan_year = dec_year + 1
    
    for day in range(17, 25):
        dates.append({
            "period": "4th Week of Advent",
            "day_identifier": f"December {day}",
            "special_name": None,
            "date": f"{dec_year}-12-{day:02d}",
            "source_year": year,
        })
    
    dates.append({
        "period": "Christmas",
        "day_identifier": "December 25",
        "special_name": "Christmas Day",
        "date": f"{dec_year}-12-25",
        "source_year": year,
    })
    
    for day in range(26, 32):
        dates.append({
            "period": "Christmas Season",
            "day_identifier": f"December {day}",
            "special_name": None,
            "date": f"{dec_year}-12-{day:02d}",
            "source_year": year,
        })
    
    for day in range(1, 6):
        dates.append({
            "period": "Christmas Season",
            "day_identifier": f"January {day}",
            "special_name": None,
            "date": f"{jan_year}-01-{day:02d}",
            "source_year": year,
        })
    
    # ==========================================================================
    # EPIPHANY (Jan 6-12)
    # ==========================================================================
    for day in range(6, 13):
        dates.append({
            "period": "Epiphany and Following",
            "day_identifier": f"January {day}",
            "special_name": "Epiphany" if day == 6 else None,
            "date": f"{jan_year}-01-{day:02d}",
            "source_year": year,
        })
    
    # ==========================================================================
    # BAPTISM OF THE LORD & WEEKS UNTIL DATED WEEKS START
    # ==========================================================================
    baptism = find_sunday_between(jan_year, 1, 7, 1, 13)
    
    dates.append({
        "period": "Baptism of the Lord",
        "day_identifier": "Sunday",
        "special_name": "Baptism of the Lord",
        "date": format_date(baptism),
        "source_year": year,
    })
    
    # Find when dated weeks start (Feb 4-10 range)
    first_dated_sunday = find_sunday_between(jan_year, 2, 4, 2, 10)
    
    current = baptism + timedelta(days=1)
    week_num = 1
    
    while current < first_dated_sunday:
        dates.append({
            "period": f"Week {week_num} after Baptism of the Lord",
            "day_identifier": day_name(current),
            "special_name": None,
            "date": format_date(current),
            "source_year": year,
        })
        current += timedelta(days=1)
        if current.weekday() == 6:
            week_num += 1
    
    # NOTE: Dated weeks (Feb 4-10, Feb 11-17, etc.) are handled separately!
    
    # ==========================================================================
    # ASH WEDNESDAY & FOLLOWING
    # ==========================================================================
    for i, (day, special) in enumerate([
        ("Wednesday", "Ash Wednesday"),
        ("Thursday", None),
        ("Friday", None),
        ("Saturday", None),
    ]):
        dates.append({
            "period": "Ash Wednesday and Following",
            "day_identifier": day,
            "special_name": special,
            "date": format_date(ash_wed + timedelta(days=i)),
            "source_year": year,
        })
    
    # ==========================================================================
    # LENT (Weeks 1-5)
    # ==========================================================================
    lent1 = ash_wed + timedelta(days=4)
    
    for week in range(1, 6):
        start = lent1 + timedelta(days=(week - 1) * 7)
        for i, day in enumerate(["Sunday", "Monday", "Tuesday", "Wednesday",
                                  "Thursday", "Friday", "Saturday"]):
            dates.append({
                "period": f"{ordinal(week)} Week of Lent",
                "day_identifier": day,
                "special_name": None,
                "date": format_date(start + timedelta(days=i)),
                "source_year": year,
            })
    
    # ==========================================================================
    # HOLY WEEK
    # ==========================================================================
    palm = easter - timedelta(days=7)
    
    for offset, day, special in [
        (0, "Sunday", "Palm Sunday"),
        (1, "Monday", None),
        (2, "Tuesday", None),
        (3, "Wednesday", None),
        (4, "Thursday", "Maundy Thursday"),
        (5, "Friday", "Good Friday"),
        (6, "Saturday", "Holy Saturday"),
    ]:
        dates.append({
            "period": "Holy Week",
            "day_identifier": day,
            "special_name": special,
            "date": format_date(palm + timedelta(days=offset)),
            "source_year": year,
        })
    
    # ==========================================================================
    # EASTER WEEK
    # ==========================================================================
    for i, day in enumerate(["Sunday", "Monday", "Tuesday", "Wednesday",
                              "Thursday", "Friday", "Saturday"]):
        dates.append({
            "period": "Easter Week",
            "day_identifier": day,
            "special_name": "Easter Day" if i == 0 else None,
            "date": format_date(easter + timedelta(days=i)),
            "source_year": year,
        })
    
    # ==========================================================================
    # EASTER SEASON (Weeks 2-7)
    # ==========================================================================
    for week in range(2, 8):
        start = easter + timedelta(days=(week - 1) * 7)
        for i, day in enumerate(["Sunday", "Monday", "Tuesday", "Wednesday",
                                  "Thursday", "Friday", "Saturday"]):
            d = start + timedelta(days=i)
            special = None
            if d == ascension:
                special = "Ascension Day"
            dates.append({
                "period": f"{ordinal(week)} Week of Easter",
                "day_identifier": day,
                "special_name": special,
                "date": format_date(d),
                "source_year": year,
            })
    
    # ==========================================================================
    # PENTECOST & ORDINARY TIME
    # ==========================================================================
    dates.append({
        "period": "Pentecost",
        "day_identifier": "Sunday",
        "special_name": "Day of Pentecost",
        "date": format_date(pentecost),
        "source_year": year,
    })
    
    christ_king = next_advent - timedelta(days=7)
    trinity = pentecost + timedelta(days=7)
    
    current = pentecost + timedelta(days=1)
    week_num = 1
    
    while current < christ_king:
        special = None
        if current == trinity:
            period = "Trinity Sunday and Following"
            special = "Trinity Sunday"
        else:
            period = f"Week {week_num} after Pentecost"
        
        dates.append({
            "period": period,
            "day_identifier": day_name(current),
            "special_name": special,
            "date": format_date(current),
            "source_year": year,
        })
        
        current += timedelta(days=1)
        if current.weekday() == 6:
            week_num += 1
    
    dates.append({
        "period": "Christ the King",
        "day_identifier": "Sunday",
        "special_name": "Christ the King / Reign of Christ",
        "date": format_date(christ_king),
        "source_year": year,
    })
    
    return dates


def main():
    print("Analyzing dated weeks coverage...", file=sys.stderr)
    
    # Find which years to use for each dated week
    dated_week_sources = {"cycle_1": {}, "cycle_2": {}}
    
    for start_m, start_d, end_m, end_d, period_name in DATED_WEEK_RANGES:
        for cycle in [1, 2]:
            years = find_years_with_dated_week(start_m, start_d, end_m, end_d, cycle)
            key = f"cycle_{cycle}"
            # Prefer years >= 2025 for "freshness", but take any if needed
            preferred = [y for y in years if y >= 2025]
            if preferred:
                dated_week_sources[key][period_name] = preferred[0]
            elif years:
                dated_week_sources[key][period_name] = years[-1]  # Latest available
            else:
                dated_week_sources[key][period_name] = None
    
    print("Dated week sources:", file=sys.stderr)
    print(f"  Cycle 1: {dated_week_sources['cycle_1']}", file=sys.stderr)
    print(f"  Cycle 2: {dated_week_sources['cycle_2']}", file=sys.stderr)
    
    # Primary years for base content
    # Year 1 = cycle 1, Year 2 = cycle 2
    base_year_1 = 2027  # Closest cycle 1 year >= 2025
    base_year_2 = 2026  # Closest cycle 2 year >= 2025
    
    # Generate base dates
    print(f"Generating base Year 1 from {base_year_1}...", file=sys.stderr)
    year1_dates = generate_base_year_dates(base_year_1)
    
    print(f"Generating base Year 2 from {base_year_2}...", file=sys.stderr)
    year2_dates = generate_base_year_dates(base_year_2)
    
    # Add dated week dates from appropriate years
    for start_m, start_d, end_m, end_d, period_name in DATED_WEEK_RANGES:
        # Year 1
        source_year = dated_week_sources['cycle_1'].get(period_name)
        if source_year:
            print(f"  Year 1 {period_name}: from {source_year}", file=sys.stderr)
            year1_dates.extend(generate_dated_week_dates(
                source_year, start_m, start_d, end_m, end_d, period_name))
        
        # Year 2
        source_year = dated_week_sources['cycle_2'].get(period_name)
        if source_year:
            print(f"  Year 2 {period_name}: from {source_year}", file=sys.stderr)
            year2_dates.extend(generate_dated_week_dates(
                source_year, start_m, start_d, end_m, end_d, period_name))
    
    # Sort by date
    year1_dates.sort(key=lambda x: x['date'])
    year2_dates.sort(key=lambda x: x['date'])
    
    # Collect URLs (format: YYYY/MM/DD with slashes)
    all_urls = set()
    for d in year1_dates:
        date_with_slashes = d['date'].replace('-', '/')
        all_urls.add(f"https://pcusa.org/daily/devotion/{date_with_slashes}")
    for d in year2_dates:
        date_with_slashes = d['date'].replace('-', '/')
        all_urls.add(f"https://pcusa.org/daily/devotion/{date_with_slashes}")
    
    # Check for --urls-only
    if len(sys.argv) > 1 and sys.argv[1] == "--urls-only":
        for url in sorted(all_urls):
            print(url)
        return
    
    # Build output
    output = {
        "metadata": {
            "generated_at": datetime.now().isoformat(),
            "description": "Dates to scrape for the 2-year daily lectionary cycle (complete coverage)",
            "strategy": "Uses multiple years to ensure all dated weeks are covered",
            "year_1_sources": {
                "base_year": base_year_1,
                "dated_weeks": dated_week_sources['cycle_1'],
            },
            "year_2_sources": {
                "base_year": base_year_2,
                "dated_weeks": dated_week_sources['cycle_2'],
            },
            "year_1_date_count": len(year1_dates),
            "year_2_date_count": len(year2_dates),
            "unique_urls_to_scrape": len(all_urls),
        },
        "year_1_dates": year1_dates,
        "year_2_dates": year2_dates,
        "all_urls": sorted(all_urls),
    }
    
    print(json.dumps(output, indent=2))


if __name__ == "__main__":
    main()
