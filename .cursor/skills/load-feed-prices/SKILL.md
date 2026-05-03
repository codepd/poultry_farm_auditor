---
name: load-feed-prices
description: Load feed prices from Everest Feeds rate sheet images into the AWS RDS database. Use when the user wants to load feed prices, process feed price images, update feed rates, or mentions feed price history images.
---

# Load Feed Prices to Database

Process Everest Feeds rate sheet images (JPEG/PNG) using OCR and load extracted prices into the AWS RDS PostgreSQL database.

## Prerequisites

- Tesseract OCR installed (`brew install tesseract`)
- Python venv at `python_backend/venv/` with `pytesseract`, `pillow`, `psycopg2` installed
- AWS CLI configured with profile `AdminAccess-Pradeep`

## Image Location

Feed price images are stored in `feed_price_history/<YEAR>/` with naming convention `MM_DD_YYYY.jpeg` (e.g., `02_16_2026.jpeg` for Feb 16, 2026).

## Step-by-Step Workflow

### 1. Identify images to process

Check which images exist in `feed_price_history/<YEAR>/` and confirm with the user which dates to load.

### 2. Start SSM port-forward tunnel to RDS

The RDS is in a private subnet. Connect through the bastion host using SSM:

```bash
aws ssm start-session \
  --target i-0b27dddea8b9b649d \
  --document-name AWS-StartPortForwardingSessionToRemoteHost \
  --parameters '{"host":["poultry-farm-dev.cx68k0g04oac.ap-south-1.rds.amazonaws.com"],"portNumber":["5432"],"localPortNumber":["15432"]}' \
  --profile AdminAccess-Pradeep \
  --region ap-south-1
```

Run this in the background (`block_until_ms: 0`). Wait ~5 seconds, then verify the tunnel is up by reading the terminal output (look for "Waiting for connections...").

### 3. Process each image

Run the processing script with RDS connection env vars:

```bash
cd python_backend && \
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm \
DB_USER=poultry_admin DB_PASSWORD='Kolipannai2025!' \
venv/bin/python cli/process_feed_price_images.py <image_path> --tenant-name "Pradeep Farm"
```

Replace `<image_path>` with the relative path from `python_backend/`, e.g., `../feed_price_history/2026/02_16_2026.jpeg`.

### 4. Verify OCR accuracy

**CRITICAL**: After each image is processed, visually inspect the image using the Read tool and compare:

1. **Date**: The OCR-extracted date vs the actual date in the image. The image uses DD.MM.YY format. The filename uses MM_DD_YYYY format. These may differ (e.g., filename `02_24_2026` but image shows `23.02.26` meaning Feb 23).
2. **Prices**: Verify all 4 feed prices match what's visible in the image:
   - Layer Mash
   - Pre-Layer Mash
   - Grower Mash
   - Chick Mash

**Known OCR issues**:
- "Layer" can be misread as "jtayer" or "Ltayer", causing the script to miss Layer Mash and instead duplicate the Pre-Layer Mash price (since "LAYER MASH" is a substring of "PRE-LAYER MASH").
- Date field may not be detected if OCR adds spaces between digits.

### 5. Fix incorrect data

If prices or dates are wrong, correct them directly:

```python
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
        UPDATE price_history SET price = <correct_price>
        WHERE tenant_id = %s AND price_date = '<date>' AND item_name = '<FEED_NAME>' AND price_type = 'FEED'
    ''', (tenant_id,))
    print(f'Updated {cursor.rowcount} rows')
"
```

### 6. Verify final state

Query the database to confirm all prices are correct:

```python
cursor.execute('''
    SELECT item_name, price, price_date FROM price_history
    WHERE tenant_id = %s AND price_type = 'FEED' AND price_date >= '<start_date>'
    ORDER BY price_date, item_name
''', (tenant_id,))
```

## Database Details

- **Table**: `price_history`
- **Columns**: `tenant_id`, `price_date`, `price_type` ('FEED'), `item_name`, `price`
- **Tenant**: Pradeep Farm (`8d7939f7-b716-4eb0-98d4-544c18c8dfb8`)
- **Feed types**: LAYER MASH, PRE-LAYER MASH, GROWER MASH, CHICK MASH

## Infrastructure Reference

| Resource | Value |
|----------|-------|
| RDS Endpoint | `poultry-farm-dev.cx68k0g04oac.ap-south-1.rds.amazonaws.com` |
| Bastion Instance | `i-0b27dddea8b9b649d` |
| AWS Profile | `AdminAccess-Pradeep` |
| AWS Region | `ap-south-1` |
| Local tunnel port | `15432` |
