#!/usr/bin/env python3
"""
Load tenant egg prices from NECC rate sheet.

Current behavior:
- Uses NECC zone average for each selected month (zone configurable per tenant).
- Stores LARGE EGG in rupees/egg (NECC value is per 100 eggs, so divide by 100).
- Derives MEDIUM/SMALL from LARGE:
  - MEDIUM = LARGE - 0.10
  - SMALL  = LARGE - 0.15

Future-friendly note:
- If pricing moves from monthly average to sale-day price, keep this script as
  a "reference price backfill" path and add a daily granularity loader.
"""

import argparse
import calendar
import datetime as dt
import html
import os
import re
import ssl
import sys
from typing import Optional, Tuple
from urllib import parse, request

import psycopg2


NECC_URL = "https://www.e2necc.com/home/eggprice"
DEFAULT_TENANT_NAME = "Pradeep Farm"
DEFAULT_ZONE = "Namakkal"


def month_iter(start_month: dt.date, end_month: dt.date):
    current = dt.date(start_month.year, start_month.month, 1)
    final = dt.date(end_month.year, end_month.month, 1)
    while current <= final:
        yield current
        if current.month == 12:
            current = dt.date(current.year + 1, 1, 1)
        else:
            current = dt.date(current.year, current.month + 1, 1)


def clean_text(value: str) -> str:
    value = html.unescape(value or "")
    value = re.sub(r"<[^>]+>", "", value)
    return " ".join(value.replace("\xa0", " ").split()).strip()


def normalize_zone(zone: str) -> str:
    return " ".join((zone or "").upper().split())


def fetch_zone_monthly_average(zone: str, year: int, month: int) -> Optional[float]:
    # Daily sheet contains the month day-wise table with trailing "Average" column.
    payload = parse.urlencode(
        {
            "ddlMonth": f"{month:02d}",
            "ddlYear": str(year),
            "rblReportType": "DailyReport",
            "btnReport": "Get Sheet",
        }
    ).encode("utf-8")
    req = request.Request(NECC_URL, data=payload, method="POST")
    req.add_header("Content-Type", "application/x-www-form-urlencoded")
    req.add_header("User-Agent", "Mozilla/5.0")

    # NECC cert chain occasionally fails in local python builds.
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    with request.urlopen(req, timeout=30, context=ctx) as resp:
        html_doc = resp.read().decode("utf-8", errors="ignore")

    # Locate rows and match target zone in first td.
    for row in re.findall(r"<tr[^>]*>(.*?)</tr>", html_doc, flags=re.I | re.S):
        cells = re.findall(r"<td[^>]*>(.*?)</td>", row, flags=re.I | re.S)
        if len(cells) < 2:
            continue
        zone_cell = clean_text(cells[0])
        if normalize_zone(zone_cell) != normalize_zone(zone):
            continue
        avg_cell = clean_text(cells[-1]).replace(",", "")
        if not re.match(r"^\d+(\.\d+)?$", avg_cell):
            return None
        return float(avg_cell)

    return None


def get_tenant(conn, tenant_name: str) -> Tuple[str, str]:
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT id::text, COALESCE(NULLIF(trim(egg_price_reference_zone), ''), %s)
            FROM tenants
            WHERE name = %s
            LIMIT 1
            """,
            (DEFAULT_ZONE, tenant_name),
        )
        row = cur.fetchone()
    if not row:
        raise RuntimeError(f"Tenant not found: {tenant_name}")
    return row[0], row[1]


def upsert_prices(conn, tenant_id: str, month_date: dt.date, large: float, dry_run: bool):
    medium = round(large - 0.10, 2)
    small = round(large - 0.15, 2)
    entries = [("LARGE EGG", large), ("MEDIUM EGG", medium), ("SMALL EGG", small)]

    if dry_run:
        return entries

    with conn.cursor() as cur:
        for item_name, price in entries:
            cur.execute(
                """
                INSERT INTO price_history (tenant_id, price_date, price_type, item_name, price)
                VALUES (%s, %s, 'EGG', %s, %s)
                ON CONFLICT (tenant_id, price_date, price_type, item_name)
                DO UPDATE SET price = EXCLUDED.price
                """,
                (tenant_id, month_date, item_name, price),
            )
    return entries


def parse_args():
    parser = argparse.ArgumentParser(
        description="Load NECC monthly egg prices for tenant zone into price_history."
    )
    parser.add_argument("--tenant-name", default=DEFAULT_TENANT_NAME)
    parser.add_argument(
        "--zone",
        default="",
        help="Override tenant egg_price_reference_zone (e.g., Namakkal, Hyderabad).",
    )
    parser.add_argument(
        "--from-month",
        default="2025-10",
        help="Start month YYYY-MM (inclusive), default 2025-10.",
    )
    parser.add_argument(
        "--to-month",
        default=dt.date.today().strftime("%Y-%m"),
        help="End month YYYY-MM (inclusive), default current month.",
    )
    parser.add_argument("--dry-run", action="store_true")
    return parser.parse_args()


def parse_month(s: str) -> dt.date:
    try:
        return dt.datetime.strptime(s, "%Y-%m").date().replace(day=1)
    except ValueError as exc:
        raise ValueError(f"Invalid month format '{s}', expected YYYY-MM") from exc


def main():
    args = parse_args()
    start = parse_month(args.from_month)
    end = parse_month(args.to_month)
    if end < start:
        raise ValueError("--to-month cannot be earlier than --from-month")

    conn = psycopg2.connect(
        host=os.environ.get("DB_HOST", "localhost"),
        port=int(os.environ.get("DB_PORT", "5432")),
        dbname=os.environ.get("DB_NAME", "poultry_farm"),
        user=os.environ.get("DB_USER", "poultry_admin"),
        password=os.environ.get("DB_PASSWORD", ""),
        sslmode=os.environ.get("DB_SSLMODE", "require"),
    )
    conn.autocommit = False
    try:
        tenant_id, tenant_zone = get_tenant(conn, args.tenant_name)
        zone = args.zone.strip() or tenant_zone or DEFAULT_ZONE
        print(f"Tenant: {args.tenant_name} ({tenant_id})")
        print(f"Zone: {zone}")
        print(f"Range: {start.strftime('%Y-%m')} -> {end.strftime('%Y-%m')}")
        if args.dry_run:
            print("Mode: DRY RUN")

        inserted = 0
        for month_date in month_iter(start, end):
            avg_per_100 = fetch_zone_monthly_average(zone, month_date.year, month_date.month)
            if avg_per_100 is None:
                month_name = calendar.month_abbr[month_date.month]
                print(f"SKIP {month_date.year}-{month_date.month:02d}: no NECC average for zone '{zone}' ({month_name})")
                continue

            large = round(avg_per_100 / 100.0, 2)
            entries = upsert_prices(conn, tenant_id, month_date, large, args.dry_run)
            inserted += len(entries)
            print(
                f"{month_date.year}-{month_date.month:02d}: LARGE={entries[0][1]:.2f}, "
                f"MEDIUM={entries[1][1]:.2f}, SMALL={entries[2][1]:.2f}"
            )

        if args.dry_run:
            conn.rollback()
            print(f"Dry run complete. Would upsert {inserted} rows.")
        else:
            conn.commit()
            print(f"Done. Upserted {inserted} rows.")
    finally:
        conn.close()


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(1)
