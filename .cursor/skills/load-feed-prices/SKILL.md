---
name: load-feed-prices
description: Load feed prices from Sakthi/Everest weekly rate cards or monthly daily-rate screenshots into the AWS RDS database. Use when the user wants to load feed prices, process feed price images, update feed rates, validate prices already in the DB, or mentions feed price history images.
---

# Load Feed Prices to Database

Load feed rates into AWS RDS PostgreSQL `price_history`. Two image types exist. A monthly daily grid is the source of truth for every date it covers. A weekly rate card is used only for dates that monthly grid does not cover.

## Provider Preference

- **Primary**: Sakthi Poultry Feeds (matches the farm ledger provider).
- **Fallback**: Everest Feeds — use only when a Sakthi sheet is missing for that date range.
- Never use a future sheet. The price for a transaction date is the latest known rate on or before that date.
- Ledger rates are typically ₹/kg. Sheet rates are **per bag**. Divide the bag rate by **75** only when comparing to a ledger ₹/kg rate.
- The database stores the **bag rate** (typically 2000–3200), the same unit as the weekly cards. Do not store the ₹/kg figure.

## Prerequisites

- Tesseract OCR installed (`brew install tesseract`) — weekly cards only
- Python venv at `python_backend/venv/` with `pytesseract`, `pillow`, `psycopg2` installed
- AWS CLI configured with profile `AdminAccess-Pradeep`
- Sign-in and the RDS tunnel are `scripts/aws-sso-ensure.sh` and `scripts/start-rds-port-forward.sh`. Runbook: `scripts/README_RDS_TUNNEL.md`. The ledger skill uses the same tunnel.

## Image Location

`feed_price_history/<YEAR>/`

| Kind | Filename | Example | What it is |
|------|----------|---------|------------|
| Monthly daily grid | `MM_DD_DD_YYYY.jpeg` | `07_01_31_2026.jpeg` | One row per calendar day for that month |
| Weekly rate card | `MM_DD_YYYY.jpeg` | `06_01_2026.jpeg` | One date, four bag rates |

Monthly filename parts are month, first day, last day, year. `07_01_31_2026` is 1–31 Jul 2026. It is not three dates.

## Monthly daily grid (preferred)

Screenshot of a spreadsheet. Header is the month (`Jul-26`, `Aug-26`). Columns:

| Date | LM | PLM | GM | CM |
|------|----|-----|----|----|

Map columns to `item_name`:

- LM → `LAYER MASH`
- PLM → `PRE-LAYER MASH`
- GM → `GROWER MASH`
- CM → `CHICK MASH`

Each body cell is that day's **bag rate**. Footer rows are checksums only. Do not insert them:

- A totals row (sum of the daily bag rates)
- **Per bag** — average of the daily bag rates (total ÷ number of days)
- **Per Kg** — Per bag ÷ 75

These sheets do not name Sakthi or Everest. For dates on the grid, the grid wins over any weekly card.

### How to read and check

The weekly OCR script cannot parse this grid. Do **not** run `cli/process_feed_price_images.py` on a monthly image. It will collapse the month into one date and one price.

1. Read the image with the Read tool and transcribe every day.
2. Group runs of identical rates so a misread day stands out.
3. Before any write, check all four columns:
   - sum of daily rates = the totals row
   - total ÷ day count = Per bag (2–3 decimals)
   - Per bag ÷ 75 = Per Kg
4. If a checksum fails, re-read the image. Do not write.
5. Write one `price_history` row per day per feed (`price_type = 'FEED'`). Upsert on `(tenant_id, price_date, price_type, item_name)`.
6. Write only dates printed on the sheet. Do not copy the last day into the next month.

### If those dates are already in the DB

Weekly loads used to store the card date and then **carry that bag rate forward** for days with no card. Compare the monthly grid to those rows before changing anything.

1. Select existing `FEED` rows for the sheet's first and last date.
2. Report three counts per feed: already correct, missing, different.
3. List each different day as `date, item, db price → sheet price`.
4. A DB price under 500 is a ₹/kg value stored by mistake. The sheet bag rate replaces it; say so in the report.
5. Insert missing days and update different days. Leave matching days unchanged.
6. Re-query and confirm the written prices match the sheet.

### Months with no monthly file

Do not invent a daily grid from weekly cards when the user asked for the monthly screenshot. Name the expected file (`04_01_30_2026.jpeg`, `05_01_31_2026.jpeg`, `06_01_30_2026.jpeg`) and wait for it. Weekly cards for that month stay as they are until the grid arrives.

## Weekly rate card (only where no monthly grid covers the date)

Sakthi or Everest header, one `DATE`, and four lines: Layer Mash, Pre-Layer Mash, Grower Mash, Chick Mash, each `NNNN/-` per bag.

OCR with:

```bash
cd python_backend && \
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm \
DB_USER=poultry_admin DB_PASSWORD='Kolipannai2025!' \
venv/bin/python cli/process_feed_price_images.py <image_path> --tenant-name "Pradeep Farm"
```

`<image_path>` is relative to `python_backend/`, e.g. `../feed_price_history/2026/06_01_2026.jpeg`.

After OCR, open the image and check:

1. **Date.** The card is `DD.MM.YY` or `DD/Mon/YY`. The filename is `MM_DD_YYYY`. They can disagree (filename `02_24_2026`, card `23.02.26` = 23 Feb). Store the date printed on the card.
2. **All four prices.** Known OCR miss: "Layer" read as "jtayer" / "Ltayer" drops Layer Mash and copies the Pre-Layer price, because `LAYER MASH` is inside `PRE-LAYER MASH`.

Correct a bad row:

