# Hen Sale and Expense Backfill Runbook

This runbook covers:
- loading the planned backfill data,
- validating monthly numbers,
- correcting assumed values later (sale date/count/price).

## 1) Apply schema update (one time)

```bash
psql "$DATABASE_URL" -f add_hen_batch_sales.sql
```

If you use direct DB env vars:

```bash
PGPASSWORD="$DB_PASSWORD" psql \
  -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
  -f add_hen_batch_sales.sql
```

## 2) Run backfill script

Dry run first:

```bash
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm DB_USER=poultry_admin DB_PASSWORD='***' \
python3 scripts/backfill_hen_sale_and_expenses.py --dry-run
```

Commit changes:

```bash
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm DB_USER=poultry_admin DB_PASSWORD='***' \
python3 scripts/backfill_hen_sale_and_expenses.py
```

## 3) Validation queries

### 3.1 Batch sale events

```sql
SELECT hbs.id,
       hb.batch_name,
       hbs.sale_date,
       hbs.count,
       hbs.price_per_hen,
       hbs.total_amount,
       hbs.created_at
FROM hen_batch_sales hbs
JOIN hen_batches hb ON hb.id = hbs.batch_id
ORDER BY hbs.sale_date DESC, hbs.id DESC;
```

### 3.2 Batch current count after sales + mortality

```sql
SELECT hb.id,
       hb.batch_name,
       hb.initial_count,
       hb.current_count,
       COALESCE(m.total_mortality, 0) AS total_mortality,
       COALESCE(s.total_sold, 0) AS total_sold,
       (hb.initial_count - COALESCE(m.total_mortality, 0) - COALESCE(s.total_sold, 0)) AS expected_current_count
FROM hen_batches hb
LEFT JOIN (
  SELECT batch_id, SUM(count) AS total_mortality
  FROM hen_mortality
  GROUP BY batch_id
) m ON m.batch_id = hb.id
LEFT JOIN (
  SELECT batch_id, SUM(count) AS total_sold
  FROM hen_batch_sales
  GROUP BY batch_id
) s ON s.batch_id = hb.id
ORDER BY hb.id;
```

### 3.3 Seed sale income transaction

```sql
SELECT transaction_date,
       transaction_type,
       category,
       item_name,
       quantity,
       rate,
       amount
FROM transactions
WHERE transaction_type = 'INCOME'
  AND item_name ILIKE 'HEN BATCH SALE - %'
ORDER BY transaction_date DESC, id DESC;
```

### 3.4 Recurring expenses by month (Labor / EMI / Electricity)

```sql
SELECT date_trunc('month', transaction_date)::date AS month,
       SUM(CASE WHEN UPPER(item_name) LIKE '%LABOR%' THEN amount ELSE 0 END) AS labor,
       SUM(CASE WHEN UPPER(item_name) LIKE '%EMI%' THEN amount ELSE 0 END) AS emi,
       SUM(CASE WHEN UPPER(item_name) LIKE '%ELECTRICITY%' THEN amount ELSE 0 END) AS electricity
FROM transactions
WHERE transaction_type = 'EXPENSE'
  AND category = 'OTHER'
GROUP BY 1
ORDER BY 1 DESC;
```

### 3.5 One-time expenses

```sql
SELECT transaction_date,
       item_name,
       amount
FROM transactions
WHERE transaction_type = 'EXPENSE'
  AND category = 'OTHER'
  AND item_name ILIKE '%ONE-TIME%'
ORDER BY transaction_date DESC, id DESC;
```

## 4) Correct assumptions later

If the sale assumptions change, update both `hen_batch_sales` and its linked income transaction.

### 4.1 Update sale event

```sql
UPDATE hen_batch_sales
SET sale_date = DATE '2026-04-11',
    count = 6000,
    price_per_hen = 90,
    total_amount = 6000 * 90,
    notes = 'Updated with confirmed values'
WHERE id = <sale_event_id>;
```

### 4.2 Update corresponding income transaction

```sql
UPDATE transactions
SET transaction_date = DATE '2026-04-11',
    quantity = 6000,
    rate = 90,
    amount = 6000 * 90,
    payment_date = DATE '2026-04-11',
    period_month = DATE '2026-04-01',
    notes = 'Updated with confirmed values'
WHERE transaction_type = 'INCOME'
  AND item_name = 'HEN BATCH SALE - <batch_name>'
  AND amount = 6000 * 90;
```

After corrections, re-check:
- Hen batch current count reconciliation query.
- Monthly dashboard net profit and headcount views.
