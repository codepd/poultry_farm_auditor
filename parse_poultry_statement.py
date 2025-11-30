#!/usr/bin/env python3
"""
Parse poultry ledger PDF statements (like 'PRADEEP PF JULY 25.pdf') and export a structured Excel workbook.

Updated:
 - formats exported date fields as '2-Jun-25'
 - grand_total = Eggs - Feed - Medicine - Other - TDS + Discount (payments are not expenses, they're money received)
 - net_profit = Eggs + Discounts - Feeds - Medicines - Other - TDS
 - FIXED: Now correctly categorizes items based on item name, not header type
 - FIXED: grand_total calculation - payments are money received (reduce balance owed), not expenses to subtract
"""

import sys
import argparse
import re
import os
import pdfplumber
import pandas as pd
from datetime import datetime

# --- configuration / known keywords (tweak if you want) ---
EGG_KEYWORDS = ["LARGE EGG", "MEDIUM EGG", "SMALL EGG", "CORRECT EGG", "EXPORT EGG", "CORRECT SIZE"]
FEED_KEYWORDS = ["LAYER MASH", "GROWER MASH", "PRE LAYER MASH", "LAYER MASH BULK", "GROWER MASH BULK", "PRE LAYER MASH BULK"]
MEDICINES_KNOWN = ["D3 FORTE", "VETMULIN", "OXYCYCLINE", "TIAZIN", "BPPS FORTE", "CTC", "SHELL GRIT",
                   "ROVIMIX", "CHOLIMARIN", "ZAGROMIN", "G PRO NATURO", "NECROVET", "TOXOL", "FRA C12 DRY", "CALCI ROYAL FS", "CALDLIV FS", "RESPAFEED", "VENTRIM (VITAL)"]

NUM_RE = re.compile(r'[\d,]+\.\d+|[\d,]+')
DATE_HDR_RE = re.compile(r'^(\d{1,2}-[A-Za-z]{3}-\d{2,4})\b')

def format_indian_number(num, decimals=2):
    """Format a number in Indian numbering system (lakhs/crores format).
    Example: 1234567.89 -> 12,34,567.89
    """
    if num is None:
        return ''
    try:
        # Handle both int and float
        num = float(num)
        # Split into integer and decimal parts
        if num < 0:
            sign = '-'
            num = abs(num)
        else:
            sign = ''
        
        # Format with specified decimal places
        num_str = f"{num:.{decimals}f}"
        parts = num_str.split('.')
        integer_part_str = parts[0]
        decimal_part_str = parts[1] if len(parts) > 1 else ''
        
        # Convert integer part to string and reverse for easier processing
        reversed_str = integer_part_str[::-1]
        
        # Group digits: first 3, then pairs of 2
        if len(reversed_str) <= 3:
            formatted_int = integer_part_str
        else:
            # First 3 digits
            formatted_parts = [reversed_str[:3]]
            # Remaining digits in pairs
            remaining = reversed_str[3:]
            for i in range(0, len(remaining), 2):
                formatted_parts.append(remaining[i:i+2])
            # Reverse back and join with commas
            formatted_int = ','.join([part[::-1] for part in formatted_parts[::-1]])
        
        # Add decimal part
        if decimal_part_str:
            return f"{sign}{formatted_int}.{decimal_part_str}"
        else:
            return f"{sign}{formatted_int}"
    except (ValueError, TypeError):
        return str(num)

def clean_number(s):
    if not s:
        return None
    s = str(s).strip().replace(',', '')
    try:
        return float(s)
    except:
        m = re.search(r'-?[\d]+(?:\.[\d]+)?', s)
        return float(m.group(0)) if m else None

def find_last_number_token(tokens):
    for i in range(len(tokens)-1, -1, -1):
        if NUM_RE.search(tokens[i]):
            return i, tokens[i]
    return None, None


