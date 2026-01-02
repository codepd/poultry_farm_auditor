-- ============================================================
-- Simple Migration: Convert tenant_id from INTEGER to UUID
-- ============================================================
-- This is a simpler version that works with most SQL clients
-- Run each section separately if needed
-- ============================================================

-- STEP 1: Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- STEP 2: Check current tenant (run this first to see what you have)
SELECT 
    id, 
    name, 
    pg_typeof(id) as id_type,
    CASE 
        WHEN pg_typeof(id)::text = 'integer' THEN 'NEEDS MIGRATION'
        WHEN pg_typeof(id)::text = 'uuid' THEN 'ALREADY UUID'
        ELSE 'UNKNOWN TYPE'
    END as migration_status
FROM tenants 
WHERE id = 1 OR name LIKE '%Pradeep%' OR name LIKE '%Farm%'
LIMIT 5;

-- ============================================================
-- STEP 3A: If tenant table ALREADY uses UUID
-- ============================================================
-- Just update the name (uncomment and run if needed)

-- UPDATE tenants 
-- SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
-- WHERE id IN (
--     SELECT id FROM tenants 
--     WHERE id::text = '1' 
--        OR name LIKE '%Pradeep%' 
--        OR name LIKE '%Farm%'
--     LIMIT 1
-- );

-- ============================================================
-- STEP 3B: If tenant table uses INTEGER (needs full migration)
-- ============================================================
-- Uncomment and run this section if Step 2 shows "NEEDS MIGRATION"

-- Generate UUID and create new tenant record
-- Replace '00000000-0000-0000-0000-000000000001' with a generated UUID if needed
-- Or use: SELECT uuid_generate_v4(); to generate one first

-- First, generate a UUID (run this and copy the result):
-- SELECT uuid_generate_v4() as new_tenant_uuid;

-- Then replace <NEW_UUID_HERE> below with the UUID from above:

/*
DO $$
DECLARE
    new_uuid UUID;
    old_id INTEGER := 1;
BEGIN
    -- Generate new UUID
    new_uuid := uuid_generate_v4();
    
    RAISE NOTICE 'Migrating tenant from id=% to UUID=%', old_id, new_uuid;
    
    -- Create new tenant with UUID (copying all data from id=1)
    INSERT INTO tenants (
        id, name, location, country_code, currency, 
        number_format, date_format, capacity, 
        created_at, updated_at
    )
    SELECT 
        new_uuid,
        'Pradeep Farm',
        COALESCE(location, ''),
        COALESCE(country_code, 'IND'),
        COALESCE(currency, 'INR'),
        COALESCE(number_format, 'lakhs'),
        COALESCE(date_format, 'DD-MM-YYYY'),
        capacity,
        created_at,
        CURRENT_TIMESTAMP
    FROM tenants
    WHERE id = old_id;
    
    RAISE NOTICE 'Created new tenant record';
    
    -- Update all foreign key references
    UPDATE tenant_users SET tenant_id = new_uuid WHERE tenant_id = old_id;
    UPDATE invitations SET tenant_id = new_uuid WHERE tenant_id = old_id;
    UPDATE transactions SET tenant_id = new_uuid WHERE tenant_id = old_id;
    UPDATE price_history SET tenant_id = new_uuid WHERE tenant_id = old_id;
    UPDATE ledger_parses SET tenant_id = new_uuid WHERE tenant_id = old_id;
    UPDATE hen_batches SET tenant_id = new_uuid WHERE tenant_id = old_id;
    UPDATE employees SET tenant_id = new_uuid WHERE tenant_id = old_id;
    UPDATE sensitive_data_config SET tenant_id = new_uuid WHERE tenant_id = old_id;
    UPDATE role_permissions SET tenant_id = new_uuid WHERE tenant_id = old_id;
    
    RAISE NOTICE 'Updated all foreign key references';
    
    -- Delete old tenant record
    DELETE FROM tenants WHERE id = old_id;
    
    RAISE NOTICE 'Migration complete! New UUID: %', new_uuid;
END $$;
*/

-- ============================================================
-- STEP 4: Get final tenant UUID (run this after migration)
-- ============================================================
SELECT 
    id::text as tenant_uuid,
    name,
    location,
    country_code,
    currency,
    created_at
FROM tenants 
WHERE name = 'Pradeep Farm'
ORDER BY created_at DESC
LIMIT 1;

-- ============================================================
-- STEP 5: Verify migration (optional - check for orphaned records)
-- ============================================================
-- These should all return 0 rows if migration was successful

SELECT 'tenant_users' as table_name, COUNT(*) as orphaned_count
FROM tenant_users tu
LEFT JOIN tenants t ON tu.tenant_id = t.id
WHERE t.id IS NULL

UNION ALL

SELECT 'transactions' as table_name, COUNT(*) as orphaned_count
FROM transactions tr
LEFT JOIN tenants t ON tr.tenant_id = t.id
WHERE t.id IS NULL

UNION ALL

SELECT 'price_history' as table_name, COUNT(*) as orphaned_count
FROM price_history ph
LEFT JOIN tenants t ON ph.tenant_id = t.id
WHERE t.id IS NULL;

