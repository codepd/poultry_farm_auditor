-- Simple SQL script to fix medicine items incorrectly categorized as FEED
-- Run this script using: psql your_database < fix_medicine_simple.sql
-- Or connect to your database and run these commands

-- Step 1: Check what medicine items are in FEED category
SELECT 
    COUNT(*) as count,
    SUM(amount) as total_amount,
    MIN(transaction_date) as earliest_date,
    MAX(transaction_date) as latest_date
FROM transactions
WHERE category = 'FEED'
    AND transaction_type IN ('PURCHASE', 'EXPENSE', 'SALE')
    AND (
        UPPER(item_name) LIKE '%D3%' OR
        UPPER(item_name) LIKE '%VETMULIN%' OR
        UPPER(item_name) LIKE '%OXYCYCLINE%' OR
        UPPER(item_name) LIKE '%TIAZIN%' OR
        UPPER(item_name) LIKE '%BPPS%' OR
        UPPER(item_name) LIKE '%CTC%' OR
        UPPER(item_name) LIKE '%SHELL GRIT%' OR
        UPPER(item_name) LIKE '%ROVIMIX%' OR
        UPPER(item_name) LIKE '%CHOLIMARIN%' OR
        UPPER(item_name) LIKE '%ZAGROMIN%' OR
        UPPER(item_name) LIKE '%G PRO NATURO%' OR
        UPPER(item_name) LIKE '%NECROVET%' OR
        UPPER(item_name) LIKE '%TOXOL%' OR
        UPPER(item_name) LIKE '%FRA C12%' OR
        UPPER(item_name) LIKE '%FRA C 12%' OR
        UPPER(item_name) LIKE '%CALCI%' OR
        UPPER(item_name) LIKE '%CALDLIV%' OR
        UPPER(item_name) LIKE '%RESPAFEED%' OR
        UPPER(item_name) LIKE '%VENTRIM%' OR
        UPPER(item_name) LIKE '%VITAL%' OR
        UPPER(item_name) LIKE '%MEDICINE%' OR
        UPPER(item_name) LIKE '%MEDIC%' OR
        UPPER(item_name) LIKE '%VITAMIN%' OR
        UPPER(item_name) LIKE '%SUPPLEMENT%' OR
        UPPER(item_name) LIKE '%GRIT%' OR
        UPPER(item_name) LIKE '%VET%' OR
        UPPER(item_name) LIKE '%NECRO%' OR
        UPPER(item_name) LIKE '%TOX%'
    );

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
WHERE category = 'FEED'
    AND transaction_type IN ('PURCHASE', 'EXPENSE', 'SALE')
    AND (
        UPPER(item_name) LIKE '%D3%' OR
        UPPER(item_name) LIKE '%VETMULIN%' OR
        UPPER(item_name) LIKE '%OXYCYCLINE%' OR
        UPPER(item_name) LIKE '%TIAZIN%' OR
        UPPER(item_name) LIKE '%BPPS%' OR
        UPPER(item_name) LIKE '%CTC%' OR
        UPPER(item_name) LIKE '%SHELL GRIT%' OR
        UPPER(item_name) LIKE '%ROVIMIX%' OR
        UPPER(item_name) LIKE '%CHOLIMARIN%' OR
        UPPER(item_name) LIKE '%ZAGROMIN%' OR
        UPPER(item_name) LIKE '%G PRO NATURO%' OR
        UPPER(item_name) LIKE '%NECROVET%' OR
        UPPER(item_name) LIKE '%TOXOL%' OR
        UPPER(item_name) LIKE '%FRA C12%' OR
        UPPER(item_name) LIKE '%FRA C 12%' OR
        UPPER(item_name) LIKE '%CALCI%' OR
        UPPER(item_name) LIKE '%CALDLIV%' OR
        UPPER(item_name) LIKE '%RESPAFEED%' OR
        UPPER(item_name) LIKE '%VENTRIM%' OR
        UPPER(item_name) LIKE '%VITAL%' OR
        UPPER(item_name) LIKE '%MEDICINE%' OR
        UPPER(item_name) LIKE '%MEDIC%' OR
        UPPER(item_name) LIKE '%VITAMIN%' OR
        UPPER(item_name) LIKE '%SUPPLEMENT%' OR
        UPPER(item_name) LIKE '%GRIT%' OR
        UPPER(item_name) LIKE '%VET%' OR
        UPPER(item_name) LIKE '%NECRO%' OR
        UPPER(item_name) LIKE '%TOX%'
    )
ORDER BY transaction_date DESC
LIMIT 20;

