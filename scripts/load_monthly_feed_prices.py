#!/usr/bin/env python3
"""Upsert daily feed bag rates transcribed from monthly screenshots.

Each band is inclusive. Prices are rupees per bag, matching weekly rate cards.
The footer averages on the screenshots are checksums and are not stored.

Usage (tunnel must already be listening):
    DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm \\
    DB_USER=poultry_admin DB_PASSWORD='...' \\
    python_backend/venv/bin/python scripts/load_monthly_feed_prices.py --dry-run

    ... same env ... scripts/load_monthly_feed_prices.py --apply
"""

import argparse
import os
import sys
from datetime import datetime, timedelta
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "python_backend"))

from database.connection import get_db_cursor  # noqa: E402

TENANT_ID = "8d7939f7-b716-4eb0-98d4-544c18c8dfb8"
FEEDS = ("LAYER MASH", "PRE-LAYER MASH", "GROWER MASH", "CHICK MASH")

# (start, end, layer, pre-layer, grower, chick) bag rates.
# Column sums were checked against the Per bag / Per Kg footer on each sheet.
BANDS = {
    "2026-04": [
        ("2026-04-01", "2026-04-05", 2390, 2400, 2230, 2600),
        ("2026-04-06", "2026-04-12", 2360, 2370, 2260, 2630),
        ("2026-04-13", "2026-04-19", 2180, 2190, 2260, 2630),
        ("2026-04-20", "2026-04-26", 2120, 2130, 2260, 2630),
        ("2026-04-27", "2026-04-30", 2180, 2190, 2260, 2630),
    ],
    "2026-05": [
        ("2026-05-01", "2026-05-03", 2180, 2190, 2260, 2630),
        ("2026-05-04", "2026-05-10", 2240, 2250, 2320, 2710),
        ("2026-05-11", "2026-05-17", 2300, 2310, 2320, 2710),
        ("2026-05-18", "2026-05-24", 2390, 2400, 2500, 2930),
        ("2026-05-25", "2026-05-31", 2480, 2490, 2600, 3080),
    ],
    "2026-06": [
        ("2026-06-01", "2026-06-14", 2460, 2470, 2510, 2960),
        ("2026-06-15", "2026-06-28", 2400, 2410, 2520, 2960),
        ("2026-06-29", "2026-06-30", 2450, 2460, 2570, 3010),
    ],
    "2026-07": [
        ("2026-07-01", "2026-07-05", 2490, 2500, 2610, 3050),
        ("2026-07-06", "2026-07-12", 2610, 2620, 2610, 3050),
        ("2026-07-13", "2026-07-26", 2540, 2550, 2660, 3100),
        ("2026-07-27", "2026-07-31", 2720, 2730, 2720, 3160),
    ],
    "2026-08": [
        ("2026-08-01", "2026-08-01", 2720, 2730, 2720, 3160),
        ("2026-08-02", "2026-08-09", 2870, 2880, 2720, 3160),
        ("2026-08-10", "2026-08-16", 2750, 2760, 2650, 3160),
        ("2026-08-17", "2026-08-23", 2540, 2550, 2650, 3160),
        ("2026-08-24", "2026-08-30", 2630, 2640, 2650, 3160),
        ("2026-08-31", "2026-08-31", 2550, 2560, 2600, 3110),
    ],
    "2026-09": [
        ("2026-09-01", "2026-09-13", 2550, 2560, 2600, 3110),
        ("2026-09-14", "2026-09-20", 2490, 2500, 2600, 3110),
        ("2026-09-21", "2026-09-27", 2550, 2560, 2600, 3110),
        ("2026-09-28", "2026-09-30", 2750, 2760, 2600, 3110),
    ],
}

# Sum of daily bag rates printed on each screenshot.
EXPECTED_TOTALS = {
    "2026-04": (67290, 67590, 67650, 78750),
    "2026-05": (72410, 72720, 74960, 87900),
    "2026-06": (72940, 73240, 75560, 88900),
    "2026-07": (79880, 80190, 82160, 95800),
    "2026-08": (83670, 83980, 82730, 97910),
    "2026-09": (76680, 76980, 78000, 93300),
}


