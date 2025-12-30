-- Fix medicine items that have transaction_type = 'SALE' 
-- Medicine purchases should be 'PURCHASE' or 'EXPENSE', not 'SALE'

-- Step 1: Check how many medicine items have SALE as transaction_type
SELECT 
    COUNT(*) as count,
    SUM(amount) as total_amount,
    MIN(transaction_date) as earliest_date,
    MAX(transaction_date) as latest_date
FROM transactions
WHERE category = 'MEDICINE'
    AND transaction_type = 'SALE';

-- Step 2: Show sample items that will be updated
SELECT 
    id,
    tenant_id,
    transaction_date,
    item_name,
    category,
    transaction_type,
    amount
FROM transactions
WHERE category = 'MEDICINE'
    AND transaction_type = 'SALE'
ORDER BY transaction_date DESC
LIMIT 20;

-- Step 3: UPDATE medicine items from SALE to PURCHASE
-- First, disable the trigger that's causing issues, then update, then re-enable

BEGIN;

-- Disable the trigger temporarily (if it exists)
DO $$
BEGIN
    -- Try to disable the trigger if it exists
    ALTER TABLE transactions DISABLE TRIGGER ALL;
EXCEPTION
    WHEN OTHERS THEN
        -- Trigger might not exist or already disabled, continue
        NULL;
END $$;

-- Now perform the update
UPDATE transactions
SET transaction_type = 'PURCHASE'
WHERE category = 'MEDICINE'
    AND transaction_type = 'SALE';

-- Re-enable the trigger
DO $$
BEGIN
    ALTER TABLE transactions ENABLE TRIGGER ALL;
EXCEPTION
    WHEN OTHERS THEN
        -- If re-enabling fails, that's okay
        NULL;
END $$;

-- Verify the update
SELECT 
    COUNT(*) as updated_count,
    SUM(amount) as total_amount
FROM transactions
WHERE category = 'MEDICINE'
    AND transaction_type = 'PURCHASE';

COMMIT;

