#!/usr/bin/env python3
"""
Validate prices in parsed PDF statements against actual price history.
Compares rates from PDF with expected prices from price history Excel file.
"""

import sys
import pandas as pd
import argparse
from datetime import datetime
from pathlib import Path
import re

def load_price_history(price_file, month=None):
    """
    Load price history from Excel file.
    Expected structure:
    - 'FEED PRICE' sheet with columns: Date, LAYER MASH, GROWER MASH, PRE LAYER MASH (or PLM)
    - 'EGG PRICE' sheet (or similar) with columns: Date, LARGE EGG, MEDIUM EGG, SMALL EGG
    month: 'JUL', 'AUG', 'SEP' etc. to select the correct egg price sheet
    """
    if not Path(price_file).exists():
        print(f"Error: Price history file {price_file} not found")
        print("Please convert Poultry_Farm_Price_History_Details_2025.numbers to Excel first")
        return None, None
    
    try:
        xl = pd.ExcelFile(price_file, engine='openpyxl')
        sheet_names = xl.sheet_names
        print(f"Found sheets: {', '.join(sheet_names)}")
        
        feed_prices = None
        egg_prices = None
        
        # Look for feed price sheet - try multiple patterns
        feed_sheet = None
        for name in sheet_names:
            name_upper = name.upper()
            if 'FEED' in name_upper and 'PRICE' in name_upper:
                feed_sheet = name
                break
            elif 'FEED' in name_upper:
                feed_sheet = name
                break
        
        if feed_sheet:
            # Read without header first to check structure
            feed_prices_raw = pd.read_excel(xl, sheet_name=feed_sheet, header=None)
            # Check if row 1 contains headers (like "Start Date")
            # Row 0 might be "Table 1", row 1 is the actual header row
            first_cell_row1 = str(feed_prices_raw.iloc[1, 0]) if len(feed_prices_raw) > 1 else ''
            if 'Start Date' in first_cell_row1 or ('Date' in first_cell_row1 and 'Table' not in first_cell_row1):
                # Use row 1 as column names, skip row 0
                feed_prices = feed_prices_raw.copy()
                feed_prices.columns = feed_prices.iloc[1]
                feed_prices = feed_prices[2:].reset_index(drop=True)
            else:
                # Try row 0 as headers
                feed_prices = feed_prices_raw.copy()
                feed_prices.columns = feed_prices.iloc[0]
                feed_prices = feed_prices[1:].reset_index(drop=True)
            print(f"Loaded feed prices from '{feed_sheet}' sheet")
            # Show actual column names (filter out "Unnamed" and NaN)
            col_names = [str(col) for col in feed_prices.columns.tolist() 
                        if 'Unnamed' not in str(col) and str(col) != 'nan' and pd.notna(col)]
            print(f"Columns: {', '.join(col_names[:8])}...")
        
        # Look for egg price sheet - use month-specific sheet if provided
        egg_sheet = None
        if month:
            month_upper = month.upper()
            for name in sheet_names:
                name_upper = name.upper()
                if month_upper in name_upper and 'EGG' in name_upper:
                    egg_sheet = name
                    break
        
        # Fallback to any egg price sheet
        if not egg_sheet:
            for name in sheet_names:
                name_upper = name.upper()
                if 'EGG' in name_upper and 'PRICE' in name_upper:
                    egg_sheet = name
                    break
                elif 'EGG' in name_upper and 'PRICE' not in name_upper:
                    # Might be a sheet with egg prices but not named "EGG PRICE"
                    egg_sheet = name
                    break
        
        if egg_sheet:
            # Read the entire sheet to find AVG PRICE row
            egg_prices_raw = pd.read_excel(xl, sheet_name=egg_sheet, header=None)
            print(f"Loaded egg prices from '{egg_sheet}' sheet")
            
            # Find the "AVG PRICE in Rupee" row
            avg_price_row = None
            header_row = None
            for idx, row in egg_prices_raw.iterrows():
                first_cell = str(row.iloc[0]).upper()
                if 'AVG PRICE IN RUPEE' in first_cell or ('AVG' in first_cell and 'RUPEE' in first_cell):
                    avg_price_row = row
                    # Header row should be the first row (index 0)
                    if idx > 0:
                        header_row = egg_prices_raw.iloc[0]
                    break
            
            if avg_price_row is not None and header_row is not None:
                # Store as dict with avg prices and headers
                egg_prices = {'avg_prices': avg_price_row, 'headers': header_row}
                print(f"Found AVG PRICE in Rupee row")
            else:
                print("Warning: Could not find 'AVG PRICE in Rupee' row")
                egg_prices = None
        
        return feed_prices, egg_prices
        
    except Exception as e:
        print(f"Error loading price history: {e}")
        return None, None

