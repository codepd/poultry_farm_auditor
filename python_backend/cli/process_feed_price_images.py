#!/usr/bin/env python3
"""
Process feed price images (like Everest Feeds rate sheets) and load prices into database.

Usage:
    python process_feed_price_images.py <image_path> [--tenant-id <uuid>] [--tenant-name <name>]
"""

import sys
import os
import re
import argparse
from datetime import datetime
from pathlib import Path

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

try:
    import pytesseract
    from PIL import Image
except ImportError:
    print("Error: pytesseract and PIL are required. Install with: pip install pytesseract pillow")
    sys.exit(1)

from database.connection import get_db_cursor, get_db_config

# Feed type mappings
FEED_TYPE_MAPPINGS = {
    'LAYER MASH': 'LAYER MASH',
    'PRE-LAYER MASH': 'PRE-LAYER MASH',
    'PRE LAYER MASH': 'PRE-LAYER MASH',
    'PLM': 'PRE-LAYER MASH',
    'GROWER MASH': 'GROWER MASH',
    'CHICK MASH': 'CHICK MASH',
    'CHICK': 'CHICK MASH',
}

def extract_text_from_image(image_path):
    """Extract text from image using OCR."""
    try:
        image = Image.open(image_path)
        text = pytesseract.image_to_string(image, lang='eng')
        return text
    except Exception as e:
        print(f"Error extracting text from image: {e}")
        return None

def parse_date(date_str):
    """Parse date string in DD.MM.YY or DD.MM.YYYY format."""
    if not date_str:
        return None
    
    # Clean the date string - remove extra spaces and non-digit characters except dots and dashes
    date_str = re.sub(r'[^\d.\-]', '', date_str.strip())
    
    # Try DD.MM.YY format (e.g., 01.12.25, 22.12.25)
    try:
        parsed = datetime.strptime(date_str, "%d.%m.%y")
        # Validate year is reasonable (2020-2030)
        if 2020 <= parsed.year <= 2030:
            return parsed
    except:
        pass
    
    # Try DD.MM.YYYY format
    try:
        parsed = datetime.strptime(date_str, "%d.%m.%Y")
        if 2020 <= parsed.year <= 2030:
            return parsed
    except:
        pass
    
    # Try DD-MM-YY format
    try:
        parsed = datetime.strptime(date_str, "%d-%m-%y")
        if 2020 <= parsed.year <= 2030:
            return parsed
    except:
        pass
    
    # Try DD-MM-YYYY format
    try:
        parsed = datetime.strptime(date_str, "%d-%m-%Y")
        if 2020 <= parsed.year <= 2030:
            return parsed
    except:
        pass
    
    return None

