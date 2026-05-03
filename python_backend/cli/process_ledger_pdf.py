#!/usr/bin/env python3
"""
Parse a poultry ledger PDF, validate totals, and load results into the database.
Stores:
- ledger_parses summary
- ledger_breakdowns (aggregated items + payments/tds/discounts)
- transactions (row-level items + payments/tds/discounts)
"""

import argparse
import os
import re
import sys
from datetime import date
from pathlib import Path

import pandas as pd

# Allow importing parse_poultry_statement.py from repo root
REPO_ROOT = Path(__file__).resolve().parents[2]
PY_BACKEND_ROOT = REPO_ROOT / "python_backend"
sys.path.insert(0, str(REPO_ROOT))
sys.path.insert(0, str(PY_BACKEND_ROOT))

from parse_poultry_statement import parse_pdf_statement, export_to_excel  # noqa: E402
from database.connection import get_db_connection  # noqa: E402


MONTH_MAP = {
    "JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
    "JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12
}


def derive_month_year(pdf_path: Path, parsed: dict):
    name = pdf_path.stem.upper()
    m = re.search(r"(\d{2})_(\d{2})_([A-Z]{3})", name)
    if m:
        yy = int(m.group(1))
        mm = int(m.group(2))
        return (2000 + yy, mm)

    # Fall back to the first available date in raw items/payments
    for key in ("raw_items", "payments", "tds", "discounts"):
        df = parsed.get(key)
        if df is not None and not df.empty and "date" in df.columns:
            first = df["date"].dropna().min()
            if pd.notna(first):
                return (int(first.year), int(first.month))
    raise ValueError("Unable to derive year/month from PDF or parsed data.")


def map_transaction_type(category: str) -> str:
    if category == "egg":
        return "SALE"
    if category == "feed":
        return "PURCHASE"
    if category == "medicine":
        return "PURCHASE"
    if category == "other":
        return "EXPENSE"
    return "EXPENSE"


def normalize_category(category: str) -> str:
    mapping = {
        "egg": "EGG",
        "feed": "FEED",
        "medicine": "MEDICINE",
        "other": "OTHER",
    }
    return mapping.get(category, "OTHER")


def coerce_date(value):
    if value is None or (isinstance(value, float) and pd.isna(value)):
        return None
    if hasattr(value, "date"):
        return value.date()
    parsed = pd.to_datetime(value, errors="coerce")
    if pd.isna(parsed):
        return None
    return parsed.date()


def coerce_nullable_number(value):
    if value is None:
        return None
    try:
        if pd.isna(value):
            return None
    except Exception:
        pass
    try:
        return float(value)
    except Exception:
        return None


def _sum_egg_qty(eggs_df, keyword):
    if eggs_df is None or eggs_df.empty:
        return None
    mask = eggs_df["item_name"].astype(str).str.upper().str.contains(keyword)
    if not mask.any():
        return None
    return float(eggs_df.loc[mask, "total_qty"].fillna(0).sum())


def _sum_feeds_qty(feeds_df):
    if feeds_df is None or feeds_df.empty:
        return None
    return float(feeds_df["total_qty"].fillna(0).sum())


def resolve_provider(cur, tenant_id, summary):
    """Find or create a provider from the parsed summary. Returns provider UUID."""
    provider_name = summary.get("feed_provider")
    if not provider_name:
        return None

    cur.execute(
        "SELECT id FROM providers WHERE tenant_id = %s AND name = %s",
        (tenant_id, provider_name),
    )
    row = cur.fetchone()
    if row:
        provider_id = row[0]
        cur.execute(
            """UPDATE providers SET contact = COALESCE(%s, contact),
               email = COALESCE(%s, email), updated_at = CURRENT_TIMESTAMP
               WHERE id = %s""",
            (summary.get("provider_contact"), summary.get("provider_email"), provider_id),
        )
        return provider_id

    cur.execute(
        """INSERT INTO providers (tenant_id, name, contact, email)
           VALUES (%s, %s, %s, %s) RETURNING id""",
        (tenant_id, provider_name, summary.get("provider_contact"), summary.get("provider_email")),
    )
    return cur.fetchone()[0]


