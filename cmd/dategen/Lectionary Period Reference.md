# Complete Lectionary Periods and Days Reference

This document lists ALL periods and day identifiers that should exist in your `lectionary_days` table
for the Daily Lectionary to work for any date.

## How to Use This Document

1. Query your database: `SELECT DISTINCT period, day_identifier FROM lectionary_days ORDER BY period, day_identifier`
2. Compare against this list
3. Any missing entries need to be parsed and imported

---

## ADVENT (4 weeks)

### 1st Week of Advent
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 2nd Week of Advent
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 3rd Week of Advent
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 4th Week of Advent
**Note**: Uses date-based identifiers because the week length varies based on when Christmas falls.

| Day Identifier | Period Type |
|----------------|-------------|
| December 17 | liturgical_week |
| December 18 | liturgical_week |
| December 19 | liturgical_week |
| December 20 | liturgical_week |
| December 21 | liturgical_week |
| December 22 | liturgical_week |
| December 23 | liturgical_week |
| December 24 | liturgical_week |

---

## CHRISTMAS SEASON

### Christmas
| Day Identifier | Period Type |
|----------------|-------------|
| December 25 | fixed_days |

### Christmas Season
| Day Identifier | Period Type |
|----------------|-------------|
| December 26 | fixed_days |
| December 27 | fixed_days |
| December 28 | fixed_days |
| December 29 | fixed_days |
| December 30 | fixed_days |
| December 31 | fixed_days |
| January 1 | fixed_days |
| January 2 | fixed_days |
| January 3 | fixed_days |
| January 4 | fixed_days |
| January 5 | fixed_days |

---

## EPIPHANY SEASON

### Epiphany and Following
| Day Identifier | Period Type |
|----------------|-------------|
| January 6 | fixed_days |
| January 7 | fixed_days |
| January 8 | fixed_days |
| January 9 | fixed_days |
| January 10 | fixed_days |
| January 11 | fixed_days |
| January 12 | fixed_days |

### Baptism of the Lord
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |

### Week 1 after Baptism of the Lord
| Day Identifier | Period Type |
|----------------|-------------|
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### Week 2 after Baptism of the Lord
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### Week 3 after Baptism of the Lord
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### Week 4 after Baptism of the Lord
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### Week 5 after Baptism of the Lord
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### Week 6 after Baptism of the Lord
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### Week 7 after Baptism of the Lord
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### Week 8 after Baptism of the Lord
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### Dated Weeks (Epiphany to Lent transition)
These are the variable weeks that depend on when Easter falls.

**Week following Sun. between Feb. 4 and 10**
| Day Identifier | Period Type |
|----------------|-------------|
| Monday | dated_week |
| Tuesday | dated_week |
| Wednesday | dated_week |
| Thursday | dated_week |
| Friday | dated_week |
| Saturday | dated_week |

**Week following Sun. between Feb. 11 and 17**
| Day Identifier | Period Type |
|----------------|-------------|
| Monday | dated_week |
| Tuesday | dated_week |
| Wednesday | dated_week |
| Thursday | dated_week |
| Friday | dated_week |
| Saturday | dated_week |

**Week following Sun. between Feb. 18 and 24**
| Day Identifier | Period Type |
|----------------|-------------|
| Monday | dated_week |
| Tuesday | dated_week |
| Wednesday | dated_week |
| Thursday | dated_week |
| Friday | dated_week |
| Saturday | dated_week |

**Week following Sun. between Feb. 25 and 29**
| Day Identifier | Period Type |
|----------------|-------------|
| Monday | dated_week |
| Tuesday | dated_week |
| Wednesday | dated_week |
| Thursday | dated_week |
| Friday | dated_week |
| Saturday | dated_week |

**Week following Sun. between Mar. 1 and 7**
| Day Identifier | Period Type |
|----------------|-------------|
| Monday | dated_week |
| Tuesday | dated_week |
| Wednesday | dated_week |
| Thursday | dated_week |
| Friday | dated_week |
| Saturday | dated_week |

---

## LENT (6 weeks + Ash Wednesday)

### Ash Wednesday and Following
| Day Identifier | Period Type |
|----------------|-------------|
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 1st Week of Lent
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 2nd Week of Lent
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 3rd Week of Lent
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 4th Week of Lent
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 5th Week of Lent
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 6th Week of Lent
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

---

## HOLY WEEK

### Holy Week
| Day Identifier | Special Name | Period Type |
|----------------|--------------|-------------|
| Sunday | Palm Sunday | liturgical_week |
| Monday | Monday of Holy Week | liturgical_week |
| Tuesday | Tuesday of Holy Week | liturgical_week |
| Wednesday | Wednesday of Holy Week | liturgical_week |
| Thursday | Maundy Thursday | liturgical_week |
| Friday | Good Friday | liturgical_week |
| Saturday | Holy Saturday | liturgical_week |

