-- Check for medicine items that are incorrectly categorized as FEED
-- This will help us identify what needs to be fixed

SELECT 
    id,
    tenant_id,
    transaction_date,
    item_name,
    category,
    transaction_type,
    amount,
    quantity,
    unit
FROM transactions
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
ORDER BY transaction_date DESC, tenant_id;

-- Also check total amounts
SELECT 
    tenant_id,
    EXTRACT(YEAR FROM transaction_date) as year,
    EXTRACT(MONTH FROM transaction_date) as month,
    COUNT(*) as medicine_count,
    SUM(amount) as total_medicine_amount
FROM transactions
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
GROUP BY tenant_id, EXTRACT(YEAR FROM transaction_date), EXTRACT(MONTH FROM transaction_date)
ORDER BY year DESC, month DESC, tenant_id;

