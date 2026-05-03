---
name: parse-ledger-pdf
description: Parse poultry ledger PDFs, verify parsed data via Excel, and load into AWS RDS database. Use when the user wants to parse a ledger, process a ledger PDF, verify ledger data, load ledger data to DB, or mentions ledger statements.
---

# Parse Ledger PDF with Price Validation (Feed + Egg) → Load to DB

End-to-end workflow for processing poultry ledger PDFs from feed providers into the database, while validating ledger rates against reference data already loaded in DB.

## IMPORTANT Workflow Contract

For each ledger month, follow this order strictly:

1. Ensure **egg monthly average prices** for that month are loaded in DB.
2. Ensure **feed prices for relevant dates** are loaded in DB.
3. Parse PDF to Excel.
4. Validate ledger feed/egg rates against reference prices (do not skip).
5. Load parsed ledger to DB only after validation is accepted.

## Required Skill Dependency

Before processing ledger feed rates, you MUST refer to and follow:

- `.cursor/skills/load-feed-prices/SKILL.md`

Use that skill to load/verify feed price images for all dates relevant to the ledger period.

## Prerequisites

- Python venv at `python_backend/venv/` with `pdfplumber`, `pandas`, `openpyxl`, `psycopg2`
- AWS CLI with profile `AdminAccess-Pradeep`
- Feed price images available in `feed_price_history/<YEAR>/`
- Ledger PDF available in `ledgers/<YEAR>_ledgers/`

## File Conventions

| Item | Pattern | Example |
|------|---------|---------|
| Ledger PDFs | `ledgers/<YEAR>_ledgers/<YY>_<MM>_<MONTH>.pdf` | `ledgers/2026_ledgers/26_03_MAR.pdf` |
| Second provider same month | append `_2` suffix | `26_03_MAR_2.pdf` |
| Parsed Excel output | same directory, prefixed `parsed_` | `parsed_26_03_MAR.xlsx` |

## Quick Command Checklist (Single Month)

Use this as a fast copy/paste runbook for one ledger month:

```bash
# 0) Inputs (set once per run)
MONTH=2026-03
PDF="ledgers/2026_ledgers/26_03_MAR.pdf"
TENANT_ID="8d7939f7-b716-4eb0-98d4-544c18c8dfb8"
DB_PORT=15432   # use 15433 if 15432 is busy

# 1) Start RDS tunnel (keep this terminal open)
aws ssm start-session \
  --target i-0b27dddea8b9b649d \
  --document-name AWS-StartPortForwardingSessionToRemoteHost \
  --parameters "{\"host\":[\"poultry-farm-dev.cx68k0g04oac.ap-south-1.rds.amazonaws.com\"],\"portNumber\":[\"5432\"],\"localPortNumber\":[\"$DB_PORT\"]}" \
  --profile AdminAccess-Pradeep \
  --region ap-south-1

# 2) Load monthly avg egg prices for this month (dry-run first)
DB_HOST=localhost DB_PORT=$DB_PORT DB_NAME=poultry_farm DB_USER=poultry_admin DB_PASSWORD='***' DB_SSLMODE=require \
python_backend/venv/bin/python scripts/load_necc_egg_prices.py \
  --tenant-name "Pradeep Farm" \
  --from-month $MONTH \
  --to-month $MONTH \
  --dry-run

# 3) Commit egg price load
DB_HOST=localhost DB_PORT=$DB_PORT DB_NAME=poultry_farm DB_USER=poultry_admin DB_PASSWORD='***' DB_SSLMODE=require \
python_backend/venv/bin/python scripts/load_necc_egg_prices.py \
  --tenant-name "Pradeep Farm" \
  --from-month $MONTH \
  --to-month $MONTH

# 4) Load feed prices for ledger period (follow load-feed-prices skill)
# Example (repeat for each image):
# cd python_backend && DB_HOST=localhost DB_PORT=$DB_PORT DB_NAME=poultry_farm DB_USER=poultry_admin DB_PASSWORD='***' \
# venv/bin/python cli/process_feed_price_images.py ../feed_price_history/2026/03_16_2026.jpeg --tenant-name "Pradeep Farm"

# 5) Parse ledger PDF -> Excel
python3 parse_poultry_statement.py "$PDF"

# 6) Validate parsed rates vs reference prices
python3 validate_prices.py "ledgers/2026_ledgers/parsed_26_03_MAR.xlsx" \
  --price-history "Poultry_Farm_Price_History_Details_2025.xlsx"

# 7) Load parsed ledger to DB (only after validation is acceptable)
cd python_backend && \
DB_HOST=localhost DB_PORT=$DB_PORT DB_NAME=poultry_farm DB_USER=poultry_admin DB_PASSWORD='***' \
venv/bin/python cli/process_ledger_pdf.py "../$PDF" --tenant-id "$TENANT_ID"
```

