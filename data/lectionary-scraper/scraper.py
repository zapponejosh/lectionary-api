#!/usr/bin/env python3
"""
PCUSA Daily Lectionary Scraper - Date Range Version

Scrapes readings from https://pcusa.org/daily/devotion/{date} for a date range.

Features:
- Accepts start and end dates as arguments
- Saves progress after each page (can stop and resume)
- Random delays between requests to be polite
- Extracts all 5 reading types (Morning, First, Second, Gospel, Evening)
- Outputs clean JSON

Usage:
    python3 scraper.py 2024-01-01 2027-12-31  # Scrape 4 years
    python3 scraper.py 2026-01-01 2026-12-31  # Scrape 2026 only
    python3 scraper.py --status                 # Show progress
    python3 scraper.py --reset                  # Clear progress and start over
    python3 scraper.py --export                 # Export completed data to final JSON
"""

import json
import os
import random
import sys
import time
from datetime import datetime, timedelta
from pathlib import Path
from typing import Optional, Dict, List
from urllib.parse import urljoin

import requests
from bs4 import BeautifulSoup

# =============================================================================
# CONFIGURATION
# =============================================================================

BASE_URL = "https://pcusa.org/daily/devotion"
PROGRESS_FILE = "scrape_progress.json"
OUTPUT_FILE = "scraped_readings.json"

# Rate limiting: random delay between MIN and MAX seconds
MIN_DELAY = 1.0
MAX_DELAY = 3.0

# Request settings
REQUEST_TIMEOUT = 30
USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

# Reading types to extract (anchor href values)
READING_TYPES = [
    ("Morning", "#reading-Morning"),
    ("First Reading", "#reading-First-Reading"),
    ("Second Reading", "#reading-Second-Reading"),
    ("Gospel", "#reading-Gospel-Reading"),
    ("Evening", "#reading-Evening"),
]

# =============================================================================
# DATE RANGE GENERATION
# =============================================================================

def generate_urls(start_date: str, end_date: str) -> List[str]:
    """
    Generate URLs for date range.
    
    Args:
        start_date: YYYY-MM-DD
        end_date: YYYY-MM-DD (inclusive)
    
    Returns:
        List of URLs
    """
    start = datetime.strptime(start_date, "%Y-%m-%d")
    end = datetime.strptime(end_date, "%Y-%m-%d")
    
    urls = []
    current = start
    while current <= end:
        url = f"{BASE_URL}/{current.year}/{current.month:02d}/{current.day:02d}"
        urls.append(url)
        current += timedelta(days=1)
    
    return urls


def extract_date_from_url(url: str) -> str:
    """Extract YYYY-MM-DD from URL."""
    # URL format: https://pcusa.org/daily/devotion/2025/12/01
    parts = url.rstrip('/').split('/')
    return f"{parts[-3]}-{parts[-2]}-{parts[-1]}"


# =============================================================================
# PROGRESS TRACKING
# =============================================================================

def load_progress() -> Dict:
    """Load progress from file, or return empty state."""
    if os.path.exists(PROGRESS_FILE):
        with open(PROGRESS_FILE, 'r') as f:
            return json.load(f)
    return {
        "completed": {},  # date -> readings dict
        "failed": {},     # date -> error message
        "date_range": None,  # {"start": "YYYY-MM-DD", "end": "YYYY-MM-DD"}
        "last_updated": None,
    }


def save_progress(progress: Dict):
    """Save progress to file."""
    progress["last_updated"] = datetime.now().isoformat()
    with open(PROGRESS_FILE, 'w') as f:
        json.dump(progress, f, indent=2)


# =============================================================================
# SCRAPING
# =============================================================================

def fetch_page(url: str) -> Optional[str]:
    """Fetch a page with proper headers. Returns HTML or None on error."""
    headers = {
        "User-Agent": USER_AGENT,
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        "Accept-Language": "en-US,en;q=0.5",
    }
    
    try:
        response = requests.get(url, headers=headers, timeout=REQUEST_TIMEOUT)
        response.raise_for_status()
        return response.text
    except requests.RequestException as e:
        print(f"  ERROR fetching {url}: {e}")
        return None