def extract_feed_prices(text):
    """Extract feed prices from OCR text."""
    if not text:
        return None, None
    
    lines = text.split('\n')
    
    # Find date - look for DATE field or date pattern
    date = None
    feed_prices = {}
    
    # First pass: find date
    # Look for "DATE" label followed by date (most common case)
    for line in lines:
        line_upper = line.upper()
        
        # Look for "DATE" label followed by date pattern DD.MM.YY
        if 'DATE' in line_upper:
            # Try multiple patterns - OCR might add spaces or other characters
            # Pattern 1: DATE: 01.12.25 (flexible with spaces)
            date_match = re.search(r'DATE[:\s]+(\d{1,2})\s*\.\s*(\d{1,2})\s*\.\s*(\d{2,4})', line, re.IGNORECASE)
            if date_match:
                day = date_match.group(1)
                month = date_match.group(2)
                year = date_match.group(3)
                date_str = f"{day}.{month}.{year}"
                date = parse_date(date_str)
                if date:
                    print(f"Found date from DATE field: {date_str} -> {date.strftime('%Y-%m-%d')}")
                    break
            
            # Pattern 2: DATE: 01.12.25 (compact, no spaces)
            if not date:
                date_match = re.search(r'DATE[:\s]+(\d{1,2}\.\d{1,2}\.\d{2,4})', line, re.IGNORECASE)
                if date_match:
                    date_str = date_match.group(1)
                    date = parse_date(date_str)
                    if date:
                        print(f"Found date from DATE field: {date_str} -> {date.strftime('%Y-%m-%d')}")
                        break
    
    # Second pass: if date not found, look for standalone date pattern DD.MM.YY anywhere in text
    if not date:
        # Search entire text for date patterns, not just line by line
        full_text = ' '.join(lines)
        
        # Look for DD.MM.YY pattern (most common format) - be flexible with spaces
        date_match = re.search(r'(\d{1,2})\s*\.\s*(\d{1,2})\s*\.\s*(\d{2,4})', full_text)
        if date_match:
            # Reconstruct date string
            day = date_match.group(1)
            month = date_match.group(2)
            year = date_match.group(3)
            date_str = f"{day}.{month}.{year}"
            parsed = parse_date(date_str)
            if parsed:
                print(f"Found date from pattern in text: {date_str} -> {parsed.strftime('%Y-%m-%d')}")
                date = parsed
        
        # Also try line by line for better context
        if not date:
            for line in lines:
                date_match = re.search(r'(\d{1,2})\s*\.\s*(\d{1,2})\s*\.\s*(\d{2,4})', line)
                if date_match:
                    day = date_match.group(1)
                    month = date_match.group(2)
                    year = date_match.group(3)
                    date_str = f"{day}.{month}.{year}"
                    parsed = parse_date(date_str)
                    if parsed:
                        print(f"Found date from pattern: {date_str} -> {parsed.strftime('%Y-%m-%d')}")
                        date = parsed
                        break
    
    # Second pass: find feed prices
    # Look for feed names and prices in same line or adjacent lines
    for i, line in enumerate(lines):
        line_upper = line.upper().strip()
        
        # Skip empty lines
        if not line_upper:
            continue
        
        # Look for feed type keywords
        for feed_type, normalized_type in FEED_TYPE_MAPPINGS.items():
            if feed_type in line_upper:
                # Try to find price in the same line
                price_match = re.search(r'(\d{3,5})\s*/?\s*-?', line)
                
                # If not found in same line, check next line
                if not price_match and i + 1 < len(lines):
                    next_line = lines[i + 1]
                    price_match = re.search(r'(\d{3,5})\s*/?\s*-?', next_line)
                
                if price_match:
                    try:
                        price = float(price_match.group(1))
                        # Only add if we don't already have this feed type
                        if normalized_type not in feed_prices:
                            feed_prices[normalized_type] = price
                            print(f"Found {normalized_type}: ₹{price}")
                    except ValueError:
                        pass
    
    return date, feed_prices

def normalize_feed_name(feed_name):
    """Normalize feed name to match database values."""
    feed_upper = feed_name.upper().strip()
    
    # Map variations to standard names
    if 'LAYER' in feed_upper and 'PRE' not in feed_upper:
        return 'LAYER MASH'
    elif 'PRE' in feed_upper or 'PLM' in feed_upper:
        return 'PRE-LAYER MASH'
    elif 'GROWER' in feed_upper:
        return 'GROWER MASH'
    elif 'CHICK' in feed_upper:
        return 'CHICK MASH'
    
    return feed_name.upper()

def get_tenant_id(tenant_name=None, tenant_id=None):
    """Get tenant UUID from name or use provided UUID."""
    with get_db_cursor() as cursor:
        if tenant_id:
            cursor.execute("SELECT id FROM tenants WHERE id = %s::uuid", (tenant_id,))
            result = cursor.fetchone()
            if result:
                return result[0]
            print(f"Warning: Tenant ID {tenant_id} not found")
        
        if tenant_name:
            cursor.execute("SELECT id FROM tenants WHERE name = %s", (tenant_name,))
            result = cursor.fetchone()
            if result:
                return result[0]
            print(f"Warning: Tenant '{tenant_name}' not found")
        
        # Get first tenant as fallback
        cursor.execute("SELECT id FROM tenants LIMIT 1")
        result = cursor.fetchone()
        if result:
            print(f"Using first tenant: {result[0]}")
            return result[0]
    
    return None

def insert_feed_prices(tenant_id, date, feed_prices):
    """Insert feed prices into price_history table."""
    if not feed_prices:
        print("No feed prices to insert")
        return 0
    
    inserted = 0
    with get_db_cursor() as cursor:
        for feed_name, price in feed_prices.items():
            normalized_name = normalize_feed_name(feed_name)
            
            # Check if price already exists for this date and item
            cursor.execute("""
                SELECT id FROM price_history
                WHERE tenant_id = %s
                    AND price_date = %s
                    AND price_type = 'FEED'
                    AND item_name = %s
            """, (tenant_id, date.date(), normalized_name))
            
            existing = cursor.fetchone()
            
            if existing:
                # Update existing price
                cursor.execute("""
                    UPDATE price_history
                    SET price = %s
                    WHERE id = %s
                """, (price, existing[0]))
                print(f"Updated {normalized_name}: ₹{price} for {date.strftime('%Y-%m-%d')}")
            else:
                # Insert new price
                cursor.execute("""
                    INSERT INTO price_history (tenant_id, price_date, price_type, item_name, price)
                    VALUES (%s, %s, 'FEED', %s, %s)
                    ON CONFLICT (tenant_id, price_date, price_type, item_name)
                    DO UPDATE SET price = EXCLUDED.price
                """, (tenant_id, date.date(), normalized_name, price))
                print(f"Inserted {normalized_name}: ₹{price} for {date.strftime('%Y-%m-%d')}")
            
            inserted += 1
    
    return inserted

