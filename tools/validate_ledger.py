#!/usr/bin/env python3
import pandas as pd
from pathlib import Path

# Configuration (user-provided values if not present in parsed summary)
opening_balance_user = 1070815.35
closing_balance_user = 1862326.55
payments_user = 1000000.0

xlsx = Path('parsed_JULY_25.xlsx')
if not xlsx.exists():
    print('parsed_JULY_25.xlsx not found; run parse_poultry_statement.py JULY_25.pdf first')
    quit(1)

xl = pd.ExcelFile(xlsx)
summary = pd.read_excel(xl, 'summary')
summary_d = {r['metric']: float(r['value']) for _, r in summary.iterrows()}

grand_total = summary_d.get('grand_total', None)
net_profit = summary_d.get('net_profit', None)
opening_balance_parsed = summary_d.get('opening_balance', None)

# Use parsed opening_balance if available
opening_balance = opening_balance_parsed if opening_balance_parsed is not None else opening_balance_user

# also compute payments/deductions from parsed sheets
payments_df = pd.read_excel(xl, 'payments') if 'payments' in xl.sheet_names else pd.DataFrame()
payments_parsed = payments_df['amount'].sum() if not payments_df.empty else 0.0
payments_sum = payments_parsed if payments_parsed and payments_parsed > 0 else payments_user

print('Parsed grand_total:', grand_total)
print('Parsed net_profit:', net_profit)
print('Opening balance (parsed or provided):', opening_balance)
print('Payments (parsed or provided):', payments_sum)
print('Closing balance (provided):', closing_balance_user)

# Expected net change (operations) must satisfy
# opening_balance + operations - payments = closing
# => operations = closing - opening + payments
expected_ops = closing_balance_user - opening_balance + payments_sum
print('\nExpected operations (closing - opening + payments):', expected_ops)

# Compare with parsing results
print('\nGrand total (parsed):', grand_total)
print('Difference (expected_ops - grand_total):', None if grand_total is None else round(expected_ops - grand_total, 2))
print('\nNet profit (parsed):', net_profit)
print('Difference (expected_ops - net_profit):', None if net_profit is None else round(expected_ops - net_profit, 2))

# Quick diagnostics: tds, discounts, payments
if 'tds' in xl.sheet_names:
    tds_df = pd.read_excel(xl, 'tds')
    print('\nTDS sum (parsed):', tds_df['amount'].sum() if not tds_df.empty else 0.0)
else:
    print('\nNo TDS sheet found')

if 'discounts' in xl.sheet_names:
    discounts_df = pd.read_excel(xl, 'discounts')
    print('Discounts sum (parsed):', discounts_df['amount'].sum() if not discounts_df.empty else 0.0)
else:
    print('No discounts sheet found')

if 'payments' in xl.sheet_names:
    print('Payments sum (parsed):', payments_sum)

print('\nDone validation.')
