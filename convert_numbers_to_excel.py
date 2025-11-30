#!/usr/bin/env python3
"""
Convert Apple Numbers file to Excel format.
This script attempts to read a .numbers file and convert it to .xlsx
"""

import sys
import os
import pandas as pd

def convert_numbers_to_excel(numbers_file, output_file=None):
    """
    Convert Numbers file to Excel.
    If numbers-parser is not available, prompts user to export manually.
    """
    if not os.path.exists(numbers_file):
        print(f"Error: File {numbers_file} not found")
        return False
    
    try:
        import numbers_parser
        doc = numbers_parser.Document(numbers_file)
        sheets = doc.sheets
        
        if output_file is None:
            base = os.path.splitext(numbers_file)[0]
            output_file = f"{base}.xlsx"
        
        with pd.ExcelWriter(output_file, engine='openpyxl') as writer:
            for sheet in sheets:
                tables = sheet.tables
                for table in tables:
                    # Get table data
                    data = []
                    for row in table.rows():
                        row_data = []
                        for cell in row:
                            if cell.value is not None:
                                row_data.append(str(cell.value))
                            else:
                                row_data.append('')
                        data.append(row_data)
                    
                    if data:
                        df = pd.DataFrame(data[1:], columns=data[0] if len(data) > 1 else None)
                        sheet_name = sheet.name or 'Sheet1'
                        df.to_excel(writer, sheet_name=sheet_name, index=False)
        
        print(f"Successfully converted {numbers_file} to {output_file}")
        return True
        
    except ImportError:
        print("ERROR: numbers-parser library is not installed.")
        print("\nPlease do one of the following:")
        print("1. Install numbers-parser: pip3 install numbers-parser")
        print("2. Or manually export the Numbers file to Excel format:")
        print(f"   - Open {numbers_file} in Numbers")
        print(f"   - File > Export To > Excel")
        print(f"   - Save as {output_file or 'Poultry_Farm_Price_History_Details_2025.xlsx'}")
        return False
    except Exception as e:
        print(f"Error converting file: {e}")
        print("\nPlease manually export the Numbers file to Excel format:")
        print(f"   - Open {numbers_file} in Numbers")
        print(f"   - File > Export To > Excel")
        return False

if __name__ == '__main__':
    numbers_file = 'Poultry_Farm_Price_History_Details_2025.numbers'
    if len(sys.argv) > 1:
        numbers_file = sys.argv[1]
    
    output_file = None
    if len(sys.argv) > 2:
        output_file = sys.argv[2]
    
    convert_numbers_to_excel(numbers_file, output_file)


