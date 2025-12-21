#!/usr/bin/env python3
"""
Lectionary Data Merge Script

Merges:
- scraping_plan_v4.json (structural mapping: period + day_identifier → date)
- scrape_progress.json (raw readings by date)

Outputs:
- daily_lectionary.json (position-based lectionary with Year 1 and Year 2 readings)

Usage:
    python3 merge_lectionary.py --plan PATH --readings PATH --output PATH
    python3 merge_lectionary.py  # Uses default paths
"""

import argparse
import json
import logging
import re
import sys
from datetime import datetime
from pathlib import Path
from typing import Optional

# =============================================================================
# LOGGING SETUP
# =============================================================================

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    datefmt='%H:%M:%S'
)
logger = logging.getLogger(__name__)

# Separate logger for edge cases we want to review
edge_case_logger = logging.getLogger('edge_cases')
edge_case_handler = logging.FileHandler('merge_edge_cases.log', mode='w')
edge_case_handler.setFormatter(logging.Formatter('%(message)s'))
edge_case_logger.addHandler(edge_case_handler)
edge_case_logger.setLevel(logging.WARNING)

# =============================================================================
# BOOK NAME PATTERNS
# =============================================================================

# Regex pattern to detect if a string starts with a book name
# Handles: "Genesis", "1 John", "2 Kings", "Song of Solomon", etc.
BOOK_PATTERN = re.compile(
    r'^('
    r'Genesis|Exodus|Leviticus|Numbers|Deuteronomy|'
    r'Joshua|Judges|Ruth|'
    r'1 Samuel|2 Samuel|1 Kings|2 Kings|'
    r'1 Chronicles|2 Chronicles|'
    r'Ezra|Nehemiah|Esther|Job|'
    r'Psalms?|Proverbs?|Ecclesiastes|'
    r'Song of Solomon|Song of Songs|Canticles|'
    r'Isaiah|Jeremiah|Lamentations|Ezekiel|Daniel|'
    r'Hosea|Joel|Amos|Obadiah|Jonah|Micah|'
    r'Nahum|Habakkuk|Zephaniah|Haggai|Zechariah|Malachi|'
    r'Matthew|Mark|Luke|John|Acts|'
    r'Romans|1 Corinthians|2 Corinthians|'
    r'Galatians|Ephesians|Philippians|Colossians|'
    r'1 Thessalonians|2 Thessalonians|'
    r'1 Timothy|2 Timothy|Titus|Philemon|'
    r'Hebrews|James|1 Peter|2 Peter|'
    r'1 John|2 John|3 John|Jude|Revelation|'
    # Abbreviated forms
    r'Gen\.|Exod?\.|Lev\.|Num\.|Deut\.|'
    r'Josh\.|Judg\.|'
    r'1 Sam\.|2 Sam\.|1 Kgs\.|2 Kgs\.|'
    r'1 Chr\.|2 Chr\.|'
    r'Neh\.|Esth\.|'
    r'Pss?\.|Prov\.|Eccl\.|Eccles\.|'
    r'Song\.|Cant\.|'
    r'Isa\.|Jer\.|Lam\.|Ezek\.|Dan\.|'
    r'Hos\.|Obad\.|Jon\.|Mic\.|'
    r'Nah\.|Hab\.|Zeph\.|Hag\.|Zech\.|Mal\.|'
    r'Matt\.|Mt\.|Mk\.|Lk\.|Jn\.|'
    r'Rom\.|1 Cor\.|2 Cor\.|'
    r'Gal\.|Eph\.|Phil\.|Col\.|'
    r'1 Thess\.|2 Thess\.|'
    r'1 Tim\.|2 Tim\.|'
    r'Heb\.|Jas\.|1 Pet\.|2 Pet\.|'
    r'1 Jn\.|2 Jn\.|3 Jn\.|Rev\.|'
    # Apocrypha (sometimes included)
    r'Wisd\. of Sol\.|Wisdom|Sirach|Ecclesiasticus|'
    r'Baruch|1 Maccabees|2 Maccabees|'
    r'Tobit|Judith'
    r')'
    r'\.?\s*[\d(]',  # Followed by optional period and a number or opening paren
    re.IGNORECASE
)

