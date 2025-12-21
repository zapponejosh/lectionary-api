# Lectionary PDF Parsing Specification

## Purpose

This document specifies how to parse the Presbyterian Daily Lectionary from image files into structured JSON for import into a Go/SQLite application.

## Source Material

**File**: `Lectionaryfull.pdf` (actually a ZIP archive of JPEG images)
**Source**: Book of Common Worship, Westminster John Knox Press, 2018
**Content**: 67 JPEG images extracted to a folder

### Page Map

| Pages | Content | Action |
|-------|---------|--------|
| 1-2 | Title page, copyright | SKIP |
| 3-4 | Introduction text | SKIP |
| 5-19 | Sunday/Festival Lectionary (3-year cycle: A, B, C) | PARSE |
| 20-65 | Daily Lectionary (2-year cycle: Year 1, Year 2) | PARSE |
| 66-67 | Liturgical Calendar Table (1992-2040) | PARSE |

---

## Output Schema

### Master Output File: `lectionary_data.json`

```json
{
  "metadata": {
    "source": "Book of Common Worship, Westminster John Knox Press, 2018",
    "parsed_date": "YYYY-MM-DD",
    "schema_version": "1.0"
  },
  "liturgical_calendar": [...],
  "sunday_lectionary": [...],
  "daily_lectionary": [...]
}
```

---

## Schema Definitions

### 1. Liturgical Calendar Entry

Parsed from pages 66-67. Maps civil years to liturgical cycles and key dates.

```json
{
  "year": 2025,
  "sunday_cycle": "C",
  "daily_cycle": 1,
  "first_sunday_of_advent": "2024-12-01",
  "ash_wednesday": "2025-03-05",
  "easter": "2025-04-20",
  "ascension": "2025-05-29",
  "pentecost": "2025-06-08"
}
```

**Field Definitions:**
- `year`: Civil year (integer)
- `sunday_cycle`: "A", "B", or "C" — the 3-year Sunday lectionary cycle
- `daily_cycle`: 1 or 2 — the 2-year daily lectionary cycle
- `first_sunday_of_advent`: ISO date string (note: this is when the liturgical year STARTS, often in previous civil year)
- `ash_wednesday`: ISO date string
- `easter`: ISO date string
- `ascension`: ISO date string
- `pentecost`: ISO date string

**Parsing Notes:**
- The calendar table shows dates across columns
- Sunday cycle follows pattern: A, B, C, A, B, C...
- Daily cycle alternates: 1, 2, 1, 2...
- Extract all years from 1992 to 2040

---

### 2. Sunday Lectionary Entry

Parsed from pages 5-19. The 3-year Revised Common Lectionary for Sundays and festivals.

```json
{
  "occasion": "1st Sunday of Advent",
  "season": "Advent",
  "fixed_date": null,
  "year_a": {
    "readings": [
      {
        "position": 1,
        "label": "First Reading",
        "references": ["Isa. 2:1-5"]
      },
      {
        "position": 2,
        "label": "Psalm",
        "references": ["Ps. 122"]
      },
      {
        "position": 3,
        "label": "Second Reading",
        "references": ["Rom. 13:11-14"]
      },
      {
        "position": 4,
        "label": "Gospel",
        "references": ["Matt. 24:36-44"]
      }
    ]
  },
  "year_b": {
    "readings": [...]
  },
  "year_c": {
    "readings": [...]
  }
}
```

**Field Definitions:**
- `occasion`: Name of the Sunday or festival (string)
- `season`: Liturgical season — one of: "Advent", "Christmas", "Epiphany", "Lent", "Easter", "Pentecost", "Ordinary Time"
- `fixed_date`: For fixed festivals like "Christmas Day (Dec. 25)", store "December 25". Otherwise null.
- `year_a`, `year_b`, `year_c`: Readings for each year of the 3-year cycle

**Reading Object:**
- `position`: Integer 1-4 (order in service)
- `label`: One of "First Reading", "Psalm", "Second Reading", "Gospel"
- `references`: Array of scripture references (array to handle alternatives)

**Note on Alternatives:**
When the source shows `"Luke 1:47-55 or Ps. 80:1-7"`, store as:
```json
{
  "position": 2,
  "label": "Psalm",
  "references": ["Luke 1:47-55", "Ps. 80:1-7"],
  "is_alternative": true
}
```

---

### 3. Daily Lectionary Entry

Parsed from pages 20-65. The 2-year daily lectionary.

