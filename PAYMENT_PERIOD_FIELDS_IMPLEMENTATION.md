# Payment Period Fields Implementation

## Overview
Added support for tracking payment dates and payment periods for expenses. This allows recording when payments are made and which period (month/week/days) they cover.

## Changes Made

### 1. Database Migration
**File**: `add_payment_period_fields.sql`

Added four new columns to the `transactions` table:
- `payment_date` (DATE, nullable) - Date when payment was made (defaults to `transaction_date`)
- `period_month` (DATE, nullable) - Month the payment is for (stored as first day of month, defaults to payment_date's month)
- `period_week` (INTEGER, nullable) - Week number within the payment period (optional)
- `period_days` (INTEGER, nullable) - Number of days the payment covers (optional)

The migration:
- Adds the new columns
- Sets default values for existing records (payment_date = transaction_date, period_month = transaction_date's month)
- Adds documentation comments

### 2. Go Model Updates
**File**: `go_api/models/transaction.go`

Added four new fields to the `Transaction` struct:
- `PaymentDate *time.Time` - Date when payment was made
- `PeriodMonth *time.Time` - Month the payment is for
- `PeriodWeek *int` - Week number (optional)
- `PeriodDays *int` - Number of days (optional)

### 3. API Request/Response Updates
**File**: `go_api/handlers/transactions.go`

- **TransactionCreateRequest**: Added the four new fields
- **Helper Function**: Added `setPaymentPeriodDefaults()` to set defaults:
  - `payment_date` defaults to `transaction_date` if not provided
  - `period_month` defaults to the first day of `payment_date`'s month if not provided
- **GetTransactions**: Updated SELECT query and Scan to include new fields
- **GetTransaction**: Updated SELECT query and Scan to include new fields
- **CreateTransaction**: Updated INSERT queries to include new fields with defaults
- **UpdateTransaction**: Updated UPDATE query to include new fields

## Usage

### Creating an Expense with Payment Period

When creating a transaction (especially expenses), you can now specify:

```json
{
  "transaction_date": "2025-01-15T00:00:00Z",
  "payment_date": "2025-02-10T00:00:00Z",  // When payment was made (optional, defaults to transaction_date)
  "period_month": "2025-01-01T00:00:00Z",  // Month the payment is for (optional, defaults to payment_date month)
  "period_week": 2,                         // Week number (optional)
  "period_days": 30,                        // Number of days (optional)
  "transaction_type": "PURCHASE",
  "category": "OTHER",
  "amount": 5000.00,
  "item_name": "Electricity Bill"
}
```

### Default Behavior

- If `payment_date` is not provided, it defaults to `transaction_date`
- If `period_month` is not provided, it defaults to the first day of `payment_date`'s month
- `period_week` and `period_days` are optional and can be null

### Electricity Bills

For electricity bills (paid on 10th or before 15th of the following month):
- Set `payment_date` to the actual payment date (e.g., 2025-02-10)
- Set `period_month` to the month the bill is for (e.g., 2025-01-01 for January's bill)

Example:
```json
{
  "transaction_date": "2025-02-10T00:00:00Z",
  "payment_date": "2025-02-10T00:00:00Z",  // Paid on 10th
  "period_month": "2025-01-01T00:00:00Z",  // For January's bill
  "transaction_type": "PURCHASE",
  "category": "OTHER",
  "amount": 15000.00,
  "item_name": "Electricity Bill"
}
```

## Migration Steps

1. Run the migration SQL:
   ```bash
   docker exec -i poultry_farm_db_new psql -U postgres -d poultry_farm < add_payment_period_fields.sql
   ```
   
   Or manually:
   ```bash
   psql -U postgres -d poultry_farm -f add_payment_period_fields.sql
   ```

2. Restart the Go backend to pick up the changes:
   ```bash
   ./restart_backend.sh
   ```

## Notes

- All new fields are nullable/optional
- Existing transactions will have defaults set (payment_date = transaction_date, period_month = transaction_date's month)
- The frontend will need to be updated to support these new fields in the expense entry form
- Analytics queries may need to be updated if you want to filter/report by payment periods
