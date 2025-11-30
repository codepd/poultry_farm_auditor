#!/usr/bin/env python3
"""
Batch process all 2025 ledger PDFs and calculate monthly/yearly profit/loss summary.
"""

import os
import sys
from pathlib import Path
import pandas as pd
from parse_poultry_statement import parse_pdf_statement, format_indian_number
from datetime import datetime

# Month mapping
MONTH_MAP = {
    'JAN': 'January', 'FEB': 'February', 'FED': 'February', 'MAR': 'March', 
    'APR': 'April', 'MAY': 'May', 'JUN': 'June', 'JUL': 'July', 
    'AUG': 'August', 'SEP': 'September', 'OCT': 'October', 
    'NOV': 'November', 'DEC': 'December'
}

def get_month_from_filename(filename):
    """Extract month abbreviation from filename like 25_01_JAN.pdf"""
    parts = Path(filename).stem.split('_')
    if len(parts) >= 3:
        month_abbr = parts[2].upper()
        return month_abbr
    return None

def process_all_ledgers(ledgers_dir='2025_ledgers', output_dir='2025_parsed_data'):
    """
    Process all PDF files in ledgers directory and generate summary.
    """
    ledgers_path = Path(ledgers_dir)
    output_path = Path(output_dir)
    
    # Create output directory if it doesn't exist
    output_path.mkdir(exist_ok=True)
    
    if not ledgers_path.exists():
        print(f"Error: {ledgers_dir} directory not found")
        return None
    
    # Find all PDF files
    pdf_files = sorted(ledgers_path.glob('*.pdf'))
    
    if not pdf_files:
        print(f"No PDF files found in {ledgers_dir}")
        return None
    
    print(f"Found {len(pdf_files)} PDF files to process\n")
    print("=" * 80)
    
    monthly_summary = []
    all_results = []
    
    for pdf_file in pdf_files:
        month_abbr = get_month_from_filename(pdf_file.name)
        month_name = MONTH_MAP.get(month_abbr, month_abbr)
        
        print(f"\nProcessing: {pdf_file.name} ({month_name} 2025)")
        print("-" * 80)
        
        try:
            # Parse the PDF
            parsed = parse_pdf_statement(str(pdf_file), auto_correct=True, add_parsing_notes=True)
            summary = parsed['summary']
            
            # Generate output filename
            output_filename = f"parsed_{pdf_file.stem}.xlsx"
            output_file = output_path / output_filename
            
            # Export to Excel
            from parse_poultry_statement import export_to_excel
            export_to_excel(parsed, str(output_file))
            
            # Extract key metrics
            net_profit = summary.get('net_profit', 0)
            opening_balance = summary.get('opening_balance')
            closing_balance = summary.get('closing_balance')
            total_eggs = summary.get('total_eggs', 0)
            total_feeds = summary.get('total_feeds', 0)
            total_medicines = summary.get('total_medicines', 0)
            total_payments = summary.get('total_payments', 0)
            total_tds = summary.get('total_tds', 0)
            total_discounts = summary.get('total_discounts', 0)
            validation_diff = summary.get('validation_difference', 0)
            
            monthly_summary.append({
                'Month': month_name,
                'Month_Abbr': month_abbr,
                'Opening_Balance': opening_balance,
                'Closing_Balance': closing_balance,
                'Total_Eggs_Sold': total_eggs,
                'Total_Feeds_Purchased': total_feeds,
                'Total_Medicines': total_medicines,
                'Total_Payments': total_payments,
                'Total_TDS': total_tds,
                'Total_Discounts': total_discounts,
                'Net_Profit_Loss': net_profit,
                'Validation_Difference': validation_diff,
                'File': pdf_file.name,
                'Parsed_File': output_filename
            })
            
            all_results.append(parsed)
            
            print(f"✓ Processed successfully")
            print(f"  Net Profit/Loss: {format_indian_number(net_profit)}")
            if validation_diff is not None and validation_diff > 0.01:
                print(f"  ⚠️  Validation difference: {format_indian_number(validation_diff)}")
            
        except Exception as e:
            print(f"✗ Error processing {pdf_file.name}: {e}")
            import traceback
            traceback.print_exc()
            continue
    
    # Create summary DataFrame
    if monthly_summary:
        summary_df = pd.DataFrame(monthly_summary)
        
        # Calculate totals
        total_profit_loss = summary_df['Net_Profit_Loss'].sum()
        total_eggs = summary_df['Total_Eggs_Sold'].sum()
        total_feeds = summary_df['Total_Feeds_Purchased'].sum()
        total_medicines = summary_df['Total_Medicines'].sum()
        total_payments = summary_df['Total_Payments'].sum()
        total_tds = summary_df['Total_TDS'].sum()
        total_discounts = summary_df['Total_Discounts'].sum()
        
        # Save summary to Excel
        summary_file = output_path / '2025_Yearly_Summary.xlsx'
        with pd.ExcelWriter(summary_file, engine='openpyxl') as writer:
            summary_df.to_excel(writer, sheet_name='Monthly_Summary', index=False)
            
            # Create totals row
            totals_row = pd.DataFrame([{
                'Month': 'TOTAL (Jan-Oct 2025)',
                'Month_Abbr': '',
                'Opening_Balance': summary_df['Opening_Balance'].iloc[0] if len(summary_df) > 0 else None,
                'Closing_Balance': summary_df['Closing_Balance'].iloc[-1] if len(summary_df) > 0 else None,
                'Total_Eggs_Sold': total_eggs,
                'Total_Feeds_Purchased': total_feeds,
                'Total_Medicines': total_medicines,
                'Total_Payments': total_payments,
                'Total_TDS': total_tds,
                'Total_Discounts': total_discounts,
                'Net_Profit_Loss': total_profit_loss,
                'Validation_Difference': None,
                'File': '',
                'Parsed_File': ''
            }])
            totals_row.to_excel(writer, sheet_name='Yearly_Total', index=False)
        
        # Print summary
        print("\n" + "=" * 80)
        print("YEARLY SUMMARY (January - October 2025)")
        print("=" * 80)
        print(f"\nMonthly Breakdown:")
        print("-" * 80)
        for _, row in summary_df.iterrows():
            profit_loss = row['Net_Profit_Loss']
            status = "Profit" if profit_loss >= 0 else "Loss"
            print(f"{row['Month']:12} | Net {status:6} | {format_indian_number(profit_loss):>20}")
        
        print("-" * 80)
        print(f"\nTOTAL PROFIT/LOSS (Jan-Oct 2025): {format_indian_number(total_profit_loss)}")
        print(f"  Total Eggs Sold:     {format_indian_number(total_eggs)}")
        print(f"  Total Feeds:         {format_indian_number(total_feeds)}")
        print(f"  Total Medicines:     {format_indian_number(total_medicines)}")
        print(f"  Total Payments:      {format_indian_number(total_payments)}")
        print(f"  Total TDS:           {format_indian_number(total_tds)}")
        print(f"  Total Discounts:     {format_indian_number(total_discounts)}")
        
        if len(summary_df) > 0:
            opening = summary_df['Opening_Balance'].iloc[0]
            closing = summary_df['Closing_Balance'].iloc[-1]
            if opening is not None and closing is not None:
                print(f"\n  Opening Balance (Jan): {format_indian_number(opening)}")
                print(f"  Closing Balance (Oct): {format_indian_number(closing)}")
        
        print(f"\n✓ Summary saved to: {summary_file}")
        print(f"✓ All parsed files saved to: {output_path}")
        
        return summary_df, total_profit_loss
    
    return None, None

if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser(description='Batch process 2025 ledger PDFs')
    parser.add_argument('--ledgers-dir', default='2025_ledgers', help='Directory containing PDF files')
    parser.add_argument('--output-dir', default='2025_parsed_data', help='Output directory for parsed files')
    args = parser.parse_args()
    
    process_all_ledgers(args.ledgers_dir, args.output_dir)

