-- Migration: Add payment period fields to transactions table
-- This adds fields to track payment dates and periods for expenses

-- Add payment_date column (nullable, will default to transaction_date in application logic)
ALTER TABLE transactions 
ADD COLUMN IF NOT EXISTS payment_date DATE;

-- Add period_month column (which month the payment is for, stored as first day of month)
ALTER TABLE transactions 
ADD COLUMN IF NOT EXISTS period_month DATE;

-- Add period_week column (which week of the period, optional)
ALTER TABLE transactions 
ADD COLUMN IF NOT EXISTS period_week INTEGER;

-- Add period_days column (number of days the payment covers, optional)
ALTER TABLE transactions 
ADD COLUMN IF NOT EXISTS period_days INTEGER;

-- Set default payment_date to transaction_date for existing records
UPDATE transactions 
SET payment_date = transaction_date 
WHERE payment_date IS NULL;

-- Set default period_month to transaction_date's month for existing records
UPDATE transactions 
SET period_month = DATE_TRUNC('month', transaction_date)::DATE 
WHERE period_month IS NULL;

-- Add comments for documentation
COMMENT ON COLUMN transactions.payment_date IS 'Date when payment was made. Defaults to transaction_date if not specified.';
COMMENT ON COLUMN transactions.period_month IS 'Month the payment is for (stored as first day of month). Defaults to payment_date month if not specified.';
COMMENT ON COLUMN transactions.period_week IS 'Week number within the payment period (optional).';
COMMENT ON COLUMN transactions.period_days IS 'Number of days the payment covers (optional).';
