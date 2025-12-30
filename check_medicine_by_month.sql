-- Check medicine transactions by month to see what's actually in the database

SELECT 
    EXTRACT(YEAR FROM transaction_date)::int as year,
    EXTRACT(MONTH FROM transaction_date)::int as month,
    COUNT(*) as medicine_count,
    SUM(amount) as total_medicine_amount,
    transaction_type,
    STRING_AGG(DISTINCT item_name, ', ') as sample_items
FROM transactions
WHERE category = 'MEDICINE'
    AND transaction_type IN ('PURCHASE', 'EXPENSE', 'SALE')
    AND EXTRACT(YEAR FROM transaction_date) = 2025
GROUP BY 
    EXTRACT(YEAR FROM transaction_date),
    EXTRACT(MONTH FROM transaction_date),
    transaction_type
ORDER BY year, month, transaction_type;

-- Also check total medicine for 2025
SELECT 
    COUNT(*) as total_count,
    SUM(amount) as total_amount,
    MIN(transaction_date) as earliest,
    MAX(transaction_date) as latest
FROM transactions
WHERE category = 'MEDICINE'
    AND transaction_type IN ('PURCHASE', 'EXPENSE', 'SALE')
    AND EXTRACT(YEAR FROM transaction_date) = 2025;