---

## EASTER SEASON (7 weeks)

### 1st Week of Easter
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 2nd Week of Easter
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 3rd Week of Easter
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 4th Week of Easter
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 5th Week of Easter
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 6th Week of Easter
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### 7th Week of Easter
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

---

## PENTECOST AND ORDINARY TIME

### Pentecost
| Day Identifier | Period Type |
|----------------|-------------|
| Sunday | liturgical_week |

### Week 1 after Pentecost
| Day Identifier | Period Type |
|----------------|-------------|
| Monday | liturgical_week |
| Tuesday | liturgical_week |
| Wednesday | liturgical_week |
| Thursday | liturgical_week |
| Friday | liturgical_week |
| Saturday | liturgical_week |

### Weeks 2-27 after Pentecost
**Pattern**: Each week needs Sunday through Saturday (7 days)

The exact number of weeks varies by year (22-27 weeks typically), but you should have data for up to Week 27 to handle all years.

| Period | Days Needed |
|--------|-------------|
| Week 2 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 3 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 4 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 5 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 6 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 7 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 8 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 9 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 10 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 11 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 12 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 13 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 14 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 15 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 16 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 17 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 18 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 19 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 20 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 21 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 22 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 23 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 24 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 25 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 26 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |
| Week 27 after Pentecost | Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday |

---

## SUMMARY: Total Expected Entries

| Season | Periods | Days per Period | Total Days |
|--------|---------|-----------------|------------|
| Advent (weeks 1-3) | 3 | 7 | 21 |
| Advent (week 4) | 1 | 8 | 8 |
| Christmas | 1 | 1 | 1 |
| Christmas Season | 1 | 11 | 11 |
| Epiphany and Following | 1 | 7 | 7 |
| Baptism of the Lord | 1 | 1 | 1 |
| Weeks after Baptism (1-8) | 8 | 6-7 | ~55 |
| Dated Weeks (5 periods) | 5 | 6 | 30 |
| Ash Wednesday and Following | 1 | 4 | 4 |
| Lent (weeks 1-6) | 6 | 7 | 42 |
| Holy Week | 1 | 7 | 7 |
| Easter (weeks 1-7) | 7 | 7 | 49 |
| Pentecost | 1 | 1 | 1 |
| Weeks after Pentecost (1-27) | 27 | 6-7 | ~185 |
| **TOTAL** | | | **~422 entries** |

---

## SQL Query to Check Your Data

Run this to see what you have vs what you need:

```sql
-- Count entries by period pattern
SELECT 
    CASE 
        WHEN period LIKE '%Advent%' THEN 'Advent'
        WHEN period LIKE '%Christmas%' THEN 'Christmas'
        WHEN period LIKE '%Epiphany%' THEN 'Epiphany'
        WHEN period LIKE '%Baptism%' THEN 'Baptism'
        WHEN period LIKE '%after Baptism%' THEN 'After Baptism'
        WHEN period LIKE '%Week following%' THEN 'Dated Weeks'
        WHEN period LIKE '%Ash Wednesday%' THEN 'Ash Wednesday'
        WHEN period LIKE '%Lent%' THEN 'Lent'
        WHEN period LIKE '%Holy Week%' THEN 'Holy Week'
        WHEN period LIKE '%Easter%' THEN 'Easter'
        WHEN period LIKE '%Pentecost%' THEN 'Pentecost'
        WHEN period LIKE '%after Pentecost%' THEN 'After Pentecost'
        ELSE 'Other'
    END as season,
    COUNT(*) as entry_count
FROM lectionary_days
GROUP BY season
ORDER BY 
    CASE season
        WHEN 'Advent' THEN 1
        WHEN 'Christmas' THEN 2
        WHEN 'Epiphany' THEN 3
        WHEN 'Baptism' THEN 4
        WHEN 'After Baptism' THEN 5
        WHEN 'Dated Weeks' THEN 6
        WHEN 'Ash Wednesday' THEN 7
        WHEN 'Lent' THEN 8
        WHEN 'Holy Week' THEN 9
        WHEN 'Easter' THEN 10
        WHEN 'Pentecost' THEN 11
        WHEN 'After Pentecost' THEN 12
        ELSE 99
    END;
```

```sql
-- List all unique periods
SELECT DISTINCT period FROM lectionary_days ORDER BY period;
```

```sql
-- Find periods with incomplete days
SELECT period, COUNT(*) as day_count
FROM lectionary_days
GROUP BY period
HAVING day_count < 6  -- Most periods should have 6-7 days
ORDER BY day_count;
```