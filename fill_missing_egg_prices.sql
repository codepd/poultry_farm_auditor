-- Fill missing egg monthly average prices based on Large egg prices from transactions
-- Large = Average price from transactions (calculated)
-- Medium = Large - 0.10 (10 paise)
-- Small = Large - 0.15 (15 paise)

-- This script calculates monthly averages from transactions and fills missing values
-- It ensures LARGE EGG prices are also inserted into price_history table

DO $$
DECLARE
    tenant_uuid UUID;
    large_egg_record RECORD;
    medium_price DECIMAL(12, 2);
    small_price DECIMAL(12, 2);
    month_start_date DATE;
    inserted_count INTEGER := 0;
BEGIN
    -- Get tenant UUID (assuming "Pradeep Farm" or first tenant)
    SELECT id INTO tenant_uuid 
    FROM tenants 
    WHERE name = 'Pradeep Farm' 
    LIMIT 1;
    
    IF tenant_uuid IS NULL THEN
        SELECT id INTO tenant_uuid FROM tenants LIMIT 1;
    END IF;
    
    IF tenant_uuid IS NULL THEN
        RAISE EXCEPTION 'No tenant found!';
    END IF;
    
    RAISE NOTICE 'Using tenant: %', tenant_uuid;
    RAISE NOTICE 'Filling missing Medium and Small egg prices...';
    RAISE NOTICE '';
    
    -- Loop through each month that has Large egg transactions
    FOR large_egg_record IN
        SELECT 
            EXTRACT(YEAR FROM transaction_date)::INTEGER as year,
            EXTRACT(MONTH FROM transaction_date)::INTEGER as month,
            SUM(amount) as total_amount,
            SUM(quantity) as total_quantity,
            CASE 
                WHEN SUM(quantity) > 0 THEN SUM(amount) / SUM(quantity)
                ELSE 0
            END as average_price
        FROM transactions
        WHERE tenant_id = tenant_uuid
            AND category = 'EGG'
            AND transaction_type = 'SALE'
            AND item_name LIKE '%LARGE%EGG%'
            AND quantity IS NOT NULL
            AND quantity > 0
        GROUP BY EXTRACT(YEAR FROM transaction_date), EXTRACT(MONTH FROM transaction_date)
        ORDER BY year DESC, month DESC
    LOOP
        month_start_date := TO_DATE(
            large_egg_record.year::TEXT || '-' || 
            LPAD(large_egg_record.month::TEXT, 2, '0') || '-01',
            'YYYY-MM-DD'
        );
        
        -- Calculate Medium and Small prices
        medium_price := large_egg_record.average_price - 0.10;
        small_price := large_egg_record.average_price - 0.15;
        
        -- First, ensure LARGE EGG price exists for this month
        IF NOT EXISTS (
            SELECT 1 FROM price_history
            WHERE tenant_id = tenant_uuid
                AND price_type = 'EGG'
                AND item_name = 'LARGE EGG'
                AND EXTRACT(YEAR FROM price_date) = large_egg_record.year
                AND EXTRACT(MONTH FROM price_date) = large_egg_record.month
        ) THEN
            -- Insert Large egg price
            INSERT INTO price_history (tenant_id, price_date, price_type, item_name, price)
            SELECT tenant_uuid, month_start_date, 'EGG', 'LARGE EGG', large_egg_record.average_price
            WHERE NOT EXISTS (
                SELECT 1 FROM price_history
                WHERE tenant_id = tenant_uuid
                    AND price_date = month_start_date
                    AND price_type = 'EGG'
                    AND item_name = 'LARGE EGG'
            );
            
            RAISE NOTICE 'Inserted LARGE EGG: %-%: ₹%',
                large_egg_record.year,
                large_egg_record.month,
                large_egg_record.average_price;
            inserted_count := inserted_count + 1;
        END IF;
        
        -- Check if Medium egg price exists for this month
        IF NOT EXISTS (
            SELECT 1 FROM price_history
            WHERE tenant_id = tenant_uuid
                AND price_type = 'EGG'
                AND item_name = 'MEDIUM EGG'
                AND EXTRACT(YEAR FROM price_date) = large_egg_record.year
                AND EXTRACT(MONTH FROM price_date) = large_egg_record.month
        ) THEN
            -- Insert Medium egg price (using INSERT ... SELECT to avoid constraint issues)
            INSERT INTO price_history (tenant_id, price_date, price_type, item_name, price)
            SELECT tenant_uuid, month_start_date, 'EGG', 'MEDIUM EGG', medium_price
            WHERE NOT EXISTS (
                SELECT 1 FROM price_history
                WHERE tenant_id = tenant_uuid
                    AND price_date = month_start_date
                    AND price_type = 'EGG'
                    AND item_name = 'MEDIUM EGG'
            );
            
            RAISE NOTICE 'Inserted MEDIUM EGG: %-%: ₹% (based on LARGE EGG: ₹%)',
                large_egg_record.year,
                large_egg_record.month,
                medium_price,
                large_egg_record.average_price;
            inserted_count := inserted_count + 1;
        END IF;
        
        -- Check if Small egg price exists for this month
        IF NOT EXISTS (
            SELECT 1 FROM price_history
            WHERE tenant_id = tenant_uuid
                AND price_type = 'EGG'
                AND item_name = 'SMALL EGG'
                AND EXTRACT(YEAR FROM price_date) = large_egg_record.year
                AND EXTRACT(MONTH FROM price_date) = large_egg_record.month
        ) THEN
            -- Insert Small egg price (using INSERT ... SELECT to avoid constraint issues)
            INSERT INTO price_history (tenant_id, price_date, price_type, item_name, price)
            SELECT tenant_uuid, month_start_date, 'EGG', 'SMALL EGG', small_price
            WHERE NOT EXISTS (
                SELECT 1 FROM price_history
                WHERE tenant_id = tenant_uuid
                    AND price_date = month_start_date
                    AND price_type = 'EGG'
                    AND item_name = 'SMALL EGG'
            );
            
            RAISE NOTICE 'Inserted SMALL EGG: %-%: ₹% (based on LARGE EGG: ₹%)',
                large_egg_record.year,
                large_egg_record.month,
                small_price,
                large_egg_record.average_price;
            inserted_count := inserted_count + 1;
        END IF;
    END LOOP;
    
    RAISE NOTICE '';
    RAISE NOTICE '✅ Completed! Inserted/Updated % price records', inserted_count;
END $$;

-- Verify the results
SELECT 
    price_date,
    item_name,
    price,
    created_at
FROM price_history
WHERE tenant_id = (SELECT id FROM tenants WHERE name = 'Pradeep Farm' LIMIT 1)
    AND price_type = 'EGG'
    AND item_name IN ('LARGE EGG', 'MEDIUM EGG', 'SMALL EGG')
ORDER BY price_date DESC, item_name;