def parse_readings(html: str) -> Dict[str, Optional[str]]:
    """
    Parse readings from HTML.
    
    Returns dict with keys: Morning, First Reading, Second Reading, Gospel, Evening
    Values are the scripture references (trimmed) or None if not found.
    """
    soup = BeautifulSoup(html, 'html.parser')
    readings = {}
    
    for reading_name, href in READING_TYPES:
        # Find anchor tag with matching href
        anchor = soup.find('a', href=href)
        
        if anchor:
            # Find paragraph with class "c-lectionary__tab-verse" inside
            verse_p = anchor.find('p', class_='c-lectionary__tab-verse')
            
            if verse_p:
                # Get text and strip whitespace
                reference = verse_p.get_text().strip()
                readings[reading_name] = reference if reference else None
            else:
                readings[reading_name] = None
        else:
            readings[reading_name] = None
    
    return readings


def scrape_date(url: str) -> Dict:
    """
    Scrape readings for a single date.
    
    Returns dict with:
        - date: YYYY-MM-DD
        - url: the source URL
        - readings: dict of reading_name -> reference
        - scraped_at: timestamp
    """
    date = extract_date_from_url(url)
    
    html = fetch_page(url)
    if html is None:
        raise Exception("Failed to fetch page")
    
    readings = parse_readings(html)
    
    return {
        "date": date,
        "url": url,
        "readings": readings,
        "scraped_at": datetime.now().isoformat(),
    }


# =============================================================================
# MAIN SCRAPING LOOP
# =============================================================================

def run_scraper(start_date: str, end_date: str):
    """Main scraping loop with progress tracking."""
    print("=" * 60)
    print("PCUSA Daily Lectionary Scraper")
    print("=" * 60)
    print(f"Date range: {start_date} to {end_date}")
    print()
    
    # Generate URLs
    urls = generate_urls(start_date, end_date)
    print(f"Generated {len(urls)} URLs")
    
    # Load progress
    progress = load_progress()
    
    # Update date range in progress
    progress["date_range"] = {"start": start_date, "end": end_date}
    
    # Figure out what's left to do
    completed_dates = set(progress["completed"].keys())
    failed_dates = set(progress["failed"].keys())
    
    remaining_urls = [
        url for url in urls 
        if extract_date_from_url(url) not in completed_dates
    ]
    
    print(f"Already completed: {len(completed_dates)}")
    print(f"Previously failed: {len(failed_dates)}")
    print(f"Remaining: {len(remaining_urls)}")
    
    if not remaining_urls:
        print("\nAll URLs have been scraped!")
        print(f"Run with --export to generate {OUTPUT_FILE}")
        return
    
    print(f"\nStarting scrape... (Ctrl+C to stop safely)")
    print("-" * 60)
    
    try:
        for i, url in enumerate(remaining_urls):
            date = extract_date_from_url(url)
            
            print(f"[{len(completed_dates) + i + 1}/{len(urls)}] Scraping {date}...", end=" ")
            
            try:
                result = scrape_date(url)
                progress["completed"][date] = result
                
                # Count how many readings we got
                readings_found = sum(1 for v in result["readings"].values() if v)
                print(f"OK ({readings_found}/5 readings)")
                
            except Exception as e:
                progress["failed"][date] = str(e)
                print(f"FAILED: {e}")
            
            # Save progress after each page
            save_progress(progress)
            
            # Random delay before next request
            delay = random.uniform(MIN_DELAY, MAX_DELAY)
            time.sleep(delay)
    
    except KeyboardInterrupt:
        print("\n\nStopping... (progress saved)")
        save_progress(progress)
        print(f"Completed: {len(progress['completed'])}")
        print(f"Run again to resume.")


