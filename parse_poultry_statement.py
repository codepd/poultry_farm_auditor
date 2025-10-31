#!/usr/bin/env python3
"""
Parse poultry ledger PDF statements (like 'PRADEEP PF JULY 25.pdf') and export a structured Excel workbook.

Updated:
 - formats exported date fields as '2-Jun-25'
 - grand_total = Eggs - Feed - Medicine - Payments - TDS + Discount
"""

import sys
import re
import os
import pdfplumber
import pandas as pd
from datetime import datetime

# --- configuration / known keywords (tweak if you want) ---
EGG_KEYWORDS = ["LARGE EGG", "MEDIUM EGG", "SMALL EGG", "CORRECT EGG"]
FEED_KEYWORDS = ["LAYER MASH", "GROWER MASH", "PRE LAYER MASH", "LAYER MASH BULK", "GROWER MASH BULK", "PRE LAYER MASH BULK"]
MEDICINES_KNOWN = ["D3 FORTE", "VETMULIN", "OXYCYCLINE", "TIAZIN", "BPPS FORTE", "CTC", "SHELL GRIT",
                   "ROVIMIX", "CHOLIMARIN", "ZAGROMIN", "G PRO NATURO", "NECROVET", "TOXOL", "FRA C12 DRY", "CALCI ROYAL FS", "CALDLIV FS", "RESPAFEED", "VENTRIM (VITAL)"]

NUM_RE = re.compile(r'[\d,]+\.\d+|[\d,]+')
DATE_HDR_RE = re.compile(r'^(\d{1,2}-[A-Za-z]{3}-\d{2,4})\b')

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

def parse_item_line(line):
    text = line.strip()
    tokens = text.split()
    idx_amount, amount_tok = find_last_number_token(tokens)
    if idx_amount is None:
        return None
    amount = clean_number(amount_tok)
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
    return {
        'item_name': item_name_norm,
        'qty': qty,
        'unit': (unit.upper() if unit else None),
        'rate': rate,
        'amount': amount
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

def parse_pdf_statement(pdf_path):
    items = []
    payments = []
    tds_entries = []
    discounts = []
    
    def infer_txn_type_from_name(item_name):
        """Return a txn_type hint based on item_name keywords or None."""
        if not item_name:
            return None
        n = item_name.upper()
        for e in EGG_KEYWORDS:
            if e in n:
                return 'egg_purchase'
        for f in FEED_KEYWORDS:
            if f in n:
                return 'feed_sale'
        for m in MEDICINES_KNOWN:
            if m in n:
                return 'medicine'
        # loose heuristics
        if any(x in n for x in ['GRIT','D3','FRA','VET','CTC','NECRO','TOX','ROVIMIX']):
            return 'medicine'
        return None
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

            j = i + 1
            while j < len(all_lines):
                nxt = all_lines[j].strip()
                if DATE_HDR_RE.match(nxt):
                    break
                bsearch = re.search(r'BILLNO\s*(\d+)|BILL\s*NO[:\s]*(\d+)|BILL NO (\d+)', nxt.upper())
                if bsearch:
                    last_bill_no = bsearch.group(1) or bsearch.group(2) or bsearch.group(3)
                if re.search(r'\bTO\s+TDS\b', nxt.upper()) or nxt.upper().startswith("TO TDS"):
                    num = re.findall(NUM_RE, nxt)
                    amt = clean_number(num[0]) if num else None
                    tds_entries.append({
                        'date': parsed_date,
                        'amount': amt,
                        'bill_no': last_bill_no,
                        'raw': nxt
                    })
                if (' KGS' in nxt.upper()) or (' NOS' in nxt.upper()) or ('/KGS' in nxt.upper()) or ('/NOS' in nxt.upper()):
                    parsed = parse_item_line(nxt)
                    if parsed:
                        rec = {
                            'date': parsed_date,
                            'txn_type': txn_type,
                            'header_amount': header_amt,
                            'item_name': parsed['item_name'],
                            'qty': parsed['qty'],
                            'unit': parsed['unit'],
                            'rate': parsed['rate'],
                            'amount': parsed['amount'],
                            'bill_no': last_bill_no,
                            'raw_line': nxt
                        }
                        # Infer txn_type from the parsed item name and override
                        # the header txn_type when a clear match exists. This
                        # fixes cases where headers are noisy (e.g. 'feed_sale')
                        # but the item is clearly an egg or feed item.
                        inferred = infer_txn_type_from_name(rec.get('item_name'))
                        if inferred:
                            rec['txn_type'] = inferred
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
                if parsed:
                    rec = {
                        'date': None,
                        'txn_type': None,
                        'header_amount': None,
                        'item_name': parsed['item_name'],
                        'qty': parsed['qty'],
                        'unit': parsed['unit'],
                        'rate': parsed['rate'],
                        'amount': parsed['amount'],
                        'bill_no': last_bill_no,
                        'raw_line': line
                    }
                    inferred = infer_txn_type_from_name(rec.get('item_name'))
                    if inferred:
                        rec['txn_type'] = inferred
                    items.append(rec)
            i += 1

    df_items = pd.DataFrame(items)
    df_items['item_name'] = df_items.get('item_name', pd.Series()).astype(str).str.strip()

    def classify_row(name):
        if not name or name == 'NONE':
            return 'other'
        n = name.upper()
        for e in EGG_KEYWORDS:
            if e in n:
                return 'egg'
        for f in FEED_KEYWORDS:
            if f in n:
                return 'feed'
        for m in MEDICINES_KNOWN:
            if m in n:
                return 'medicine'
        if any(x in n for x in ['GRIT','D3','FRA','VET','CTC','NECRO','TOX','ROVIMIX']):
            return 'medicine'
        return 'other'

    if not df_items.empty:
        df_items['category'] = df_items['item_name'].apply(classify_row)
    else:
        df_items = pd.DataFrame(columns=['date','txn_type','header_amount','item_name','qty','unit','rate','amount','bill_no','raw_line','category'])

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

    # NEW: grand total per your formula
    # Grand total = Eggs - Feed - Medicine - Payments - TDS + Discount
    grand_total = total_eggs - total_feeds - total_meds - total_payments - total_tds + total_discounts
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

    # Net profit according to requested formula:
    # net_profit = total_eggs + total_discounts - total_feeds - total_medicine - total_other - total_tds
    net_profit = (total_eggs + total_discounts) - (total_feeds + total_meds + total_other + total_tds)
    summary['net_profit'] = round(net_profit, 2)

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
    print(f"Excel exported to: {outpath}")

def main():
    if len(sys.argv) < 2:
        print("Usage: python parse_poultry_statement.py path/to/statement.pdf")
        sys.exit(1)
    pdf = sys.argv[1]
    if not os.path.isfile(pdf):
        print("File not found:", pdf)
        sys.exit(1)
    print("Parsing:", pdf)
    parsed = parse_pdf_statement(pdf)
    base = os.path.basename(pdf)
    out = os.path.join(os.path.dirname(pdf), f"parsed_{os.path.splitext(base)[0]}.xlsx")
    export_to_excel(parsed, out)
    print("Done.")

if __name__ == '__main__':
    main()
