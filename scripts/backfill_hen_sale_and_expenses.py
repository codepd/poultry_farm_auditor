#!/usr/bin/env python3
"""
Backfill recurring expenses, one-time expenses, and seed hen batch sale data.

Defaults match current planning assumptions:
- Recurring from Jan 2025:
  - Labor: 80,000
  - EMI: 290,000
  - Electricity: 12,000
- One-time on 2025-04-01:
  - Sourcing growers: 6,000,000
  - Chick growing cost: 2,500,000
- Batch 1 sale seed:
  - 2026-04-11, 6000 hens, Rs 90 per hen
"""

from __future__ import annotations

import argparse
import datetime as dt
import os
import sys
from decimal import Decimal
from typing import Iterable, List, Optional, Tuple

import psycopg2
from psycopg2.extras import DictCursor


TENANT_ID = "8d7939f7-b716-4eb0-98d4-544c18c8dfb8"
DEFAULT_BATCH_NAME_HINT = "Batch 1"

RECURRING_ITEMS = [
    ("LABOR", Decimal("80000.00")),
    ("EMI", Decimal("290000.00")),
    ("ELECTRICITY", Decimal("12000.00")),
]

ONE_TIME_ITEMS = [
    ("SOURCING GROWERS (ONE-TIME)", Decimal("6000000.00"), dt.date(2025, 4, 1)),
    ("CHICK GROWING COST (ONE-TIME)", Decimal("2500000.00"), dt.date(2025, 4, 1)),
]

SALE_SEED = {
    "sale_date": dt.date(2026, 4, 11),
    "count": 6000,
    "price_per_hen": Decimal("90.00"),
    "notes": "Seeded from planning assumptions; update when actual values are confirmed.",
}


def month_starts(start: dt.date, end: dt.date) -> Iterable[dt.date]:
    cursor = dt.date(start.year, start.month, 1)
    while cursor <= end:
        yield cursor
        if cursor.month == 12:
            cursor = dt.date(cursor.year + 1, 1, 1)
        else:
            cursor = dt.date(cursor.year, cursor.month + 1, 1)


def db_connect():
    params = {
        "host": os.environ.get("DB_HOST", "localhost"),
        "port": os.environ.get("DB_PORT", "5432"),
        "dbname": os.environ.get("DB_NAME", "poultry_farm"),
        "user": os.environ.get("DB_USER", "poultry_admin"),
        "password": os.environ.get("DB_PASSWORD", ""),
        "cursor_factory": DictCursor,
    }
    return psycopg2.connect(**params)


def ensure_transaction(
    cur,
    *,
    tenant_id: str,
    transaction_date: dt.date,
    transaction_type: str,
    category: str,
    item_name: str,
    amount: Decimal,
    notes: Optional[str] = None,
    quantity: Decimal = Decimal("1"),
    unit: str = "NOS",
    rate: Optional[Decimal] = None,
    payment_date: Optional[dt.date] = None,
    period_month: Optional[dt.date] = None,
) -> bool:
    rate = rate if rate is not None else amount
    payment_date = payment_date or transaction_date
    period_month = period_month or dt.date(transaction_date.year, transaction_date.month, 1)

    cur.execute(
        """
        SELECT id
        FROM transactions
        WHERE tenant_id = %s
          AND transaction_date = %s
          AND transaction_type = %s
          AND category = %s
          AND item_name = %s
          AND amount = %s
        LIMIT 1
        """,
        (tenant_id, transaction_date, transaction_type, category, item_name, amount),
    )
    if cur.fetchone():
        return False

    cur.execute(
        """
        INSERT INTO transactions (
            tenant_id, transaction_date, transaction_type, category,
            item_name, quantity, unit, rate, amount, notes,
            payment_date, period_month
        )
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        """,
        (
            tenant_id,
            transaction_date,
            transaction_type,
            category,
            item_name,
            quantity,
            unit,
            rate,
            amount,
            notes,
            payment_date,
            period_month,
        ),
    )
    return True


def resolve_batch(cur, tenant_id: str, batch_name_hint: str) -> Tuple[int, str]:
    cur.execute(
        """
        SELECT id, batch_name
        FROM hen_batches
        WHERE tenant_id = %s
          AND batch_name ILIKE %s
        ORDER BY date_added
        LIMIT 1
        """,
        (tenant_id, f"%{batch_name_hint}%"),
    )
    row = cur.fetchone()
    if row:
        return int(row["id"]), row["batch_name"]

    cur.execute(
        """
        SELECT id, batch_name
        FROM hen_batches
        WHERE tenant_id = %s
        ORDER BY date_added
        LIMIT 1
        """,
        (tenant_id,),
    )
    fallback = cur.fetchone()
    if not fallback:
        raise RuntimeError("No hen batches found for tenant; cannot seed sale data.")
    return int(fallback["id"]), fallback["batch_name"]