# Pattern to extract book name from beginning of reference
# Handles: "Genesis", "1 John", "Song of Solomon", "Wisd. of Sol.", etc.
BOOK_EXTRACT_PATTERN = re.compile(
    r'^((?:\d\s+)?'  # Optional leading number (1, 2, 3)
    r'(?:'
    r'Song of Solomon|Song of Songs|Wisd\. of Sol\.|Wisdom of Solomon|'  # Multi-word books first
    r'[A-Za-z]+(?:\.[A-Za-z]*)?'  # Single word books with optional abbreviation
    r')'
    r'\.?)\s*'  # Optional trailing period
    r'(\d.*)$'  # The chapter:verse part
)

# Months for detecting fixed dates
MONTHS = [
    "January", "February", "March", "April", "May", "June",
    "July", "August", "September", "October", "November", "December"
]

# Known typos/issues in source data to correct
SOURCE_CORRECTIONS = {
    "Kings ": "1 Kings ",  # Missing number prefix
    "Lamentation ": "Lamentations ",  # Singular vs plural
}


def correct_source_typos(raw: str) -> str:
    """Apply known corrections to source data typos."""
    for wrong, right in SOURCE_CORRECTIONS.items():
        if raw.startswith(wrong):
            raw = right + raw[len(wrong):]
    return raw

# =============================================================================
# REFERENCE SPLITTING
# =============================================================================

def starts_with_book_name(text: str) -> bool:
    """Check if text starts with a Bible book name."""
    text = text.strip()
    return bool(BOOK_PATTERN.match(text))


def extract_book_name(reference: str) -> Optional[str]:
    """Extract the book name from a reference like 'Isaiah 45:14-19'."""
    reference = reference.strip()
    match = BOOK_EXTRACT_PATTERN.match(reference)
    if match:
        return match.group(1).strip()
    return None


def split_references(raw: str, reading_type: str) -> list[str]:
    """
    Split a raw reference string into individual references.
    
    Rules:
    1. Semicolons always split (within same book, carry book name forward)
    2. Commas split only if followed by a book name
    3. Fragments inherit book name from previous reference
    
    Args:
        raw: Raw reference string like "Isaiah 45:14-19; 61:1-9"
        reading_type: For logging context (e.g., "first", "gospel")
    
    Returns:
        List of individual references
    """
    if not raw or raw.strip() == "":
        return []
    
    raw = raw.strip()
    
    # Apply known source data corrections
    raw = correct_source_typos(raw)
    
    results = []
    current_book = None
    
    # First, split on semicolons
    semi_parts = [p.strip() for p in raw.split(';') if p.strip()]
    
    for semi_part in semi_parts:
        # Now handle comma splits within each semicolon-separated part
        # But be careful: "John 9:1-12, 35-38" should NOT split
        
        # Split on comma and analyze each piece
        comma_parts = [p.strip() for p in semi_part.split(',')]
        
        i = 0
        while i < len(comma_parts):
            part = comma_parts[i].strip()
            
            if not part:
                i += 1
                continue
            
            if starts_with_book_name(part):
                # This part has its own book name
                current_book = extract_book_name(part)
                results.append(part)
            elif current_book:
                # Check if this looks like a verse fragment (just numbers, possibly with letter suffixes)
                # e.g., "35-38" or "23b-27" or "24-28(b)" after "John 9:1-12"
                if re.match(r'^\d+[a-z]?[:\-\d\s()a-z]*$', part, re.IGNORECASE):
                    # This is a verse continuation, append to previous
                    if results:
                        # Check if it's really a continuation (same chapter context)
                        # or a separate chapter reference
                        if ':' in part:
                            # Has chapter:verse, it's a new reference in same book
                            results.append(f"{current_book} {part}")
                        else:
                            # Just verses, continuation of previous
                            results[-1] = f"{results[-1]}, {part}"
                    else:
                        # No previous, treat as standalone with book
                        results.append(f"{current_book} {part}")
                else:
                    # Doesn't look like verse numbers, might be an issue
                    edge_case_logger.warning(
                        f"COMMA_SPLIT_AMBIGUOUS: '{raw}' -> part '{part}' "
                        f"(type: {reading_type})"
                    )
                    results.append(f"{current_book} {part}")
            else:
                # No current book and doesn't start with one - edge case
                edge_case_logger.warning(
                    f"NO_BOOK_CONTEXT: '{raw}' -> part '{part}' "
                    f"(type: {reading_type})"
                )
                results.append(part)
            
            i += 1
    
    # Validate results
    if len(results) == 0 and raw:
        edge_case_logger.warning(f"EMPTY_RESULT: '{raw}' produced no references (type: {reading_type})")
        results = [raw]  # Fallback to original
    
    return results