def show_status():
    """Show current progress status."""
    progress = load_progress()
    
    completed = len(progress["completed"])
    failed = len(progress["failed"])
    
    print("=" * 60)
    print("Scraping Status")
    print("=" * 60)
    
    if progress.get("date_range"):
        dr = progress["date_range"]
        print(f"Date range:      {dr['start']} to {dr['end']}")
        
        # Calculate total days
        start = datetime.strptime(dr['start'], "%Y-%m-%d")
        end = datetime.strptime(dr['end'], "%Y-%m-%d")
        total = (end - start).days + 1
        
        print(f"Total dates:     {total}")
        print(f"Completed:       {completed} ({100*completed/total:.1f}%)")
        print(f"Failed:          {failed}")
        print(f"Remaining:       {total - completed - failed}")
    else:
        print("No date range set. Run scraper with start and end dates.")
        print(f"Completed:       {completed}")
        print(f"Failed:          {failed}")
    
    if progress["last_updated"]:
        print(f"Last updated:    {progress['last_updated']}")
    
    if failed > 0:
        print(f"\nFailed dates:")
        for date, error in list(progress["failed"].items())[:10]:
            print(f"  {date}: {error}")
        if failed > 10:
            print(f"  ... and {failed - 10} more")
    
    # Sample of completed data
    if completed > 0:
        print(f"\nSample completed entry:")
        sample_date = list(progress["completed"].keys())[0]
        sample = progress["completed"][sample_date]
        print(f"  Date: {sample['date']}")
        for reading, ref in sample["readings"].items():
            status = ref if ref else "(not found)"
            print(f"    {reading}: {status}")


def reset_progress():
    """Reset progress (with confirmation)."""
    if os.path.exists(PROGRESS_FILE):
        confirm = input(f"Delete {PROGRESS_FILE} and start over? (yes/no): ")
        if confirm.lower() == "yes":
            os.remove(PROGRESS_FILE)
            print("Progress reset.")
        else:
            print("Cancelled.")
    else:
        print("No progress file to reset.")


def export_data():
    """Export completed data to final JSON file."""
    progress = load_progress()
    
    if not progress["completed"]:
        print("No data to export. Run the scraper first.")
        return
    
    # Sort by date
    sorted_dates = sorted(progress["completed"].keys())
    
    output = {
        "metadata": {
            "exported_at": datetime.now().isoformat(),
            "total_dates": len(sorted_dates),
            "source": "https://pcusa.org/daily/devotion/",
            "date_range": progress.get("date_range"),
        },
        "readings_by_date": {
            date: progress["completed"][date]
            for date in sorted_dates
        }
    }
    
    with open(OUTPUT_FILE, 'w') as f:
        json.dump(output, f, indent=2)
    
    print(f"✅ Exported {len(sorted_dates)} dates to {OUTPUT_FILE}")
    
    # Show stats
    reading_counts = {rt: 0 for rt, _ in READING_TYPES}
    for date_data in progress["completed"].values():
        for reading_name, ref in date_data["readings"].items():
            if ref:
                reading_counts[reading_name] += 1
    
    print("\nReadings found:")
    for reading_name, count in reading_counts.items():
        pct = 100 * count / len(sorted_dates)
        print(f"  {reading_name}: {count}/{len(sorted_dates)} ({pct:.1f}%)")


# =============================================================================
# CLI
# =============================================================================

def main():
    if len(sys.argv) == 1:
        print(__doc__)
        sys.exit(1)
    
    cmd = sys.argv[1]
    
    if cmd == "--status":
        show_status()
    elif cmd == "--reset":
        reset_progress()
    elif cmd == "--export":
        export_data()
    elif cmd == "--help":
        print(__doc__)
    elif len(sys.argv) == 3:
        # Date range provided
        start_date = sys.argv[1]
        end_date = sys.argv[2]
        
        # Validate date format
        try:
            datetime.strptime(start_date, "%Y-%m-%d")
            datetime.strptime(end_date, "%Y-%m-%d")
        except ValueError:
            print("ERROR: Dates must be in YYYY-MM-DD format")
            print("Example: python3 scraper.py 2024-01-01 2027-12-31")
            sys.exit(1)
        
        run_scraper(start_date, end_date)
    else:
        print("ERROR: Invalid arguments")
        print(__doc__)
        sys.exit(1)


if __name__ == "__main__":
    main()