def parse_date(date_str):
    """Parse date string in various formats"""
    if pd.isna(date_str):
        return None
    
    if isinstance(date_str, datetime):
        return date_str
    
    date_formats = ['%d-%b-%y', '%d-%b-%Y', '%Y-%m-%d', '%d/%m/%Y', '%m/%d/%Y']
    for fmt in date_formats:
        try:
            return datetime.strptime(str(date_str), fmt)
        except:
            continue
    
    return None

def get_feed_price(feed_prices, feed_type, date):
    """
    Get feed price for a specific type and date.
    feed_type: 'LAYER MASH', 'GROWER MASH', 'PRE LAYER MASH' or 'PLM'
    """
    if feed_prices is None or feed_prices.empty:
        return None
    
    # Normalize feed type
    feed_type_upper = feed_type.upper()
    if 'PLM' in feed_type_upper or 'PRE' in feed_type_upper:
        feed_type_upper = 'PRE LAYER MASH'
    
    # Find date column
    date_col = None
    for col in feed_prices.columns:
        if 'date' in col.lower() or 'Date' in col:
            date_col = col
            break
    
    if date_col is None:
        return None
    
    # Find price column for this feed type
    price_col = None
    for col in feed_prices.columns:
        col_upper = str(col).upper().strip()
        # Try exact match or partial match
        if feed_type_upper in col_upper or col_upper in feed_type_upper:
            price_col = col
            break
        # Also try matching without "BULK" or "MASH"
        col_simple = col_upper.replace('BULK', '').replace('MASH', '').strip()
        feed_simple = feed_type_upper.replace('BULK', '').replace('MASH', '').strip()
        if feed_simple in col_simple or col_simple in feed_simple:
            price_col = col
            break
    
    if price_col is None:
        return None
    
    # Find the row with date <= target date (get most recent price)
    if date:
        feed_prices_copy = feed_prices.copy()
        feed_prices_copy[date_col] = feed_prices_copy[date_col].apply(parse_date)
        feed_prices_copy = feed_prices_copy.sort_values(date_col)
        
        valid_rows = feed_prices_copy[feed_prices_copy[date_col] <= date]
        if not valid_rows.empty:
            latest_row = valid_rows.iloc[-1]
            price = latest_row[price_col]
            if pd.notna(price):
                try:
                    return float(price)
                except:
                    return None
    
    return None

def get_egg_price(egg_prices, egg_type, date):
    """
    Get egg price for a specific type using monthly average.
    egg_type: 'LARGE EGG', 'MEDIUM EGG', 'SMALL EGG'
    Uses 'AVG PRICE in Rupee' row from the sheet (monthly average, already in rupees)
    """
    if egg_prices is None or not isinstance(egg_prices, dict):
        return None
    
    if 'avg_prices' not in egg_prices:
        return None
    
    avg_price_row = egg_prices['avg_prices']
    headers = egg_prices['headers']
    
    # Normalize egg type
    egg_type_upper = egg_type.upper()
    
    # Find the column index for this egg type in headers
    price_col_idx = None
    for idx, header in enumerate(headers):
        if idx == 0:  # Skip date column
            continue
        header_str = str(header).upper().strip()
        if egg_type_upper in header_str or header_str in egg_type_upper:
            price_col_idx = idx
            break
        # Also try matching without "EGG"
        header_simple = header_str.replace('EGG', '').strip()
        egg_simple = egg_type_upper.replace('EGG', '').strip()
        if egg_simple in header_simple or header_simple in egg_simple:
            price_col_idx = idx
            break
    
    if price_col_idx is None:
        return None
    
    # Get the average price from the row (already in rupees)
    try:
        price = avg_price_row.iloc[price_col_idx]
        if pd.notna(price):
            return float(price)
    except:
        return None
    
    return None