def split_psalm_references(raw: str) -> list[str]:
    """
    Split psalm references and strip 'Psalm' prefix.
    
    "Psalm 122; 145" -> ["122", "145"]
    "Psalm 46; 97; 147:1-11" -> ["46", "97", "147:1-11"]
    """
    if not raw or raw.strip() == "":
        return []
    
    raw = raw.strip()
    
    # Remove "Psalm " or "Psalms " prefix
    raw = re.sub(r'^Psalms?\s*', '', raw, flags=re.IGNORECASE)
    
    # Split on semicolons
    parts = [p.strip() for p in raw.split(';') if p.strip()]
    
    results = []
    for part in parts:
        # Handle comma-separated psalm numbers (rare but possible)
        # Usually semicolons are used between psalms
        results.append(part)
    
    return results


# =============================================================================
# PERIOD TYPE INFERENCE
# =============================================================================

def infer_period_type(period: str, day_identifier: str) -> str:
    """
    Infer the period type from period name and day identifier.
    
    Returns: "liturgical_week", "dated_week", or "fixed_days"
    """
    # Dated weeks have a specific pattern
    if "following Sun. between" in period:
        return "dated_week"
    
    # Fixed date periods (day_identifier is a date like "January 7")
    if day_identifier:
        for month in MONTHS:
            if month in day_identifier:
                return "fixed_days"
    
    # Everything else is liturgical_week (includes moveable feasts)
    return "liturgical_week"


# =============================================================================
# DATA LOADING
# =============================================================================

def load_json(path: Path) -> dict:
    """Load JSON file with error handling."""
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return json.load(f)
    except FileNotFoundError:
        logger.error(f"File not found: {path}")
        sys.exit(1)
    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON in {path}: {e}")
        sys.exit(1)


# =============================================================================
# MERGE LOGIC
# =============================================================================

def create_position_key(period: str, day_identifier: str) -> str:
    """Create a unique key for a lectionary position."""
    return f"{period}|{day_identifier}"


def merge_readings(
    plan_entry: dict,
    scraped_readings: dict,
    year_num: int
) -> Optional[dict]:
    """
    Merge a single plan entry with its scraped readings.
    
    Args:
        plan_entry: Entry from scraping_plan (period, day_identifier, date, etc.)
        scraped_readings: Dict of date -> scraped data
        year_num: 1 or 2 (for logging context)
    
    Returns:
        Dict with readings array, or None if no data found
    """
    date = plan_entry["date"]
    
    if date not in scraped_readings:
        edge_case_logger.warning(
            f"MISSING_DATE: Year {year_num}, {plan_entry['period']} / "
            f"{plan_entry['day_identifier']} -> date {date} not in scraped data"
        )
        return None
    
    scraped = scraped_readings[date]
    raw_readings = scraped.get("readings", {})
    
    readings = []
    
    # First Reading
    if raw_readings.get("First Reading"):
        refs = split_references(raw_readings["First Reading"], "first")
        readings.append({
            "position": 1,
            "label": "first",
            "references": refs
        })
    
    # Second Reading
    if raw_readings.get("Second Reading"):
        refs = split_references(raw_readings["Second Reading"], "second")
        readings.append({
            "position": 2,
            "label": "second",
            "references": refs
        })
    
    # Gospel
    if raw_readings.get("Gospel"):
        refs = split_references(raw_readings["Gospel"], "gospel")
        readings.append({
            "position": 3,
            "label": "gospel",
            "references": refs
        })
    
    if not readings:
        edge_case_logger.warning(
            f"NO_READINGS: Year {year_num}, {plan_entry['period']} / "
            f"{plan_entry['day_identifier']} -> date {date} has no readings"
        )
        return None
    
    return {"readings": readings}


