-- Fix medicine items that are incorrectly categorized as FEED
-- This will update the category from 'FEED' to 'MEDICINE' for items that match medicine keywords

BEGIN;

-- Update transactions that are medicine but categorized as FEED
UPDATE transactions
SET category = 'MEDICINE'
WHERE category = 'FEED'
    AND transaction_type IN ('PURCHASE', 'EXPENSE')
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

-- Show what was updated
SELECT 
    COUNT(*) as updated_count,
    SUM(amount) as total_amount_updated
FROM transactions
WHERE category = 'MEDICINE'
    AND transaction_type IN ('PURCHASE', 'EXPENSE')
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

-- If everything looks good, commit. Otherwise, rollback.
-- COMMIT;
-- ROLLBACK;