def find_last_amount_token(tokens):
    """Return (index, token) for the best-guess 'amount' numeric token.
    Skip trailing tokens that are likely reference numbers (BILL NO, SL NO, INV, JOURNAL, etc.).
    Falls back to the last numeric token if nothing else is found.
    """
    # Avoid skipping tokens that can legitimately precede amounts like 'JOURNAL' which often is followed by a journal number and amount.
    skip_prev_tokens = set(['BILL', 'BILLNO', 'BILLNO:', 'NO', 'INV', 'INVOICE', 'SL', 'SR', 'SL.', 'SR.'])
    for i in range(len(tokens)-1, -1, -1):
        if NUM_RE.search(tokens[i]):
            prev = tokens[i-1].upper() if i-1 >= 0 else ''
            # Skip if previous token indicates a bill/invoice/serial number label
            if prev in skip_prev_tokens:
                continue
            # We consider this a potential amount - return it
            return i, tokens[i]
    # If we could not find a safe candidate, return (None, None) rather than falling back to a bill number.
    return None, None

def parse_item_line(line):
    text = line.strip()
    tokens = text.split()
    idx_amount, amount_tok = find_last_amount_token(tokens)
    if idx_amount is None:
        return None
    amount = clean_number(amount_tok)
    amount_source = 'last_token'
    rate_idx = None
    for i, t in enumerate(tokens):
        if '/' in t:
            rate_idx = i
            break
    qty = None
    unit = None
    rate = None
    item_name = None

    if rate_idx is not None:
        rate_tok = tokens[rate_idx]
        rate_num = rate_tok.split('/')[0]
        rate = clean_number(rate_num)
        j = rate_idx - 1
        if j >= 0 and tokens[j].lower() in ('kgs', 'kg', 'nos', 'no', 'nos.'):
            unit = tokens[j]
            if j - 1 >= 0 and NUM_RE.search(tokens[j-1]):
                qty = clean_number(tokens[j-1])
                item_name = ' '.join(tokens[:j-1])
            else:
                if j - 1 >= 0:
                    qty = clean_number(tokens[j-1])
                    item_name = ' '.join(tokens[:j-1])
                else:
                    item_name = ' '.join(tokens[:j])
        else:
            if rate_idx - 1 >= 0 and NUM_RE.search(tokens[rate_idx - 1]):
                qty = clean_number(tokens[rate_idx - 1])
                unit = rate_tok.split('/', 1)[1] if '/' in rate_tok else None
                item_name = ' '.join(tokens[:rate_idx - 1])
            else:
                for k in range(rate_idx-1, -1, -1):
                    if tokens[k].lower() in ('kgs','kg','nos','no','nos.'):
                        unit = tokens[k]
                        if k-1 >= 0 and NUM_RE.search(tokens[k-1]):
                            qty = clean_number(tokens[k-1])
                            item_name = ' '.join(tokens[:k-1])
                        else:
                            item_name = ' '.join(tokens[:k])
                        break
                if not item_name:
                        item_name = ' '.join(tokens[:idx_amount])
    else:
        unit_idx = None
        for i, t in enumerate(tokens):
            if t.lower() in ('kgs','kg','nos','no','nos.'):
                unit_idx = i
                break
        if unit_idx is not None:
            unit = tokens[unit_idx]
            if unit_idx - 1 >= 0 and NUM_RE.search(tokens[unit_idx - 1]):
                qty = clean_number(tokens[unit_idx - 1])
                item_name = ' '.join(tokens[:unit_idx - 1])
            else:
                item_name = ' '.join(tokens[:unit_idx])
        else:
            item_name = ' '.join(tokens[:idx_amount])
            qty = None
            rate = None

    item_name_norm = item_name.strip().upper() if item_name else None

    # Determine amount source: If qty and rate available, and qty*rate matches a token, prefer that token
    try:
        if rate and qty is not None:
            expected_amount = round(float(qty) * float(rate), 2)
            # Search tokens for one matching expected amount
            for tok in tokens[::-1]:
                if NUM_RE.search(tok):
                    tval = clean_number(tok)
                    if tval is None:
                        continue
                    if abs(tval - expected_amount) < 0.01:
                        amount = tval
                        amount_source = 'qty_rate'
                        break
    except Exception:
        pass

    # If qty and rate are present, prefer an amount equal to qty*rate over
    # other trailing numbers like bill numbers or extra columns.
    try:
        if rate and qty is not None:
            expected_amount = round(float(qty) * float(rate), 2)
            # Look for a numeric token that equals expected_amount
            for tok in tokens[::-1]:
                if NUM_RE.search(tok):
                    tval = clean_number(tok)
                    if tval is None:
                        continue
                    if abs(tval - expected_amount) < 0.01:
                        amount = tval
                        break
    except Exception:
        pass
    return {
        'item_name': item_name_norm,
        'qty': qty,
        'unit': (unit.upper() if unit else None),
        'rate': rate,
        'amount': amount
        ,'amount_source': amount_source
    }

