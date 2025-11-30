#!/usr/bin/env python3
import pandas as pd
from pathlib import Path

xlsx = Path('parsed_JULY_25.xlsx')
if not xlsx.exists():
    print('parsed_JULY_25.xlsx not found; run parse_poultry_statement.py JULY_25.pdf first')
    quit(1)

xl = pd.ExcelFile(xlsx)
print('Sheets:', xl.sheet_names)
raw = pd.read_excel(xl, 'raw_rows')
print('Raw rows:', len(raw))

print('\nSummary of categories:')
print(raw['category'].value_counts())

print('\nTop 10 items by amount across all categories:')
print(raw.sort_values('amount', ascending=False).head(10)[['date','category','item_name','qty','unit','rate','amount']])

print('\nPossible misclassified items (eggs with KGS or feeds with NOS):')
mask_eggs_kg = (raw['category']=='egg') & (raw['unit'].str.contains('KG', na=False))
mask_feed_nos = (raw['category']=='feed') & (raw['unit'].str.contains('NOS', na=False))
print('Eggs with KG:', raw[mask_eggs_kg][['item_name','unit','qty','amount']].head(10))
print('Feed with NOS:', raw[mask_feed_nos][['item_name','unit','qty','amount']].head(10))

print('\nSum by category (amount):')
print(raw.groupby('category')['amount'].sum())

# show rows with qty*rate not equal to amount (helps find mis-assignments)
print('\nRows where qty*rate != amount (within 0.1 tolerance):')
mask_qty_rate = raw['qty'].notna() & raw['rate'].notna() & raw['amount'].notna()
err_rows = raw[mask_qty_rate].copy()
err_rows['expected'] = (err_rows['qty'] * err_rows['rate']).round(2)
err_rows['diff'] = (err_rows['amount'] - err_rows['expected']).round(2)
print(err_rows[abs(err_rows['diff'])>0.1][['category','item_name','qty','unit','rate','amount','expected','diff']].head(20))

print('\nDone analysis')
