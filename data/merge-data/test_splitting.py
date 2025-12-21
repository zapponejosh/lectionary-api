#!/usr/bin/env python3
"""
Test the reference splitting logic with real examples from scraped data.
"""

import sys

from merge_lectionary import split_references, split_psalm_references

# Test cases from actual scraped data
test_cases = [
    # (input, reading_type, expected_output)
    
    # Simple single reference
    ("Isaiah 1:1-9", "first", ["Isaiah 1:1-9"]),
    ("2 Peter 3:1-10", "second", ["2 Peter 3:1-10"]),
    ("Matt. 25:1-13", "gospel", ["Matt. 25:1-13"]),
    
    # Semicolon within same book (carry book forward)
    ("Isaiah 45:14-19; 61:1-9", "first", ["Isaiah 45:14-19", "Isaiah 61:1-9"]),
    
    # Comma between different books
    ("Jonah 2:2-9, Isaiah 66:18-23", "first", ["Jonah 2:2-9", "Isaiah 66:18-23"]),
    
    # Comma within same verse range (should NOT split)
    ("John 9:1-12, 35-38", "gospel", ["John 9:1-12, 35-38"]),
    
    # Mixed: comma splits books, semicolon splits within
    ("Colossians 1:24-2:7, Galatians 3:23-29; 4:4-7", "second", 
     ["Colossians 1:24-2:7", "Galatians 3:23-29", "Galatians 4:4-7"]),
    
    # Complex from actual data: 2026-01-09
    ("Isaiah 45:14-19; 61:1-9", "first", ["Isaiah 45:14-19", "Isaiah 61:1-9"]),
    ("Colossians 1:24-2:7, Galatians 3:23-29; 4:4-7", "second",
     ["Colossians 1:24-2:7", "Galatians 3:23-29", "Galatians 4:4-7"]),
    
    # Parenthetical verses (keep as-is)
    ("Deuteronomy 9:(1-3) 4-12", "first", ["Deuteronomy 9:(1-3) 4-12"]),
    ("Isaiah (42:18-25) 43:1-13", "first", ["Isaiah (42:18-25) 43:1-13"]),
    
    # Acts with parentheses from data
    ("Acts (17:12-21) 17:23-24", "second", ["Acts (17:12-21) 17:23-24"]),
    
    # Job references with multiple chapter:verse - SPLIT into separate refs
    ("Job 12:1, 13:3-17, 21-27", "first", ["Job 12:1", "Job 13:3-17, 21-27"]),
    ("Job 16:16-22, 17:1, 17:13-16", "first", ["Job 16:16-22", "Job 17:1", "Job 17:13-16"]),
    
    # 1 Tim. with parentheses
    ("1 Timothy 1:18-2:8 (9-15)", "second", ["1 Timothy 1:18-2:8 (9-15)"]),
    
    # Two completely separate books with comma
    ("Ephesians 6:10-20, Romans 15:7-13", "second", 
     ["Ephesians 6:10-20", "Romans 15:7-13"]),
    
    # Reference with chapter spanning
    ("Luke 20:41-21:4", "gospel", ["Luke 20:41-21:4"]),
    
    # Song of Solomon - multi-word book name with semicolons
    ("Song of Solomon 1:1-3, 9-11, 15-16a; 2:2-3a", "first", 
     ["Song of Solomon 1:1-3, 9-11, 15-16a", "Song of Solomon 2:2-3a"]),
    ("Song of Solomon 2:8-13; 4:1-4a, 5-7, 9-11", "first",
     ["Song of Solomon 2:8-13", "Song of Solomon 4:1-4a, 5-7, 9-11"]),
    
    # Source data typos that need correction
    ("Kings 3:5-14", "first", ["1 Kings 3:5-14"]),
    ("Lamentation 1:1-2, 6-12", "first", ["Lamentations 1:1-2, 6-12"]),
    
    # Verse fragments with letters (a, b suffixes)
    ("Ezekiel 1:1-14, 24-28(b)", "first", ["Ezekiel 1:1-14, 24-28(b)"]),
    ("Ezekiel 7:10-15, 23b-27", "first", ["Ezekiel 7:10-15, 23b-27"]),
    ("1 Samuel 1:1-2, 7b-28", "first", ["1 Samuel 1:1-2, 7b-28"]),
]

psalm_test_cases = [
    # (input, expected_output)
    ("Psalm 122; 145", ["122", "145"]),
    ("Psalm 46; 97; 147:1-11", ["46", "97", "147:1-11"]),
    ("Psalm 24; 150", ["24", "150"]),
    ("Psalm 25; 110", ["25", "110"]),
    ("Psalm 89:1-18; 147:1-11", ["89:1-18", "147:1-11"]),
    ("Psalm 119:73-80; 145", ["119:73-80", "145"]),
]

def run_tests():
    print("=" * 70)
    print("Testing Reference Splitting")
    print("=" * 70)
    
    passed = 0
    failed = 0
    
    for raw, reading_type, expected in test_cases:
        result = split_references(raw, reading_type)
        status = "✓" if result == expected else "✗"
        
        if result == expected:
            passed += 1
            print(f"{status} {reading_type}: {raw}")
            print(f"   → {result}")
        else:
            failed += 1
            print(f"{status} {reading_type}: {raw}")
            print(f"   Expected: {expected}")
            print(f"   Got:      {result}")
        print()
    
    print("=" * 70)
    print("Testing Psalm Splitting")
    print("=" * 70)
    
    for raw, expected in psalm_test_cases:
        result = split_psalm_references(raw)
        status = "✓" if result == expected else "✗"
        
        if result == expected:
            passed += 1
            print(f"{status} {raw}")
            print(f"   → {result}")
        else:
            failed += 1
            print(f"{status} {raw}")
            print(f"   Expected: {expected}")
            print(f"   Got:      {result}")
        print()
    
    print("=" * 70)
    print(f"Results: {passed} passed, {failed} failed")
    print("=" * 70)
    
    return failed == 0

if __name__ == "__main__":
    success = run_tests()
    sys.exit(0 if success else 1)