def format_date_obj(dt):
    if dt is None:
        return ''
    try:
        if pd.isna(dt):
            return ''
    except:
        pass
    if isinstance(dt, str):
        return dt
    try:
        day = dt.day
        mon = dt.strftime('%b')
        yy = dt.strftime('%y')
        return f"{day}-{mon}-{yy}"
    except Exception:
        return str(dt)

def classify_item_by_name(item_name):
    """Classify an item based on its name into: egg, feed, medicine, or other."""
    if not item_name or item_name == 'NONE':
        return 'other'
    n = item_name.upper()
    
    # Check eggs first (highest priority)
    for e in EGG_KEYWORDS:
        if e in n:
            return 'egg'
    
    # Check feed
    for f in FEED_KEYWORDS:
        if f in n:
            return 'feed'
    
    # Check medicines
    for m in MEDICINES_KNOWN:
        if m in n:
            return 'medicine'
    
    # Additional medicine keywords
    if any(x in n for x in ['GRIT','D3','FRA','VET','CTC','NECRO','TOX','ROVIMIX']):
        return 'medicine'
    
    return 'other'


def determine_category(item_name, unit=None, auto_correct=False):
    """Return (category, note) applying optional heuristics.
    If auto_correct is True, attempt to fix common misclassifications (e.g., 'LARGE' as egg when unit is NOS).
    Returns: (category, note_str_or_None)
    """
    base = classify_item_by_name(item_name)
    note = None
    if auto_correct:
        un = unit.upper() if unit else ''
        # Auto-correct rule: item_name is single word Large/Small/Medium (no 'EGG') and unit is NOS -> egg
        if base != 'egg' and item_name:
            n = item_name.strip().upper()
            # check for 'LARGE' 'SMALL' 'MEDIUM' alone or leading as single word
            if re.match(r'^(LARGE|SMALL|MEDIUM)(\s|$)', n) and un in ('NOS', 'NO'):
                base = 'egg'
                note = 'auto_classified_to_egg'
    return base, note