def ensure_hen_sale(cur, tenant_id: str, batch_id: int, batch_name: str) -> Tuple[bool, bool]:
    sale_date = SALE_SEED["sale_date"]
    count = SALE_SEED["count"]
    price_per_hen = SALE_SEED["price_per_hen"]
    notes = SALE_SEED["notes"]
    total_amount = price_per_hen * Decimal(count)

    cur.execute(
        """
        SELECT id
        FROM hen_batch_sales
        WHERE batch_id = %s
          AND sale_date = %s
          AND count = %s
          AND price_per_hen = %s
        LIMIT 1
        """,
        (batch_id, sale_date, count, price_per_hen),
    )
    sale_inserted = False
    if not cur.fetchone():
        cur.execute(
            """
            INSERT INTO hen_batch_sales (
                batch_id, sale_date, count, price_per_hen, total_amount, notes
            )
            VALUES (%s, %s, %s, %s, %s, %s)
            """,
            (batch_id, sale_date, count, price_per_hen, total_amount, notes),
        )
        sale_inserted = True

    income_inserted = ensure_transaction(
        cur,
        tenant_id=tenant_id,
        transaction_date=sale_date,
        transaction_type="INCOME",
        category="OTHER",
        item_name=f"HEN BATCH SALE - {batch_name}",
        amount=total_amount,
        notes=notes,
        quantity=Decimal(count),
        unit="NOS",
        rate=price_per_hen,
        payment_date=sale_date,
        period_month=dt.date(sale_date.year, sale_date.month, 1),
    )
    return sale_inserted, income_inserted


def main() -> int:
    parser = argparse.ArgumentParser(description="Backfill hen sale and expense assumptions.")
    parser.add_argument("--tenant-id", default=TENANT_ID)
    parser.add_argument("--batch-name-hint", default=DEFAULT_BATCH_NAME_HINT)
    parser.add_argument("--start-month", default="2025-01", help="Recurring expenses start month (YYYY-MM)")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    start_year, start_month = [int(x) for x in args.start_month.split("-")]
    recurring_start = dt.date(start_year, start_month, 1)
    recurring_end = dt.date.today().replace(day=1)

    conn = db_connect()
    conn.autocommit = False

    counters = {
        "recurring_inserted": 0,
        "one_time_inserted": 0,
        "sale_event_inserted": 0,
        "sale_income_inserted": 0,
    }

    try:
        with conn.cursor() as cur:
            # Recurring entries
            for month_start in month_starts(recurring_start, recurring_end):
                for item_name, amount in RECURRING_ITEMS:
                    inserted = ensure_transaction(
                        cur,
                        tenant_id=args.tenant_id,
                        transaction_date=month_start,
                        transaction_type="EXPENSE",
                        category="OTHER",
                        item_name=item_name,
                        amount=amount,
                        notes=f"Backfilled recurring {item_name}",
                        payment_date=month_start,
                        period_month=month_start,
                    )
                    if inserted:
                        counters["recurring_inserted"] += 1

            # One-time entries
            for item_name, amount, txn_date in ONE_TIME_ITEMS:
                inserted = ensure_transaction(
                    cur,
                    tenant_id=args.tenant_id,
                    transaction_date=txn_date,
                    transaction_type="EXPENSE",
                    category="OTHER",
                    item_name=item_name,
                    amount=amount,
                    notes="Backfilled one-time historical cost",
                    payment_date=txn_date,
                    period_month=dt.date(txn_date.year, txn_date.month, 1),
                )
                if inserted:
                    counters["one_time_inserted"] += 1

            batch_id, batch_name = resolve_batch(cur, args.tenant_id, args.batch_name_hint)
            sale_inserted, income_inserted = ensure_hen_sale(cur, args.tenant_id, batch_id, batch_name)
            if sale_inserted:
                counters["sale_event_inserted"] += 1
            if income_inserted:
                counters["sale_income_inserted"] += 1

        if args.dry_run:
            conn.rollback()
            print("Dry run complete. No changes committed.")
        else:
            conn.commit()
            print("Backfill committed successfully.")

        print("Summary:")
        for key, value in counters.items():
            print(f"  {key}: {value}")

        return 0
    except Exception as exc:
        conn.rollback()
        print(f"Backfill failed: {exc}", file=sys.stderr)
        return 1
    finally:
        conn.close()


if __name__ == "__main__":
    raise SystemExit(main())
