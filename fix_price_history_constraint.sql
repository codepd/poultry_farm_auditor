-- Fix price_history table: Add unique constraint if it doesn't exist
-- This ensures ON CONFLICT works properly

-- Check if constraint exists and add it if missing
DO $$
BEGIN
    -- Check if the unique constraint exists
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_constraint 
        WHERE conname = 'price_history_tenant_id_price_date_price_type_item_name_key'
           OR conname = 'price_history_tenant_id_price_date_price_type_item_name_unique'
    ) THEN
        -- Add unique constraint
        ALTER TABLE price_history 
        ADD CONSTRAINT price_history_tenant_id_price_date_price_type_item_name_key 
        UNIQUE (tenant_id, price_date, price_type, item_name);
        
        RAISE NOTICE '✅ Added unique constraint to price_history table';
    ELSE
        RAISE NOTICE 'ℹ️  Unique constraint already exists';
    END IF;
END $$;

-- Verify the constraint exists
SELECT 
    conname as constraint_name,
    contype as constraint_type
FROM pg_constraint
WHERE conrelid = 'price_history'::regclass
    AND contype = 'u'
ORDER BY conname;