def validate_pdf_prices(pdf_parsed_file, price_history_file):
    """
    Validate prices in parsed PDF against price history.
    """
    print(f"\n=== VALIDATING PRICES ===")
    print(f"Parsed PDF: {pdf_parsed_file}")
    print(f"Price History: {price_history_file}\n")
    
    # Extract month from filename to use correct egg price sheet
    month = None
    month_abbrs = ['JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC']
    for abbr in month_abbrs:
        if abbr in pdf_parsed_file.upper():
            month = abbr
            break
    
    # Load price history
    feed_prices, egg_prices = load_price_history(price_history_file, month=month)
    
    if feed_prices is None and egg_prices is None:
        print("Could not load price history. Please ensure the Excel file exists.")
        return
    
    # Load parsed PDF data
    if not Path(pdf_parsed_file).exists():
        print(f"Error: Parsed PDF file {pdf_parsed_file} not found")
        return
    
    xl = pd.ExcelFile(pdf_parsed_file, engine='openpyxl')
    raw_items = pd.read_excel(xl, sheet_name='raw_rows')
    
    # Validation results
    mismatches = []
    
    print("Validating prices...\n")
    
    for idx, row in raw_items.iterrows():
        item_name = str(row.get('item_name', '')).upper()
        rate = row.get('rate')
        date = row.get('date')
        category = row.get('category', '')
        amount = row.get('amount')
        qty = row.get('qty')
        
        if pd.isna(rate) or rate is None:
            continue
        
        expected_price = None
        price_source = None
        
        # Check feed prices
        if category == 'feed':
            for feed_type in ['LAYER MASH', 'GROWER MASH', 'PRE LAYER MASH', 'PLM']:
                if feed_type in item_name:
                    expected_price = get_feed_price(feed_prices, feed_type, date)
                    price_source = f"Feed Price History ({feed_type})"
                    break
        
        # Check egg prices
        elif category == 'egg':
            for egg_type in ['LARGE EGG', 'MEDIUM EGG', 'SMALL EGG']:
                if egg_type in item_name:
                    expected_price = get_egg_price(egg_prices, egg_type, date)
                    price_source = f"Egg Price History ({egg_type})"
                    break
        
        if expected_price is not None:
            diff = abs(rate - expected_price)
            diff_pct = (diff / expected_price * 100) if expected_price > 0 else 0
            
            # Flag if difference is significant (> 0.01 or > 1%)
            if diff > 0.01 or diff_pct > 1:
                mismatches.append({
                    'date': date,
                    'item_name': item_name,
                    'category': category,
                    'qty': qty,
                    'rate_in_pdf': rate,
                    'expected_rate': expected_price,
                    'difference': diff,
                    'difference_pct': diff_pct,
                    'amount': amount,
                    'price_source': price_source
                })
    
    # Report results
    if mismatches:
        print(f"\n⚠️  Found {len(mismatches)} price mismatches:\n")
        df_mismatches = pd.DataFrame(mismatches)
        print(df_mismatches.to_string(index=False))
        
        # Save to Excel
        output_file = pdf_parsed_file.replace('.xlsx', '_price_validation.xlsx')
        with pd.ExcelWriter(output_file, engine='openpyxl') as writer:
            df_mismatches.to_excel(writer, sheet_name='price_mismatches', index=False)
        print(f"\nPrice mismatches saved to: {output_file}")
    else:
        print("✓ All prices match the price history!")
    
    return mismatches

def main():
    parser = argparse.ArgumentParser(description='Validate prices in parsed PDF against price history')
    parser.add_argument('parsed_pdf', help='Path to parsed PDF Excel file (e.g., parsed_JULY_25.xlsx)')
    parser.add_argument('--price-history', default='Poultry_Farm_Price_History_Details_2025.xlsx',
                       help='Path to price history Excel file')
    args = parser.parse_args()
    
    validate_pdf_prices(args.parsed_pdf, args.price_history)

if __name__ == '__main__':
    main()