def parse_pdf_statement(pdf_path, auto_correct=False, add_parsing_notes=True):
    items = [] 
    payments = []
    tds_entries = []
    discounts = []
    opening_balance = None
    closing_balance = None
    
    with pdfplumber.open(pdf_path) as pdf:
        all_lines = []
        for p in pdf.pages:
            text = p.extract_text() or ""
            for line in text.splitlines():
                all_lines.append(line.rstrip()) 

    i = 0
    last_bill_no = None
    while i < len(all_lines):
        line = all_lines[i].strip()
        mdate = DATE_HDR_RE.match(line)
        if mdate:
            date_str = mdate.group(1)
            try:
                parsed_date = datetime.strptime(date_str, "%d-%b-%y") 
            except:
                try:
                    parsed_date = datetime.strptime(date_str, "%d-%b-%Y")
                except:
                    parsed_date = None
            rest = line[mdate.end():].strip()
            header_text = rest.upper()
            header_amt = None
            tn = re.findall(NUM_RE, header_text)
            if tn:
                header_amt = clean_number(tn[-1]) 
            
            # Determine header transaction type (for context only)
            txn_type = None
            if "POULTRY EGG" in header_text or "EGG PURCHASE" in header_text:
                txn_type = "egg_purchase"
            elif "POULTRY FEED" in header_text or "FEED SALES" in header_text:
                txn_type = "feed_sale" 
            elif "CANARA BANK" in header_text or "PAYMENT" in header_text or "TO CANARA BANK" in header_text:
                txn_type = "payment"
            elif re.search(r'\bTDS\b', header_text) and "DEDUCTED" not in header_text:
                txn_type = "tds"
            elif "DISCOUNT" in header_text:
                txn_type = "discount"
            else:
                if "OPENING BALANCE" in header_text: 
                    txn_type = "opening_balance"
                elif "CLOSING BALANCE" in header_text:
                    txn_type = "closing_balance"
                else:
                    txn_type = "other"

            m_bill = re.search(r'BILL\s*NO\.?\s*(\d+)|BILLNO\s*(\d+)', header_text)
            if m_bill:
                last_bill_no = m_bill.group(1) or m_bill.group(2)

            # If the header itself denotes a payment and contains an amount,
            # record it directly so the payments sheet is not left empty.
            if txn_type == 'payment' and header_amt is not None:
                payments.append({ 
                    'date': parsed_date,
                    'amount': header_amt,
                    'raw': line
                })
            # If the header itself denotes TDS and contains an amount, record it
            if txn_type == 'tds' and header_amt is not None:
                tds_entries.append({
                    'date': parsed_date,
                    'amount': header_amt,
                    'bill_no': last_bill_no,
                    'raw': line
                })
            # If the header itself denotes a discount and contains an amount, record it
            if txn_type == 'discount' and header_amt is not None:
                discounts.append({
                    'date': parsed_date,
                    'amount': header_amt,
                    'raw': line
                })
            if txn_type == 'opening_balance' and header_amt is not None:
                opening_balance = header_amt
            if txn_type == 'closing_balance' and header_amt is not None:
                closing_balance = header_amt

            j = i + 1
            while j < len(all_lines):
                nxt = all_lines[j].strip()
                if DATE_HDR_RE.match(nxt):
                    break
                bsearch = re.search(r'BILLNO\s*(\d+)|BILL\s*NO[:\s]*(\d+)|BILL NO (\d+)', nxt.upper())
                if bsearch:
                    last_bill_no = bsearch.group(1) or bsearch.group(2) or bsearch.group(3)

                # Capture 'To TDS' entries precisely (prefer amounts from line, and avoid trailing BILL NO values)
                if re.search(r'\bTO\s+TDS\b', nxt.upper()) or nxt.upper().startswith("TO TDS"):
                    bill_match = re.search(r'BILL\s*NO\.?\s*(\d+)|BILLNO\s*(\d+)', nxt.upper())
                    tds_bill_no = None
                    if bill_match:
                        tds_bill_no = bill_match.group(1) or bill_match.group(2)
                    tokens = nxt.split()
                    tidx, tkn = find_last_amount_token(tokens)
                    amt = clean_number(tkn) if tkn else None
                    if amt is not None and not any(e.get('raw') == nxt for e in tds_entries):
                        tds_entries.append({
                            'date': parsed_date,
                            'amount': amt,
                            'bill_no': tds_bill_no,
                            'raw': nxt
                        })
                if (' KGS' in nxt.upper()) or (' NOS' in nxt.upper()) or ('/KGS' in nxt.upper()) or ('/NOS' in nxt.upper()):
                    parsed = parse_item_line(nxt)
                    if parsed and parsed['item_name']:
                        # CRITICAL FIX: Always classify based on item name, not header
                        actual_category, note_correct = determine_category(parsed['item_name'], parsed.get('unit'), auto_correct=auto_correct)
                        notes = []
                        if add_parsing_notes:
                            # amount source
                            asrc = parsed.get('amount_source')
                            if asrc and asrc != 'last_token':
                                notes.append(f"amount_source={asrc}")
                            # qty-rate mismatch
                            if parsed.get('qty') is not None and parsed.get('rate') is not None:
                                expected = round(parsed.get('qty') * parsed.get('rate'), 2)
                                if parsed.get('amount') is not None and abs(parsed.get('amount') - expected) > 0.1:
                                    notes.append('amount_mismatch_qty_rate')
                            # unit mismatch
                            unit_val = parsed.get('unit')
                            if actual_category == 'egg' and unit_val and unit_val.upper().find('KG') >= 0:
                                notes.append('unit_kg_for_egg')
                            if actual_category == 'feed' and (not unit_val or unit_val.upper().find('KG') < 0):
                                notes.append('unit_not_kg_for_feed')
                            # header vs actual
                            if txn_type and ((txn_type.startswith('egg') and actual_category != 'egg') or (txn_type.startswith('feed') and actual_category != 'feed')):
                                notes.append('category_different_than_header')
                            if note_correct:
                                notes.append(note_correct)
                        
                        rec = {
                            'date': parsed_date,
                            'txn_type': txn_type,  # Keep for reference
                            'category': actual_category,  # Use actual classification
                            'header_amount': header_amt,
                            'item_name': parsed['item_name'],
                            'qty': parsed['qty'],
                            'unit': parsed['unit'],
                            'rate': parsed['rate'],
                            'amount': parsed['amount'],
                            'bill_no': last_bill_no,
                            'raw_line': nxt,
                            'parsing_notes': ';'.join(notes) if notes else None
                        }
                        items.append(rec)
                else:
                    if 'DISCOUNT' in nxt.upper() or 'DISCOUNT' in header_text:
                        n = re.findall(NUM_RE, nxt)
                        if n:
                            discounts.append({
                                'date': parsed_date,
                                'amount': clean_number(n[-1]),
                                'raw': nxt
                            })
                    if 'CANARA BANK' in nxt.upper() or 'PAYMENT' in nxt.upper() or 'TO CANARA BANK' in nxt.upper():
                        n = re.findall(NUM_RE, nxt)
                        amt = clean_number(n[-1]) if n else header_amt
                        payments.append({
                            'date': parsed_date,
                            'amount': amt,
                            'raw': nxt
                        })
                j += 1
            i = j
            continue
        else:
            up = line.upper()
            m = re.search(r'TDS DEDUCTED BILL NO\s*(\d+)|TDS DEDUCTED BILL NO[:\s]*(\d+)|BILLNO\s*(\d+)', up)
            if m:
                last_bill_no = m.group(1) or m.group(2) or m.group(3)
            if (' KGS' in up) or (' NOS' in up) or ('/KGS' in up) or ('/NOS' in up):
                parsed = parse_item_line(line)
                if parsed and parsed['item_name']:
                    actual_category, note_correct = determine_category(parsed['item_name'], parsed.get('unit'), auto_correct=auto_correct)
                    notes = []
                    if add_parsing_notes:
                        asrc = parsed.get('amount_source')
                        if asrc and asrc != 'last_token':
                            notes.append(f"amount_source={asrc}")
                        if parsed.get('qty') is not None and parsed.get('rate') is not None:
                            expected = round(parsed.get('qty') * parsed.get('rate'), 2)
                            if parsed.get('amount') is not None and abs(parsed.get('amount') - expected) > 0.1:
                                notes.append('amount_mismatch_qty_rate')
                        if note_correct:
                            notes.append(note_correct)
                    
                    rec = {
                        'date': None,
                        'txn_type': None,
                        'category': actual_category,
                        'header_amount': None,
                        'item_name': parsed['item_name'],
                        'qty': parsed['qty'],
                        'unit': parsed['unit'],
                        'rate': parsed['rate'],
                        'amount': parsed['amount'],
                        'bill_no': last_bill_no,
                        'raw_line': line,
                        'parsing_notes': ';'.join(notes) if notes else None
                    }
                    items.append(rec)
            i += 1

    df_items = pd.DataFrame(items)
    
    # Ensure category column exists
    if df_items.empty:
        df_items = pd.DataFrame(columns=['date','txn_type','category','header_amount','item_name','qty','unit','rate','amount','bill_no','raw_line'])
    
    # Clean item names
    df_items['item_name'] = df_items.get('item_name', pd.Series()).astype(str).str.strip()

    # Split by category
    eggs_df = df_items[df_items['category']=='egg'].copy()
    feeds_df = df_items[df_items['category']=='feed'].copy()
    meds_df = df_items[df_items['category']=='medicine'].copy()
    other_df = df_items[df_items['category']=='other'].copy()

    def agg(df):
        if df.empty:
            return pd.DataFrame(columns=['item_name','total_qty','unit','total_amount'])
        grouped = df.groupby(['item_name','unit'], dropna=False).agg(
            total_qty = ('qty', lambda x: sum([v for v in (x.fillna(0).tolist()) if v is not None])),
            total_amount = ('amount', lambda x: sum([v for v in (x.fillna(0).tolist()) if v is not None]))
        ).reset_index()
        return grouped

    eggs_agg = agg(eggs_df)
    feeds_agg = agg(feeds_df)
    meds_agg = agg(meds_df)
    other_agg = agg(other_df)

    payments_df = pd.DataFrame(payments)

    # Post-pass: ensure all 'To TDS' lines from the PDF were captured.
    # Build a date context map per line index so we can attach dates to any additional TDS entries.
    last_date = None
    line_date_by_idx = []
    for l in all_lines:
        m = DATE_HDR_RE.match(l.strip())
        if m:
            date_str = m.group(1)
            try:
                last_date = datetime.strptime(date_str, "%d-%b-%y")
            except:
                try:
                    last_date = datetime.strptime(date_str, "%d-%b-%Y")
                except:
                    last_date = None
        line_date_by_idx.append(last_date)

    for i, l in enumerate(all_lines):
        if 'TO TDS' in l.upper():
            if any(e.get('raw') == l for e in tds_entries):
                continue
            tokens = l.split()
            tidx, tkn = find_last_amount_token(tokens)
            amt = clean_number(tkn) if tkn else None
            if amt is not None:
                tds_entries.append({
                    'date': line_date_by_idx[i],
                    'amount': amt,
                    'bill_no': None,
                    'raw': l
                })
        # Also search for closing balance in lines (e.g., "To Closing Balance 4,05,455.15")
        if closing_balance is None and ('CLOSING BALANCE' in l.upper() or 'TO CLOSING BALANCE' in l.upper()):
            tokens = l.split()
            tidx, tkn = find_last_amount_token(tokens)
            amt = clean_number(tkn) if tkn else None
            if amt is not None:
                closing_balance = amt

    tds_df = pd.DataFrame(tds_entries)
    discounts_df = pd.DataFrame(discounts)

    def sum_or_zero(df):
        if df is None or df.empty:
            return 0.0
        return float(df['total_amount'].sum()) if 'total_amount' in df.columns else float(df['amount'].sum())

    total_eggs = sum_or_zero(eggs_agg)
    total_feeds = sum_or_zero(feeds_agg)
    total_meds = sum_or_zero(meds_agg)
    total_other = sum_or_zero(other_agg)
    total_payments = float(payments_df['amount'].sum()) if not payments_df.empty else 0.0
    total_tds = float(tds_df['amount'].sum()) if not tds_df.empty else 0.0
    total_discounts = float(discounts_df['amount'].sum()) if not discounts_df.empty else 0.0

    # Grand total = Eggs - Feed - Medicine - Other - TDS + Discount
    # Note: Payments are money received (reduce balance owed), not expenses, so they're not subtracted here
    grand_total = total_eggs - total_feeds - total_meds - total_other - total_tds + total_discounts
    
    summary = {
        'total_eggs': round(total_eggs, 2),
        'total_feeds': round(total_feeds, 2),
        'total_medicines': round(total_meds, 2),
        'total_other_items': round(total_other, 2),
        'total_payments': round(total_payments, 2),
        'total_tds': round(total_tds, 2),
        'total_discounts': round(total_discounts, 2),
        'grand_total': round(grand_total, 2)
    }

    # Net profit = total_eggs + total_discounts - total_feeds - total_medicine - total_other - total_tds
    net_profit = (total_eggs + total_discounts) - (total_feeds + total_meds + total_other + total_tds)
    summary['net_profit'] = round(net_profit, 2)
    summary['opening_balance'] = round(opening_balance,2) if opening_balance is not None else None
    summary['closing_balance'] = round(closing_balance,2) if closing_balance is not None else None
    
    # Validation: net_profit should equal closing_balance - opening_balance + payments
    if closing_balance is not None and opening_balance is not None:
        expected_net_profit = closing_balance - opening_balance + total_payments
        validation_diff = abs(net_profit - expected_net_profit)
        summary['validation_expected_net_profit'] = round(expected_net_profit, 2)
        summary['validation_difference'] = round(validation_diff, 2)

    

    return {
        'raw_items': df_items,
        'eggs_agg': eggs_agg,
        'feeds_agg': feeds_agg,
        'meds_agg': meds_agg,
        'other_agg': other_agg,
        'payments': payments_df,
        'tds': tds_df,
        'discounts': discounts_df,
        'summary': summary
    }

