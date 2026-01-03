#!/usr/bin/env python3
"""
PCUSA Daily Lectionary - Missing Dates Scraper

Scrapes specific failing dates and generates SQL to insert into the database.

Usage:
    python3 scrape_missing.py                     # Scrape dates from failing_dates.txt
    python3 scrape_missing.py --dates 2024-11-17 2024-11-18   # Scrape specific dates
    python3 scrape_missing.py --sql              # Generate SQL from scraped data
"""

import json
import os
import random
import sys
import time
from datetime import datetime
from typing import Optional, Dict, List

import requests
from bs4 import BeautifulSoup

# =============================================================================
# CONFIGURATION
# =============================================================================

BASE_URL = "https://pcusa.org/daily/devotion/"
DATES_FILE = "failing_dates.txt"
PROGRESS_FILE = "missing_dates_progress.json"
SQL_OUTPUT_FILE = "insert_missing_readings.sql"

# Rate limiting
MIN_DELAY = 1.0
MAX_DELAY = 2.0

# Request settings
REQUEST_TIMEOUT = 30
USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

# Reading types to extract
READING_TYPES = [
    ("Morning", "#reading-Morning"),
    ("First Reading", "#reading-First-Reading"),
    ("Second Reading", "#reading-Second-Reading"),
    ("Gospel", "#reading-Gospel-Reading"),
    ("Evening", "#reading-Evening"),
]

# =============================================================================
# PROGRESS TRACKING
# =============================================================================

def load_progress() -> Dict:
    """Load progress from file, or return empty state."""
    if os.path.exists(PROGRESS_FILE):
        with open(PROGRESS_FILE, 'r') as f:
            return json.load(f)
    return {
        "completed": {},
        "failed": {},
        "last_updated": None,
    }


def save_progress(progress: Dict):
    """Save progress to file."""
    progress["last_updated"] = datetime.now().isoformat()
    with open(PROGRESS_FILE, 'w') as f:
        json.dump(progress, f, indent=2)


def load_dates_from_file() -> List[str]:
    """Load dates from the failing dates file."""
    if not os.path.exists(DATES_FILE):
        print(f"ERROR: {DATES_FILE} not found!")
        sys.exit(1)
    
    dates = []
    with open(DATES_FILE, 'r') as f:
        for line in f:
            line = line.strip()
            # Skip empty lines and comments
            if line and not line.startswith('#'):
                dates.append(line)
    return dates


def date_to_url(date_str: str) -> str:
    """Convert YYYY-MM-DD to URL."""
    parts = date_str.split('-')
    return f"{BASE_URL}{parts[0]}/{parts[1]}/{parts[2]}"


# =============================================================================
# SCRAPING
# =============================================================================

def fetch_page(url: str) -> Optional[str]:
    """Fetch a page with proper headers."""
    headers = {
        "User-Agent": USER_AGENT,
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    }
    
    try:
        response = requests.get(url, headers=headers, timeout=REQUEST_TIMEOUT)
        response.raise_for_status()
        return response.text
    except requests.RequestException as e:
        print(f"  ERROR fetching {url}: {e}")
        return None


def parse_readings(html: str) -> Dict[str, Optional[str]]:
    """Parse readings from HTML."""
    soup = BeautifulSoup(html, 'html.parser')
    readings = {}
    
    for reading_name, href in READING_TYPES:
        anchor = soup.find('a', href=href)
        
        if anchor:
            verse_p = anchor.find('p', class_='c-lectionary__tab-verse')
            if verse_p:
                reference = verse_p.get_text().strip()
                readings[reading_name] = reference if reference else None
            else:
                readings[reading_name] = None
        else:
            readings[reading_name] = None
    
    return readings


def parse_period_info(html: str) -> Dict[str, str]:
    """
    Try to extract period information from the page.
    Returns dict with period, day_identifier, special_name if found.
    """
    soup = BeautifulSoup(html, 'html.parser')
    info = {}
    
    # Look for the lectionary title/heading
    # This varies by page structure - adjust selectors as needed
    title = soup.find('h1', class_='c-lectionary__title')
    if title:
        info['title'] = title.get_text().strip()
    
    # Look for period indicator
    period_el = soup.find('div', class_='c-lectionary__period')
    if period_el:
        info['period_text'] = period_el.get_text().strip()
    
    return info


def scrape_date(date_str: str) -> Dict:
    """Scrape readings for a single date."""
    url = date_to_url(date_str)
    
    html = fetch_page(url)
    if html is None:
        raise Exception("Failed to fetch page")
    
    readings = parse_readings(html)
    period_info = parse_period_info(html)
    
    return {
        "date": date_str,
        "url": url,
        "readings": readings,
        "period_info": period_info,
        "scraped_at": datetime.now().isoformat(),
    }