def merge_psalms(scraped_readings: dict, date: str) -> dict:
    """Extract and split psalm references from scraped data."""
    psalms = {"morning": [], "evening": []}
    
    if date not in scraped_readings:
        return psalms
    
    raw_readings = scraped_readings[date].get("readings", {})
    
    if raw_readings.get("Morning"):
        psalms["morning"] = split_psalm_references(raw_readings["Morning"])
    
    if raw_readings.get("Evening"):
        psalms["evening"] = split_psalm_references(raw_readings["Evening"])
    
    return psalms


def merge_lectionary(plan: dict, progress: dict) -> dict:
    """
    Main merge function.
    
    Combines Year 1 and Year 2 data into position-based entries.
    """
    # Extract scraped readings (from progress file's "completed" dict)
    scraped_readings = progress.get("completed", {})
    
    logger.info(f"Loaded {len(scraped_readings)} scraped dates")
    logger.info(f"Year 1 entries in plan: {len(plan.get('year_1_dates', []))}")
    logger.info(f"Year 2 entries in plan: {len(plan.get('year_2_dates', []))}")
    
    # Build lookup structures
    # Key: position_key -> {year_1: readings, year_2: readings, psalms: {...}}
    positions = {}
    
    # Process Year 1
    logger.info("Processing Year 1...")
    for entry in plan.get("year_1_dates", []):
        pos_key = create_position_key(entry["period"], entry["day_identifier"])
        
        if pos_key not in positions:
            positions[pos_key] = {
                "period": entry["period"],
                "day_identifier": entry["day_identifier"],
                "special_name": entry.get("special_name"),
                "period_type": infer_period_type(entry["period"], entry["day_identifier"]),
                "psalms": merge_psalms(scraped_readings, entry["date"]),
                "year_1": None,
                "year_2": None,
                "_year_1_date": entry["date"],  # For debugging
                "_year_2_date": None,
            }
        
        readings = merge_readings(entry, scraped_readings, 1)
        if readings:
            positions[pos_key]["year_1"] = readings
            positions[pos_key]["_year_1_date"] = entry["date"]
    
    # Process Year 2
    logger.info("Processing Year 2...")
    for entry in plan.get("year_2_dates", []):
        pos_key = create_position_key(entry["period"], entry["day_identifier"])
        
        if pos_key not in positions:
            positions[pos_key] = {
                "period": entry["period"],
                "day_identifier": entry["day_identifier"],
                "special_name": entry.get("special_name"),
                "period_type": infer_period_type(entry["period"], entry["day_identifier"]),
                "psalms": {"morning": [], "evening": []},
                "year_1": None,
                "year_2": None,
                "_year_1_date": None,
                "_year_2_date": entry["date"],
            }
        
        readings = merge_readings(entry, scraped_readings, 2)
        if readings:
            positions[pos_key]["year_2"] = readings
            positions[pos_key]["_year_2_date"] = entry["date"]
        
        # If we don't have psalms yet, get them from Year 2 date
        if not positions[pos_key]["psalms"]["morning"] and not positions[pos_key]["psalms"]["evening"]:
            positions[pos_key]["psalms"] = merge_psalms(scraped_readings, entry["date"])
    
    # Convert to list and validate
    logger.info("Validating merged data...")
    daily_lectionary = []
    
    missing_year_1 = 0
    missing_year_2 = 0
    missing_both = 0
    
    for pos_key, data in positions.items():
        # Remove debug fields for final output
        year_1_date = data.pop("_year_1_date", None)
        year_2_date = data.pop("_year_2_date", None)
        
        # Check for missing data
        if data["year_1"] is None and data["year_2"] is None:
            missing_both += 1
            edge_case_logger.warning(
                f"MISSING_BOTH_YEARS: {data['period']} / {data['day_identifier']}"
            )
        elif data["year_1"] is None:
            missing_year_1 += 1
            edge_case_logger.warning(
                f"MISSING_YEAR_1: {data['period']} / {data['day_identifier']} "
                f"(year_2_date: {year_2_date})"
            )
        elif data["year_2"] is None:
            missing_year_2 += 1
            edge_case_logger.warning(
                f"MISSING_YEAR_2: {data['period']} / {data['day_identifier']} "
                f"(year_1_date: {year_1_date})"
            )
        
        daily_lectionary.append(data)
    
    # Log summary
    logger.info(f"Total positions: {len(daily_lectionary)}")
    logger.info(f"Missing Year 1 only: {missing_year_1}")
    logger.info(f"Missing Year 2 only: {missing_year_2}")
    logger.info(f"Missing both years: {missing_both}")
    
    return {
        "metadata": {
            "generated_at": datetime.now().isoformat(),
            "source": "PCUSA Daily Devotion (pcusa.org)",
            "schema_version": "1.0",
            "total_positions": len(daily_lectionary),
            "year_1_complete": len(daily_lectionary) - missing_year_1 - missing_both,
            "year_2_complete": len(daily_lectionary) - missing_year_2 - missing_both,
        },
        "daily_lectionary": daily_lectionary
    }