def insert_ledger_parse(cur, tenant_id, pdf_name, year, month, summary, parsed):
    eggs_df = parsed.get("eggs_agg")
    feeds_df = parsed.get("feeds_agg")
    eggs_large_qty = _sum_egg_qty(eggs_df, "LARGE")
    eggs_medium_qty = _sum_egg_qty(eggs_df, "MEDIUM")
    eggs_small_qty = _sum_egg_qty(eggs_df, "SMALL")
    feeds_total_kg = _sum_feeds_qty(feeds_df)

    provider_id = resolve_provider(cur, tenant_id, summary)

    cur.execute(
        """
        SELECT id FROM ledger_parses
        WHERE tenant_id = %s AND year = %s AND month = %s AND pdf_filename = %s
        """,
        (tenant_id, year, month, pdf_name),
    )
    row = cur.fetchone()
    if row:
        ledger_id = row[0]
        cur.execute(
            """
            UPDATE ledger_parses
            SET parse_date = %s,
                opening_balance = %s,
                closing_balance = %s,
                total_eggs = %s,
                total_feeds = %s,
                total_medicines = %s,
                net_profit = %s,
                eggs_large_qty = %s,
                eggs_medium_qty = %s,
                eggs_small_qty = %s,
                feeds_total_kg = %s,
                provider_id = %s,
                account_holder = %s,
                ledger_period = %s
            WHERE id = %s
            """,
            (
                date.today(),
                summary.get("opening_balance"),
                summary.get("closing_balance"),
                summary.get("total_eggs"),
                summary.get("total_feeds"),
                summary.get("total_medicines"),
                summary.get("net_profit"),
                eggs_large_qty,
                eggs_medium_qty,
                eggs_small_qty,
                feeds_total_kg,
                provider_id,
                summary.get("account_holder"),
                summary.get("ledger_period"),
                ledger_id,
            ),
        )
        return ledger_id

    cur.execute(
        """
        INSERT INTO ledger_parses (
            tenant_id, pdf_filename, parse_date, month, year,
            opening_balance, closing_balance, total_eggs, total_feeds,
            total_medicines, net_profit, eggs_large_qty, eggs_medium_qty,
            eggs_small_qty, feeds_total_kg,
            provider_id, account_holder, ledger_period
        )
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        RETURNING id
        """,
        (
            tenant_id,
            pdf_name,
            date.today(),
            month,
            year,
            summary.get("opening_balance"),
            summary.get("closing_balance"),
            summary.get("total_eggs"),
            summary.get("total_feeds"),
            summary.get("total_medicines"),
            summary.get("net_profit"),
            eggs_large_qty,
            eggs_medium_qty,
            eggs_small_qty,
            feeds_total_kg,
            provider_id,
            summary.get("account_holder"),
            summary.get("ledger_period"),
        ),
    )
    return cur.fetchone()[0]


def insert_breakdowns(cur, ledger_parse_id, parsed):
    cur.execute("DELETE FROM ledger_breakdowns WHERE ledger_parse_id = %s", (ledger_parse_id,))
    rows = []

    def add_rows(df, prefix):
        if df is None or df.empty:
            return
        for _, row in df.iterrows():
            item = str(row.get("item_name", "")).strip()
            if not item:
                continue
            breakdown_type = f"{prefix}_{item.upper()}"
            qty = row.get("total_qty")
            if qty is None:
                continue
            rows.append((ledger_parse_id, breakdown_type, qty))

    add_rows(parsed.get("eggs_agg"), "EGG")
    add_rows(parsed.get("feeds_agg"), "FEED")
    add_rows(parsed.get("meds_agg"), "MEDICINE")
    add_rows(parsed.get("other_agg"), "OTHER")

    if rows:
        cur.executemany(
            """
            INSERT INTO ledger_breakdowns (ledger_parse_id, breakdown_type, quantity)
            VALUES (%s, %s, %s)
            """,
            rows,
        )