# =============================================================================
# SQL GENERATION
# =============================================================================

def generate_sql(progress: Dict) -> str:
    """
    Generate SQL INSERT statements for the scraped data.
    
    This creates:
    1. INSERT for lectionary_days (if period doesn't exist)
    2. INSERT for readings
    """
    lines = []
    lines.append("-- =============================================================================")
    lines.append("-- INSERT MISSING READINGS")
    lines.append(f"-- Generated: {datetime.now().isoformat()}")
    lines.append("-- =============================================================================")
    lines.append("")
    lines.append("-- NOTE: Review this SQL carefully before running!")
    lines.append("-- You may need to adjust period names to match your existing data.")
    lines.append("")
    lines.append("BEGIN TRANSACTION;")
    lines.append("")
    
    # Group by what we need to insert
    for date_str, data in sorted(progress["completed"].items()):
        readings = data.get("readings", {})
        
        lines.append(f"-- Date: {date_str}")
        lines.append(f"-- URL: {data.get('url', 'N/A')}")
        
        # Morning psalms
        morning = readings.get("Morning")
        evening = readings.get("Evening")
        first = readings.get("First Reading")
        second = readings.get("Second Reading")
        gospel = readings.get("Gospel")
        
        lines.append(f"--   Morning Psalms: {morning}")
        lines.append(f"--   Evening Psalms: {evening}")
        lines.append(f"--   First Reading: {first}")
        lines.append(f"--   Second Reading: {second}")
        lines.append(f"--   Gospel: {gospel}")
        lines.append("")
    
    lines.append("COMMIT;")
    lines.append("")
    
    # Now add template SQL for inserting readings
    lines.append("")
    lines.append("-- =============================================================================")
    lines.append("-- TEMPLATE: How to insert missing Year Cycle 2 readings")
    lines.append("-- =============================================================================")
    lines.append("-- ")
    lines.append("-- If Week 27 after Pentecost exists but only has Year Cycle 1,")
    lines.append("-- copy Year Cycle 1 readings to Year Cycle 2:")
    lines.append("-- ")
    lines.append("-- INSERT INTO readings (lectionary_day_id, year_cycle, reading_type, position, reference)")
    lines.append("-- SELECT lectionary_day_id, 2, reading_type, position, reference")
    lines.append("-- FROM readings r")
    lines.append("-- JOIN lectionary_days ld ON r.lectionary_day_id = ld.id")
    lines.append("-- WHERE ld.period = 'Week 27 after Pentecost'")
    lines.append("--   AND r.year_cycle = 1")
    lines.append("--   AND NOT EXISTS (")
    lines.append("--     SELECT 1 FROM readings r2")
    lines.append("--     WHERE r2.lectionary_day_id = r.lectionary_day_id")
    lines.append("--       AND r2.year_cycle = 2")
    lines.append("--       AND r2.reading_type = r.reading_type")
    lines.append("--   );")
    lines.append("")
    
    return "\n".join(lines)