```json
{
  "period": "1st Week of Advent",
  "period_type": "liturgical_week",
  "week_position": null,
  "entries": [
    {
      "day": "Sunday",
      "fixed_date": null,
      "special_name": null,
      "psalms": {
        "morning": ["24", "150"],
        "evening": ["25", "110"]
      },
      "year_1": {
        "readings": [
          {
            "position": 1,
            "label": "First Reading",
            "references": ["Isa. 1:1-9"]
          },
          {
            "position": 2,
            "label": "Second Reading",
            "references": ["2 Peter 3:1-10"]
          },
          {
            "position": 3,
            "label": "Gospel",
            "references": ["Matt. 25:1-13"]
          }
        ]
      },
      "year_2": {
        "readings": [...]
      }
    },
    {
      "day": "Monday",
      ...
    }
  ]
}
```

**Period Types:**

1. **`liturgical_week`**: Standard week like "1st Week of Advent"
   ```json
   {
     "period": "1st Week of Advent",
     "period_type": "liturgical_week",
     "week_position": null
   }
   ```

2. **`dated_week`**: Week defined by date range like "Week following Sun. between Feb. 11 and 17"
   ```json
   {
     "period": "Week following Sun. between Feb. 11 and 17",
     "period_type": "dated_week",
     "week_position": {
       "anchor": "sunday_between",
       "start_month_day": "02-11",
       "end_month_day": "02-17",
       "exception": "except when this Sunday is Transfiguration"
     }
   }
   ```

3. **`fixed_days`**: Individual fixed-date days like the days after Epiphany
   ```json
   {
     "period": "Days after Epiphany",
     "period_type": "fixed_days",
     "week_position": null
   }
   ```

**Day Entry Fields:**
- `day`: Day of week ("Sunday", "Monday", etc.) OR null for fixed-date entries
- `fixed_date`: "January 7", "December 24", etc. OR null for day-of-week entries
- `special_name`: "Eve of Baptism of the Lord", "Ash Wednesday", etc. OR null
- `psalms`: Object with `morning` and `evening` arrays
- `year_1`, `year_2`: Reading objects for each year cycle

**Psalm Parsing Rules:**

1. Simple list: `"Morning: Ps. 24; 150"` → `["24", "150"]`

2. Verse ranges: `"Morning: Ps. 18:1-20; 147:12-20"` → `["18:1-20", "147:12-20"]`

3. Alternatives: `"Morning: Ps. 46 or 97"` → `["46", "97"]` with note that these are alternatives

4. Laudate Psalm footnote: When you see `"Laudate Psalm *"`, expand using this mapping:
   - Sunday → Ps. 150
   - Monday → Ps. 145
   - Tuesday → Ps. 146
   - Wednesday → Ps. 147:1-11
   - Thursday → Ps. 147:12-20
   - Friday → Ps. 148
   - Saturday → Ps. 149

---

## Scripture Reference Format

**Keep references as they appear in the source.** Do not normalize. Examples:

| Source Text | Store As |
|-------------|----------|
| `Isa. 1:1-9` | `"Isa. 1:1-9"` |
| `1 Thess. 2:1-12` | `"1 Thess. 2:1-12"` |
| `Matt. 21:1-11` | `"Matt. 21:1-11"` |
| `Ps. 147:1-11` | `"147:1-11"` (for psalms, omit "Ps.") |
| `Gen. 2:4-9 (10-15) 16-25` | `"Gen. 2:4-9 (10-15) 16-25"` |
| `John 9:1-12, 35-38` | `"John 9:1-12, 35-38"` |
| `Luke 20:41-21:4` | `"Luke 20:41-21:4"` |
| `Wisd. of Sol. 3:1-9` | `"Wisd. of Sol. 3:1-9"` |

**Parenthetical verses** like `(10-15)` indicate optional extensions. Keep them in the string.

---

## Edge Cases and Special Handling

### 1. Alternative Readings

When source shows "X or Y":
```json
{
  "references": ["Wisd. of Sol. 7:1-14", "Jer. 31:23-25"],
  "is_alternative": true
}
```

### 2. Fixed Date Days Within a Week

Some periods mix day-of-week and fixed-date entries. Example from Epiphany season:

```
January 7    | Ps. 46 or 97 | Isa. 52:3-6   | Deut. 8:1-3
January 8    | Ps. 46 or 47 | Isa. 59:15b-21| Ex. 17:1-7
...
Eve of Baptism | Ps. 27; 93; or 114 | Isa. 61:1-9 | Isa. 61:1-9
```

Store fixed dates in `fixed_date` field, leave `day` as null.

### 3. Special Named Days

Days like "Ash Wednesday", "Maundy Thursday", "Eve of Baptism of the Lord" get stored in `special_name`.

### 4. Transfiguration Exception

Several weeks have notes like "except when this Sunday is Transfiguration". Store in `week_position.exception`.

### 5. Same Readings Both Years

Some entries (especially fixed festivals) have identical readings for Year 1 and Year 2. Still store both explicitly.