# =============================================================================
# CLI
# =============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Merge lectionary scraping plan with scraped readings"
    )
    parser.add_argument(
        "--plan",
        type=Path,
        default=Path("scraping_plan_v4.json"),
        help="Path to scraping plan JSON"
    )
    parser.add_argument(
        "--readings",
        type=Path,
        default=Path("scrape_progress.json"),
        help="Path to scraped readings JSON"
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("daily_lectionary.json"),
        help="Output path for merged lectionary"
    )
    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Enable verbose logging"
    )
    
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    logger.info("=" * 60)
    logger.info("Lectionary Data Merge")
    logger.info("=" * 60)
    
    # Load input files
    logger.info(f"Loading plan: {args.plan}")
    plan = load_json(args.plan)
    
    logger.info(f"Loading readings: {args.readings}")
    progress = load_json(args.readings)
    
    # Merge
    result = merge_lectionary(plan, progress)
    
    # Write output
    logger.info(f"Writing output: {args.output}")
    with open(args.output, 'w', encoding='utf-8') as f:
        json.dump(result, f, indent=2, ensure_ascii=False)
    
    logger.info("=" * 60)
    logger.info("Merge complete!")
    logger.info(f"Output: {args.output}")
    logger.info(f"Edge cases logged to: merge_edge_cases.log")
    logger.info("=" * 60)
    
    # Print summary of edge case log
    try:
        with open('merge_edge_cases.log', 'r') as f:
            edge_cases = f.readlines()
            if edge_cases:
                logger.warning(f"Found {len(edge_cases)} edge cases - review merge_edge_cases.log")
            else:
                logger.info("No edge cases found!")
    except FileNotFoundError:
        pass


if __name__ == "__main__":
    main()

# python3 merge_lectionary.py \
#   --plan ../lectionary-parser/scraping_plan_v4.json \
#   --readings ../lectionary-parser/scrape_progress.json \
#   --output daily_lectionary_merged.json