def export_to_excel(parsed, outpath):
    for key in ('raw_items', 'payments', 'tds', 'discounts'):
        df = parsed.get(key)
        if df is not None and not df.empty and 'date' in df.columns:
            df2 = df.copy()
            df2['date'] = df2['date'].apply(format_date_obj)
            parsed[key] = df2

    with pd.ExcelWriter(outpath, engine='openpyxl') as writer:
        parsed['raw_items'].to_excel(writer, sheet_name='raw_rows', index=False)
        parsed['eggs_agg'].to_excel(writer, sheet_name='eggs', index=False)
        parsed['feeds_agg'].to_excel(writer, sheet_name='feeds', index=False)
        parsed['meds_agg'].to_excel(writer, sheet_name='medicines', index=False)
        parsed['other_agg'].to_excel(writer, sheet_name='other_items', index=False)
        parsed['payments'].to_excel(writer, sheet_name='payments', index=False)
        parsed['tds'].to_excel(writer, sheet_name='tds', index=False)
        parsed['discounts'].to_excel(writer, sheet_name='discounts', index=False)
        summary_df = pd.DataFrame([{'metric': k, 'value': v} for k, v in parsed['summary'].items()])
        summary_df.to_excel(writer, sheet_name='summary', index=False)
        # Add ambiguous notes sheet if parsing notes exist
        raw = parsed.get('raw_items')
        if raw is not None and 'parsing_notes' in raw.columns:
            ambiguous = raw[raw['parsing_notes'].notna()]
            if not ambiguous.empty:
                ambiguous.to_excel(writer, sheet_name='ambiguous_rows', index=False)
    print(f"Excel exported to: {outpath}")

def main():
    parser = argparse.ArgumentParser(description='Parse a poultry ledger PDF into Excel')
    parser.add_argument('pdf', help='Path to PDF to parse')
    parser.add_argument('--auto-correct', action='store_true', help='Apply auto-correct heuristics for classification')
    parser.add_argument('--no-parsing-notes', dest='add_parsing_notes', action='store_false', help='Do not add parsing notes to raw rows')
    args = parser.parse_args()
    pdf = args.pdf
    if not os.path.isfile(pdf):
        print("File not found:", pdf)
        sys.exit(1)
    print("Parsing:", pdf)
    parsed = parse_pdf_statement(pdf, auto_correct=args.auto_correct, add_parsing_notes=args.add_parsing_notes)
    
    # Print summary for verification
    print("\n=== SUMMARY ===")
    for k, v in parsed['summary'].items():
        if isinstance(v, (int, float)) and v is not None:
            print(f"{k}: {format_indian_number(v)}")
        else:
            print(f"{k}: {v}")
    
    base = os.path.basename(pdf)
    out = os.path.join(os.path.dirname(pdf), f"parsed_{os.path.splitext(base)[0]}.xlsx")
    export_to_excel(parsed, out)
    print("Done.")

if __name__ == '__main__':
    main()