### 6. Multi-line References

Some cells span multiple lines:
```
Wisd. of Sol.
  1:16-2:11, 21-24
  or Jer. 30:1-9
```
Combine into single reference string: `"Wisd. of Sol. 1:16-2:11, 21-24"` and `"Jer. 30:1-9"` as alternatives.

---

## Liturgical Periods Reference

For the `period` field, use these exact strings as they appear in the document:

**Advent & Christmas:**
- 1st Week of Advent
- 2nd Week of Advent
- 3rd Week of Advent
- 4th Week of Advent
- Christmas Eve and Day
- 1st Sunday after Christmas Day
- 2nd Sunday after Christmas

**Epiphany:**
- Eve of Epiphany
- Epiphany and Following
- Baptism of the Lord (Sunday between Jan. 7 and 13 inclusive) and Following
- 2nd Sunday after Epiphany and Following
- [continues with "Week following Sun. between [dates]" pattern]

**Lent:**
- Ash Wednesday and Following
- 1st Week of Lent
- 2nd Week of Lent
- 3rd Week of Lent
- 4th Week of Lent
- 5th Week of Lent
- Holy Week

**Easter:**
- Easter Week
- 2nd Week of Easter
- 3rd Week of Easter
- 4th Week of Easter
- 5th Week of Easter
- 6th Week of Easter
- 7th Week of Easter

**Pentecost & Ordinary Time:**
- Day of Pentecost
- Trinity Sunday and Following
- [Weeks defined by date ranges]
- Proper 1 through Proper 29
- Christ the King / Reign of Christ

---

## Validation Checklist

After parsing, verify:

1. **Calendar table**: All years 1992-2040 present with all date fields
2. **Sunday lectionary**: All major Sundays/festivals covered, all three years populated
3. **Daily lectionary**: 
   - Every liturgical week has 7 day entries
   - Every entry has morning and evening psalms
   - Every entry has 3 readings for both Year 1 and Year 2
4. **No null references**: Every reading position should have at least one reference
5. **Cross-reference spot check**: Compare a few entries against the 2025 Daily PDF to verify accuracy

---

## Output Instructions

1. Parse all pages in order
2. Combine into single `lectionary_data.json` file
3. Validate against checklist above
4. Report any anomalies or ambiguities encountered

The output JSON will be used by a Go application to seed a SQLite database. The Go code will handle:
- Normalizing scripture references (optional)
- Mapping liturgical positions to calendar dates
- Creating database records with proper foreign keys

---

## Sample Complete Entry (for reference)

Here's a fully parsed example from page 20:

```json
{
  "period": "1st Week of Advent",
  "period_type": "liturgical_week",
  "week_position": null,
  "entries": [
    {
      "day": "Sunday",
      "fixed_date": null,
      "special_name": null,
      "psalms": {
        "morning": ["24", "150"],
        "evening": ["25", "110"]
      },
      "year_1": {
        "readings": [
          {
            "position": 1,
            "label": "First Reading",
            "references": ["Isa. 1:1-9"]
          },
          {
            "position": 2,
            "label": "Second Reading",
            "references": ["2 Peter 3:1-10"]
          },
          {
            "position": 3,
            "label": "Gospel",
            "references": ["Matt. 25:1-13"]
          }
        ]
      },
      "year_2": {
        "readings": [
          {
            "position": 1,
            "label": "First Reading",
            "references": ["Amos 1:1-5, 13-2:8"]
          },
          {
            "position": 2,
            "label": "Second Reading",
            "references": ["1 Thess. 5:1-11"]
          },
          {
            "position": 3,
            "label": "Gospel",
            "references": ["Luke 21:5-19"]
          }
        ]
      }
    },
    {
      "day": "Monday",
      "fixed_date": null,
      "special_name": null,
      "psalms": {
        "morning": ["122", "145"],
        "evening": ["40", "67"]
      },
      "year_1": {
        "readings": [
          {
            "position": 1,
            "label": "First Reading",
            "references": ["Isa. 1:10-20"]
          },
          {
            "position": 2,
            "label": "Second Reading",
            "references": ["1 Thess. 1:1-10"]
          },
          {
            "position": 3,
            "label": "Gospel",
            "references": ["Luke 20:1-8"]
          }
        ]
      },
      "year_2": {
        "readings": [
          {
            "position": 1,
            "label": "First Reading",
            "references": ["Amos 2:6-16"]
          },
          {
            "position": 2,
            "label": "Second Reading",
            "references": ["2 Peter 1:1-11"]
          },
          {
            "position": 3,
            "label": "Gospel",
            "references": ["Matt. 21:1-11"]
          }
        ]
      }
    }
  ]
}
```