def insert_transactions(cur, tenant_id, parsed, import_note):
    cur.execute("DELETE FROM transactions WHERE tenant_id = %s AND notes = %s", (tenant_id, import_note))

    rows = []
    df_items = parsed.get("raw_items")
    if df_items is not None and not df_items.empty:
        for _, row in df_items.iterrows():
            txn_date = coerce_date(row.get("date"))
            if txn_date is None:
                continue
            category = str(row.get("category", "")).lower()
            rows.append(
                (
                    tenant_id,
                    txn_date,
                    map_transaction_type(category),
                    normalize_category(category),
                    row.get("item_name"),
                    coerce_nullable_number(row.get("qty")),
                    row.get("unit"),
                    coerce_nullable_number(row.get("rate")),
                    coerce_nullable_number(row.get("amount")),
                    import_note,
                )
            )

    def add_amount_rows(df, txn_type):
        if df is None or df.empty:
            return
        for _, row in df.iterrows():
            txn_date = coerce_date(row.get("date"))
            if txn_date is None:
                continue
            rows.append(
                (
                    tenant_id,
                    txn_date,
                    txn_type,
                    "OTHER",
                    txn_type.title(),
                    None,
                    None,
                    None,
                    coerce_nullable_number(row.get("amount")),
                    import_note,
                )
            )

    add_amount_rows(parsed.get("payments"), "PAYMENT")
    add_amount_rows(parsed.get("tds"), "TDS")
    add_amount_rows(parsed.get("discounts"), "DISCOUNT")

    if rows:
        cur.executemany(
            """
            INSERT INTO transactions (
                tenant_id, transaction_date, transaction_type, category,
                item_name, quantity, unit, rate, amount, notes
            )
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """,
            rows,
        )


def main():
    parser = argparse.ArgumentParser(description="Process ledger PDF and load into DB")
    parser.add_argument("pdf", help="Path to PDF file")
    parser.add_argument("--tenant-id", required=True, help="Tenant UUID")
    parser.add_argument("--out-dir", default="2025_ledgers_excel", help="Output directory for Excel")
    parser.add_argument("--auto-correct", action="store_true", help="Enable parser auto-correct")
    parser.add_argument("--no-parsing-notes", dest="add_parsing_notes", action="store_false")
    args = parser.parse_args()

    pdf_path = Path(args.pdf).resolve()
    if not pdf_path.exists():
        raise FileNotFoundError(f"PDF not found: {pdf_path}")

    parsed = parse_pdf_statement(str(pdf_path), auto_correct=args.auto_correct, add_parsing_notes=args.add_parsing_notes)
    year, month = derive_month_year(pdf_path, parsed)

    out_dir = Path(args.out_dir).resolve()
    out_dir.mkdir(parents=True, exist_ok=True)
    excel_path = out_dir / f"parsed_{pdf_path.stem}.xlsx"
    export_to_excel(parsed, str(excel_path))

    summary = parsed.get("summary", {})
    validation_diff = summary.get("validation_difference")
    print("Summary:", summary)
    if validation_diff is not None:
        print(f"Validation difference: {validation_diff}")

    conn = get_db_connection()
    try:
        cur = conn.cursor()
        ledger_parse_id = insert_ledger_parse(cur, args.tenant_id, pdf_path.name, year, month, summary, parsed)
        insert_breakdowns(cur, ledger_parse_id, parsed)
        insert_transactions(cur, args.tenant_id, parsed, f"Ledger import: {pdf_path.name}")
        conn.commit()
        print(f"✅ Loaded ledger into DB. ledger_parse_id={ledger_parse_id}")
        print(f"✅ Excel saved to {excel_path}")
    except Exception:
        conn.rollback()
        raise
    finally:
        cur.close()
        conn.close()


if __name__ == "__main__":
    main()