## Step-by-Step Workflow

### 1. Start SSM Tunnel to RDS

```bash
aws ssm start-session \
  --target i-0b27dddea8b9b649d \
  --document-name AWS-StartPortForwardingSessionToRemoteHost \
  --parameters '{"host":["poultry-farm-dev.cx68k0g04oac.ap-south-1.rds.amazonaws.com"],"portNumber":["5432"],"localPortNumber":["15432"]}' \
  --profile AdminAccess-Pradeep \
  --region ap-south-1
```

Run in background (`block_until_ms: 0`). If `15432` is busy, use `15433` and keep that port for all subsequent commands.

### 2. Load Monthly Avg Egg Prices for Ledger Month

Use the NECC loader script to ensure the ledger month exists in `price_history`:

```bash
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm DB_USER=poultry_admin DB_PASSWORD='***' DB_SSLMODE=require \
python_backend/venv/bin/python scripts/load_necc_egg_prices.py \
  --tenant-name "Pradeep Farm" \
  --from-month YYYY-MM \
  --to-month YYYY-MM
```

Use `--dry-run` first if needed.

### 3. Load Feed Prices for Ledger Period (MANDATORY)

Follow `.cursor/skills/load-feed-prices/SKILL.md` and load all feed price images needed for the ledger month and billing dates.

Validation rule: feed rate reference must come from the latest price on or before transaction date (never a future date).

### 4. Parse PDF → Generate Excel

```bash
cd <repo_root> && python3 parse_poultry_statement.py ledgers/<YEAR>_ledgers/<FILENAME>.pdf
```

This produces `parsed_<FILENAME>.xlsx` by default.

### 5. Verify Parse Output (Manual)

Open `parsed_*.xlsx` and verify:

- Summary: opening/closing balance, validation difference, provider
- Items: egg/feed/medicine/other rows, quantities, rates, amounts
- Payments/TDS/discounts: values and dates

If wrong, fix parser logic and re-run before any DB load.

### 6. Validate Ledger Rates Against Price Data

Run price validation before DB load:

```bash
python3 validate_prices.py \
  ledgers/<YEAR>_ledgers/parsed_<FILENAME>.xlsx \
  --price-history Poultry_Farm_Price_History_Details_2025.xlsx
```

Also validate directly against DB-loaded prices when needed (recommended for final check):

- Feed items: compare ledger rate to `price_history` FEED rate effective on/before transaction date.
- Egg items: compare ledger rate to monthly avg EGG rate for that month.

If mismatches are expected business exceptions, confirm explicitly with user before load.

### 7. Load Ledger to Database

```bash
cd python_backend && \
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm \
DB_USER=poultry_admin DB_PASSWORD='***' \
venv/bin/python cli/process_ledger_pdf.py \
  ../ledgers/<YEAR>_ledgers/<FILENAME>.pdf \
  --tenant-id 8d7939f7-b716-4eb0-98d4-544c18c8dfb8
```

This inserts/updates:

- `ledger_parses` (summary)
- `ledger_breakdowns` (aggregations)
- `transactions` (row-level items/payments/TDS/discounts)

### 8. Post-load Verification

Query latest ledger rows and compare totals with verified Excel and validation outputs. If totals drift, inspect `transactions` first and then derived ledger aggregates.

## Database Schema Reference

| Table | Purpose |
|-------|---------|
| `ledger_parses` | Monthly summary per PDF (balance, totals, provider_id FK) |
| `ledger_breakdowns` | Aggregated items (EGG_LARGE, FEED_LAYER MASH, etc.) |
| `transactions` | Row-level items, payments, TDS, discounts |
| `providers` | Feed provider/egg buyer entities (name, contact, email) |

## Infrastructure

| Resource | Value |
|----------|-------|
| Tenant (Pradeep Farm) | `8d7939f7-b716-4eb0-98d4-544c18c8dfb8` |
| RDS Endpoint | `poultry-farm-dev.cx68k0g04oac.ap-south-1.rds.amazonaws.com` |
| Bastion Instance | `i-0b27dddea8b9b649d` |
| AWS Profile | `AdminAccess-Pradeep` |
| AWS Region | `ap-south-1` |
| Local tunnel port | `15432` (fallback `15433`) |