```bash
cd python_backend && \
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm \
DB_USER=poultry_admin DB_PASSWORD='Kolipannai2025!' \
venv/bin/python -c "
import sys
sys.path.insert(0, '.')
from database.connection import get_db_cursor

tenant_id = '8d7939f7-b716-4eb0-98d4-544c18c8dfb8'
with get_db_cursor() as cursor:
    cursor.execute('''
        UPDATE price_history SET price = %s
        WHERE tenant_id = %s AND price_date = %s AND item_name = %s AND price_type = 'FEED'
    ''', (<correct_price>, tenant_id, '<YYYY-MM-DD>', '<FEED_NAME>'))
    print(f'Updated {cursor.rowcount} rows')
"
```

### Filling days that have no card

Cards are usually Monday. A card's bag rate applies to:

- its own week: the card date through the next 6 days
- the following week too, when that week has no card

Stop the day before the next card. Do not carry more than one extra week. If the gap is longer than 14 days, fill those 14 days and ask the user about the rest. Never apply a later card backward.

Skip any day a monthly grid already covers.

## Step-by-step

### 1. Choose files

List `feed_price_history/<YEAR>/`. For each requested month, use `MM_DD_DD_YYYY.jpeg` when it exists. Otherwise use weekly `MM_DD_YYYY.jpeg` files inside that month, and say that the monthly screenshot is missing.

### 2. Refresh AWS login, then start the SSM tunnel

Use the shared tunnel script. It checks the SSO session, opens the approval page in the Chrome profile named **codepd** (`Profile 8`, codepd3@gmail.com) when the token is expired, then forwards RDS to localhost. Details and the "portal opened but the CLI is still waiting" case are in `scripts/README_RDS_TUNNEL.md`.

```bash
scripts/start-rds-port-forward.sh
```

Run in the background (`block_until_ms: 0`). Wait until the terminal shows `Waiting for connections...`. If port 15432 is busy, rerun with `LOCAL_PORT=15433` and keep that port for every later command.

IAM Identity Center cannot be renewed from a saved password. The AWS CLI stores the session token in `~/.aws/sso/cache` (mode 600). Do not copy an IdP password into the repo, the skill, or the shell history. Do not use `open -n` or `--new-window`; a second Chrome instance cannot attach to the codepd profile while it is already open.

The bastion is `kolipannai-dev-bastion` (`i-0a5c2678494e93072`). The old id `i-0b27dddea8b9b649d` does not exist. Another instance in the account is online and is not the bastion; do not pick "first online instance".

### 3. Load

Monthly grid for a month already transcribed in `scripts/load_monthly_feed_prices.py`: run that script. It checks the screenshot column totals, prints unchanged / missing / different, and writes only with `--apply`.

```bash
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm \
DB_USER=poultry_admin DB_PASSWORD='Kolipannai2025!' \
python_backend/venv/bin/python scripts/load_monthly_feed_prices.py

DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm \
DB_USER=poultry_admin DB_PASSWORD='Kolipannai2025!' \
python_backend/venv/bin/python scripts/load_monthly_feed_prices.py --apply
```

A month that is not in that script yet: read the image, verify the footer checksums, add the bands and expected totals to the script, then run it. Do not reuse prices from an old chat if the file on disk is newer.

Weekly card, only for dates the monthly grid does not cover: OCR script, then visual check, then forward-fill only uncovered days.

Upsert pattern:

```bash
cd python_backend && \
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm \
DB_USER=poultry_admin DB_PASSWORD='Kolipannai2025!' \
venv/bin/python -c "
import sys
sys.path.insert(0, '.')
from database.connection import get_db_cursor

tenant_id = '8d7939f7-b716-4eb0-98d4-544c18c8dfb8'
# (date, item_name, bag_price) transcribed from the sheet after checksums passed
rows = [
    ('2026-07-01', 'LAYER MASH', 2490),
]
with get_db_cursor() as cursor:
    for price_date, item_name, price in rows:
        cursor.execute('''
            INSERT INTO price_history (tenant_id, price_date, price_type, item_name, price)
            VALUES (%s, %s, 'FEED', %s, %s)
            ON CONFLICT (tenant_id, price_date, price_type, item_name)
            DO UPDATE SET price = EXCLUDED.price
            WHERE price_history.price IS DISTINCT FROM EXCLUDED.price
        ''', (tenant_id, price_date, item_name, price))
"
```

Build `rows` from the image you just checked. Do not reuse a table pasted in an old chat if the file on disk is newer.

### 4. Verify

```sql
SELECT price_date, item_name, price
FROM price_history
WHERE tenant_id = '8d7939f7-b716-4eb0-98d4-544c18c8dfb8'
  AND price_type = 'FEED'
  AND price_date BETWEEN '<start>' AND '<end>'
ORDER BY price_date, item_name
```

For a monthly grid, row count must be `days × 4`, and each price must match the sheet.

## Database

- **Table**: `price_history`
- **Columns**: `tenant_id`, `price_date`, `price_type` (`FEED`), `item_name`, `price` (bag rate)
- **Tenant**: Pradeep Farm (`8d7939f7-b716-4eb0-98d4-544c18c8dfb8`)
- **Feed types**: `LAYER MASH`, `PRE-LAYER MASH`, `GROWER MASH`, `CHICK MASH`
- **Unique key**: `(tenant_id, price_date, price_type, item_name)`

## Infrastructure

| Resource | Value |
|----------|-------|
| RDS Endpoint | `poultry-farm-dev.cx68k0g04oac.ap-south-1.rds.amazonaws.com` |
| Bastion Instance | `i-0a5c2678494e93072` |
| AWS Profile | `AdminAccess-Pradeep` |
| AWS Region | `ap-south-1` |
| Local tunnel port | `15432` |