-- Step 3: UPDATE the medicine items (UNCOMMENT TO RUN)
-- BEGIN;
-- 
-- UPDATE transactions
-- SET category = 'MEDICINE'
-- WHERE category = 'FEED'
--     AND transaction_type IN ('PURCHASE', 'EXPENSE')
--     AND (
--         UPPER(item_name) LIKE '%D3%' OR
--         UPPER(item_name) LIKE '%VETMULIN%' OR
--         UPPER(item_name) LIKE '%OXYCYCLINE%' OR
--         UPPER(item_name) LIKE '%TIAZIN%' OR
--         UPPER(item_name) LIKE '%BPPS%' OR
--         UPPER(item_name) LIKE '%CTC%' OR
--         UPPER(item_name) LIKE '%SHELL GRIT%' OR
--         UPPER(item_name) LIKE '%ROVIMIX%' OR
--         UPPER(item_name) LIKE '%CHOLIMARIN%' OR
--         UPPER(item_name) LIKE '%ZAGROMIN%' OR
--         UPPER(item_name) LIKE '%G PRO NATURO%' OR
--         UPPER(item_name) LIKE '%NECROVET%' OR
--         UPPER(item_name) LIKE '%TOXOL%' OR
--         UPPER(item_name) LIKE '%FRA C12%' OR
--         UPPER(item_name) LIKE '%FRA C 12%' OR
--         UPPER(item_name) LIKE '%CALCI%' OR
--         UPPER(item_name) LIKE '%CALDLIV%' OR
--         UPPER(item_name) LIKE '%RESPAFEED%' OR
--         UPPER(item_name) LIKE '%VENTRIM%' OR
--         UPPER(item_name) LIKE '%VITAL%' OR
--         UPPER(item_name) LIKE '%MEDICINE%' OR
--         UPPER(item_name) LIKE '%MEDIC%' OR
--         UPPER(item_name) LIKE '%VITAMIN%' OR
--         UPPER(item_name) LIKE '%SUPPLEMENT%' OR
--         UPPER(item_name) LIKE '%GRIT%' OR
--         UPPER(item_name) LIKE '%VET%' OR
--         UPPER(item_name) LIKE '%NECRO%' OR
--         UPPER(item_name) LIKE '%TOX%'
--     );
-- 
-- -- Verify the update
-- SELECT 
--     COUNT(*) as updated_count,
--     SUM(amount) as total_amount
-- FROM transactions
-- WHERE category = 'MEDICINE'
--     AND transaction_type IN ('PURCHASE', 'EXPENSE')
--     AND (
--         UPPER(item_name) LIKE '%D3%' OR
--         UPPER(item_name) LIKE '%VETMULIN%' OR
--         UPPER(item_name) LIKE '%OXYCYCLINE%' OR
--         UPPER(item_name) LIKE '%TIAZIN%' OR
--         UPPER(item_name) LIKE '%BPPS%' OR
--         UPPER(item_name) LIKE '%CTC%' OR
--         UPPER(item_name) LIKE '%SHELL GRIT%' OR
--         UPPER(item_name) LIKE '%ROVIMIX%' OR
--         UPPER(item_name) LIKE '%CHOLIMARIN%' OR
--         UPPER(item_name) LIKE '%ZAGROMIN%' OR
--         UPPER(item_name) LIKE '%G PRO NATURO%' OR
--         UPPER(item_name) LIKE '%NECROVET%' OR
--         UPPER(item_name) LIKE '%TOXOL%' OR
--         UPPER(item_name) LIKE '%FRA C12%' OR
--         UPPER(item_name) LIKE '%FRA C 12%' OR
--         UPPER(item_name) LIKE '%CALCI%' OR
--         UPPER(item_name) LIKE '%CALDLIV%' OR
--         UPPER(item_name) LIKE '%RESPAFEED%' OR
--         UPPER(item_name) LIKE '%VENTRIM%' OR
--         UPPER(item_name) LIKE '%VITAL%' OR
--         UPPER(item_name) LIKE '%MEDICINE%' OR
--         UPPER(item_name) LIKE '%MEDIC%' OR
--         UPPER(item_name) LIKE '%VITAMIN%' OR
--         UPPER(item_name) LIKE '%SUPPLEMENT%' OR
--         UPPER(item_name) LIKE '%GRIT%' OR
--         UPPER(item_name) LIKE '%VET%' OR
--         UPPER(item_name) LIKE '%NECRO%' OR
--         UPPER(item_name) LIKE '%TOX%'
--     );
-- 
-- COMMIT;