def extract_date_from_filename(filename):
    """Try to extract date from filename patterns like MM_DD_YYYY or DD_MM_YYYY."""
    filename = Path(filename).stem  # Remove extension
    
    # Try MM_DD_YYYY format (e.g., 12_01_2025 -> 2025-12-01)
    match = re.search(r'(\d{1,2})[_\-\s]+(\d{1,2})[_\-\s]+(\d{4})', filename)
    if match:
        part1 = int(match.group(1))
        part2 = int(match.group(2))
        year = int(match.group(3))
        
        # Try MM_DD_YYYY first (most common for US format)
        if 1 <= part1 <= 12 and 1 <= part2 <= 31:
            try:
                date = datetime(year, part1, part2)
                if 2020 <= year <= 2030:
                    return date
            except ValueError:
                pass
        
        # Try DD_MM_YYYY format (swap if first attempt failed)
        if 1 <= part2 <= 12 and 1 <= part1 <= 31:
            try:
                date = datetime(year, part2, part1)
                if 2020 <= year <= 2030:
                    return date
            except ValueError:
                pass
    
    # Try YYYY_MM_DD format
    match = re.search(r'(\d{4})[_\-\s]+(\d{1,2})[_\-\s]+(\d{1,2})', filename)
    if match:
        year = int(match.group(1))
        month = int(match.group(2))
        day = int(match.group(3))
        if 1 <= month <= 12 and 1 <= day <= 31 and 2020 <= year <= 2030:
            try:
                return datetime(year, month, day)
            except ValueError:
                pass
    
    return None

def process_image(image_path, tenant_id=None, tenant_name=None):
    """Process a single image and load feed prices."""
    image_path = Path(image_path)
    
    if not image_path.exists():
        print(f"Error: Image file not found: {image_path}")
        return False
    
    print(f"\nProcessing image: {image_path.name}")
    print("-" * 60)
    
    # Try to extract date from filename first (as fallback)
    filename_date = extract_date_from_filename(image_path.name)
    if filename_date:
        print(f"Found date in filename: {filename_date.strftime('%Y-%m-%d')}")
    
    # Extract text from image
    print("Extracting text from image...")
    text = extract_text_from_image(image_path)
    
    if not text:
        print("Error: Could not extract text from image")
        return False
    
    # Debug: print extracted text
    print("\nExtracted text:")
    print(text[:1000])  # Print first 1000 chars
    print("...\n")
    
    # Also print lines containing "DATE" for debugging
    print("Lines containing 'DATE' or date patterns:")
    for i, line in enumerate(text.split('\n')):
        if 'DATE' in line.upper() or re.search(r'\d{1,2}[.\-]\d{1,2}[.\-]\d{2,4}', line):
            print(f"  Line {i}: {line}")
    print()
    
    # Parse feed prices
    date, feed_prices = extract_feed_prices(text)
    
    # If date not found in OCR, use filename date
    if not date and filename_date:
        date = filename_date
        print(f"Using date from filename: {date.strftime('%Y-%m-%d')}")
    
    if not date:
        print("Error: Could not extract date from image or filename")
        print(f"Filename: {image_path.name}")
        return False
    
    if not feed_prices:
        print("Error: Could not extract feed prices from image")
        return False
    
    # Get tenant ID
    tenant_uuid = get_tenant_id(tenant_name, tenant_id)
    if not tenant_uuid:
        print("Error: Could not find tenant")
        return False
    
    # Insert prices
    print(f"\nInserting prices for tenant: {tenant_uuid}")
    inserted = insert_feed_prices(tenant_uuid, date, feed_prices)
    
    print(f"\n✅ Successfully processed {inserted} feed prices")
    return True

def main():
    parser = argparse.ArgumentParser(
        description='Process feed price images and load into database'
    )
    parser.add_argument('image_path', help='Path to image file')
    parser.add_argument('--tenant-id', help='Tenant UUID')
    parser.add_argument('--tenant-name', default='Pradeep Farm', help='Tenant name (default: Pradeep Farm)')
    
    args = parser.parse_args()
    
    success = process_image(args.image_path, args.tenant_id, args.tenant_name)
    sys.exit(0 if success else 1)

if __name__ == '__main__':
    main()