def parse_day(value):
    return datetime.strptime(value, "%Y-%m-%d").date()


def expand(month_key):
    rows = []
    for start, end, *prices in BANDS[month_key]:
        day = parse_day(start)
        last = parse_day(end)
        while day <= last:
            for feed, price in zip(FEEDS, prices):
                rows.append((day, feed, price))
            day += timedelta(days=1)
    totals = [0, 0, 0, 0]
    for _, feed, price in rows:
        totals[FEEDS.index(feed)] += price
    expected = EXPECTED_TOTALS[month_key]
    if tuple(totals) != expected:
        raise SystemExit(
            f"{month_key} totals {tuple(totals)} do not match the screenshot {expected}"
        )
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Write inserts and updates. Without this flag, only print the diff.",
    )
    parser.add_argument(
        "--months",
        default="2026-04,2026-05,2026-06,2026-07,2026-08,2026-09",
        help="Comma-separated YYYY-MM keys",
    )
    args = parser.parse_args()
    months = [item.strip() for item in args.months.split(",") if item.strip()]

    wanted = []
    for month in months:
        if month not in BANDS:
            raise SystemExit(f"No transcribed bands for {month}")
        wanted.extend(expand(month))

    start = min(day for day, _, _ in wanted)
    end = max(day for day, _, _ in wanted)
    print(f"Sheet rows: {len(wanted)} ({start} through {end})")

    with get_db_cursor() as cursor:
        cursor.execute(
            """
            SELECT price_date, item_name, price
            FROM price_history
            WHERE tenant_id = %s
              AND price_type = 'FEED'
              AND price_date BETWEEN %s AND %s
              AND item_name = ANY(%s)
            """,
            (TENANT_ID, start, end, list(FEEDS)),
        )
        existing = {
            (row[0], row[1]): float(row[2]) for row in cursor.fetchall()
        }

        inserts = []
        updates = []
        unchanged = 0
        for day, feed, price in wanted:
            current = existing.get((day, feed))
            if current is None:
                inserts.append((day, feed, price))
            elif abs(current - price) > 0.001:
                updates.append((day, feed, current, price))
            else:
                unchanged += 1

        print(f"Unchanged: {unchanged}")
        print(f"Missing (insert): {len(inserts)}")
        print(f"Different (update): {len(updates)}")
        for day, feed, old, new in updates:
            print(f"  {day} {feed}: {old:g} -> {new}")
        if inserts:
            by_month = {}
            for day, feed, _ in inserts:
                by_month.setdefault(day.strftime("%Y-%m"), set()).add(day)
            for month, days in sorted(by_month.items()):
                print(f"  insert {month}: {len(days)} days x 4 feeds")

        if not args.apply:
            print("Dry run only. Re-run with --apply to write.")
            return

        for day, feed, price in inserts:
            cursor.execute(
                """
                INSERT INTO price_history
                    (tenant_id, price_date, price_type, item_name, price)
                VALUES (%s, %s, 'FEED', %s, %s)
                ON CONFLICT (tenant_id, price_date, price_type, item_name)
                DO UPDATE SET price = EXCLUDED.price
                """,
                (TENANT_ID, day, feed, price),
            )
        for day, feed, _, price in updates:
            cursor.execute(
                """
                UPDATE price_history
                SET price = %s
                WHERE tenant_id = %s
                  AND price_date = %s
                  AND price_type = 'FEED'
                  AND item_name = %s
                """,
                (price, TENANT_ID, day, feed),
            )
        print(f"Applied {len(inserts)} inserts and {len(updates)} updates.")


if __name__ == "__main__":
    missing = [name for name in ("DB_HOST", "DB_NAME", "DB_USER", "DB_PASSWORD") if not os.getenv(name)]
    if missing:
        raise SystemExit(f"Missing env: {', '.join(missing)}")
    main()