def generate_copy_sql() -> str:
    """
    Generate SQL to copy Year Cycle 1 readings to Year Cycle 2 for missing periods.
    """
    lines = []
    lines.append("-- =============================================================================")
    lines.append("-- COPY YEAR CYCLE 1 TO YEAR CYCLE 2 FOR MISSING PERIODS")
    lines.append(f"-- Generated: {datetime.now().isoformat()}")
    lines.append("-- =============================================================================")
    lines.append("")
    lines.append("BEGIN TRANSACTION;")
    lines.append("")
    
    # Week 27 after Pentecost - copy Y1 to Y2
    lines.append("-- Week 27 after Pentecost: Copy Year Cycle 1 readings to Year Cycle 2")
    lines.append("INSERT INTO readings (lectionary_day_id, year_cycle, reading_type, position, reference)")
    lines.append("SELECT lectionary_day_id, 2, reading_type, position, reference")
    lines.append("FROM readings r")
    lines.append("JOIN lectionary_days ld ON r.lectionary_day_id = ld.id")
    lines.append("WHERE ld.period = 'Week 27 after Pentecost'")
    lines.append("  AND r.year_cycle = 1")
    lines.append("  AND NOT EXISTS (")
    lines.append("    SELECT 1 FROM readings r2")
    lines.append("    WHERE r2.lectionary_day_id = r.lectionary_day_id")
    lines.append("      AND r2.year_cycle = 2")
    lines.append("      AND r2.reading_type = r.reading_type")
    lines.append("      AND r2.position = r.position")
    lines.append("  );")
    lines.append("")
    
    # Week following Sun. between Feb. 11 and 17 - copy Y1 to Y2
    lines.append("-- Week following Sun. between Feb. 11 and 17: Copy Year Cycle 1 readings to Year Cycle 2")
    lines.append("INSERT INTO readings (lectionary_day_id, year_cycle, reading_type, position, reference)")
    lines.append("SELECT lectionary_day_id, 2, reading_type, position, reference")
    lines.append("FROM readings r")
    lines.append("JOIN lectionary_days ld ON r.lectionary_day_id = ld.id")
    lines.append("WHERE ld.period = 'Week following Sun. between Feb. 11 and 17'")
    lines.append("  AND r.year_cycle = 1")
    lines.append("  AND NOT EXISTS (")
    lines.append("    SELECT 1 FROM readings r2")
    lines.append("    WHERE r2.lectionary_day_id = r.lectionary_day_id")
    lines.append("      AND r2.year_cycle = 2")
    lines.append("      AND r2.reading_type = r.reading_type")
    lines.append("      AND r2.position = r.position")
    lines.append("  );")
    lines.append("")
    
    lines.append("COMMIT;")
    lines.append("")
    
    # Verification queries
    lines.append("-- =============================================================================")
    lines.append("-- VERIFICATION QUERIES")
    lines.append("-- =============================================================================")
    lines.append("")
    lines.append("-- Check Week 27 now has both cycles:")
    lines.append("-- SELECT ld.period, ld.day_identifier, r.year_cycle, COUNT(*) as reading_count")
    lines.append("-- FROM lectionary_days ld")
    lines.append("-- JOIN readings r ON r.lectionary_day_id = ld.id")
    lines.append("-- WHERE ld.period = 'Week 27 after Pentecost'")
    lines.append("-- GROUP BY ld.id, r.year_cycle")
    lines.append("-- ORDER BY ld.day_identifier, r.year_cycle;")
    lines.append("")
    
    return "\n".join(lines)


# =============================================================================
# MAIN
# =============================================================================

def run_scraper(dates: List[str]):
    """Scrape the specified dates."""
    print("=" * 60)
    print("Missing Dates Scraper")
    print("=" * 60)
    
    progress = load_progress()
    completed_dates = set(progress["completed"].keys())
    
    remaining = [d for d in dates if d not in completed_dates]
    
    print(f"\nTotal dates: {len(dates)}")
    print(f"Already completed: {len(completed_dates)}")
    print(f"Remaining: {len(remaining)}")
    
    if not remaining:
        print("\nAll dates scraped!")
        return
    
    print(f"\nStarting scrape...")
    print("-" * 60)
    
    try:
        for i, date_str in enumerate(remaining):
            print(f"[{i+1}/{len(remaining)}] Scraping {date_str}...", end=" ")
            
            try:
                result = scrape_date(date_str)
                progress["completed"][date_str] = result
                
                readings_found = sum(1 for v in result["readings"].values() if v)
                print(f"OK ({readings_found}/5 readings)")
                
            except Exception as e:
                progress["failed"][date_str] = str(e)
                print(f"FAILED: {e}")
            
            save_progress(progress)
            
            delay = random.uniform(MIN_DELAY, MAX_DELAY)
            time.sleep(delay)
    
    except KeyboardInterrupt:
        print("\n\nStopping...")
        save_progress(progress)


def main():
    if len(sys.argv) > 1:
        if sys.argv[1] == "--sql":
            # Generate SQL from progress
            progress = load_progress()
            sql = generate_sql(progress)
            with open(SQL_OUTPUT_FILE, 'w') as f:
                f.write(sql)
            print(f"SQL written to {SQL_OUTPUT_FILE}")
            
            # Also generate the copy SQL
            copy_sql = generate_copy_sql()
            copy_file = "copy_year_cycle.sql"
            with open(copy_file, 'w') as f:
                f.write(copy_sql)
            print(f"Copy SQL written to {copy_file}")
            
        elif sys.argv[1] == "--dates":
            # Scrape specific dates from command line
            dates = sys.argv[2:]
            if dates:
                run_scraper(dates)
            else:
                print("No dates provided")
        elif sys.argv[1] == "--help":
            print(__doc__)
        else:
            print(f"Unknown command: {sys.argv[1]}")
    else:
        # Load dates from file and scrape
        dates = load_dates_from_file()
        run_scraper(dates)


if __name__ == "__main__":
    main()