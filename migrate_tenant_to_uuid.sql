-- ============================================================
-- Migration Script: Convert tenant_id from INTEGER to UUID
-- ============================================================
-- This script:
-- 1. Checks if tenant table uses integer or UUID
-- 2. If integer, migrates tenant_id=1 to UUID
-- 3. Updates all foreign key references
-- 4. Updates tenant name to "Pradeep Farm"
-- ============================================================

-- Step 1: Enable UUID extension (if not already enabled)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Step 2: Check current tenant table structure
DO $$
DECLARE
    id_type TEXT;
    tenant_exists BOOLEAN;
    new_uuid UUID;
BEGIN
    -- Check if ten5678
    \]'=EWAant with id=1 exists and get its type
    SELECT data_type INTO id_type
    FROM information_schema.columns
    WHERE table_name = 'tenants' AND column_name = 'id';
    
    SELECT EXISTS(SELECT 1 FROM tenants WHERE id = 1) INTO tenant_exists;
    
    RAISE NOTICE 'Tenant ID type: %', id_type;
    RAISE NOTICE 'Tenant with id=1 exists: %', tenant_exists;
    
    -- If tenant table already uses UUID, just update the name
    IF id_type = 'uuid' THEN
        RAISE NOTICE 'Tenant table already uses UUID';
        
        -- Find tenant (might be UUID or integer stored as text)
        IF tenant_exists THEN
            -- Get the actual UUID value
            SELECT id::uuid INTO new_uuid FROM tenants WHERE id::text = '1' LIMIT 1;
            
            IF new_uuid IS NULL THEN
                -- Try to find by name
                SELECT id INTO new_uuid FROM tenants WHERE name LIKE '%Pradeep%' OR name LIKE '%Farm%' LIMIT 1;
            END IF;
            
            IF new_uuid IS NOT NULL THEN
                -- Update name
                UPDATE tenants 
                SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
                WHERE id = new_uuid;
                
                RAISE NOTICE 'Updated tenant name to "Pradeep Farm"';
                RAISE NOTICE 'Tenant UUID: %', new_uuid;
            ELSE
                RAISE NOTICE 'Could not find tenant to update';
            END IF;
        ELSE
            RAISE NOTICE 'No tenant with id=1 found. Listing all tenants:';
            -- This will be shown in the result set
        END IF;
        
    ELSE
        -- Tenant table uses INTEGER - need full migration
        RAISE NOTICE 'Tenant table uses INTEGER - starting migration to UUID...';
        
        -- Generate new UUID
        new_uuid := uuid_generate_v4();
        RAISE NOTICE 'Generated new UUID: %', new_uuid;
        
        -- Create new tenant record with UUID (copying all data from id=1)
        INSERT INTO tenants (
            id, name, location, country_code, currency, 
            number_format, date_format, capacity, 
            created_at, updated_at
        )
        SELECT 
            new_uuid,
            'Pradeep Farm',
            location,
            COALESCE(country_code, 'IND'),
            COALESCE(currency, 'INR'),
            COALESCE(number_format, 'lakhs'),
            COALESCE(date_format, 'DD-MM-YYYY'),
            capacity,
            created_at,
            CURRENT_TIMESTAMP
        FROM tenants
        WHERE id = 1
        ON CONFLICT DO NOTHING;
        
        RAISE NOTICE 'Created new tenant record with UUID';
        
        -- Update all foreign key references
        -- tenant_users
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tenant_users') THEN
            UPDATE tenant_users SET tenant_id = new_uuid WHERE tenant_id = 1;
            RAISE NOTICE 'Updated tenant_users table';
        END IF;
        
        -- invitations
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'invitations') THEN
            UPDATE invitations SET tenant_id = new_uuid WHERE tenant_id = 1;
            RAISE NOTICE 'Updated invitations table';
        END IF;
        
        -- transactions
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'transactions') THEN
            UPDATE transactions SET tenant_id = new_uuid WHERE tenant_id = 1;
            RAISE NOTICE 'Updated transactions table';
        END IF;
        
        -- price_history
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'price_history') THEN
            UPDATE price_history SET tenant_id = new_uuid WHERE tenant_id = 1;
            RAISE NOTICE 'Updated price_history table';
        END IF;
        
        -- ledger_parses
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ledger_parses') THEN
            UPDATE ledger_parses SET tenant_id = new_uuid WHERE tenant_id = 1;
            RAISE NOTICE 'Updated ledger_parses table';
        END IF;
        
        -- hen_batches
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'hen_batches') THEN
            UPDATE hen_batches SET tenant_id = new_uuid WHERE tenant_id = 1;
            RAISE NOTICE 'Updated hen_batches table';
        END IF;
        
        -- employees
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'employees') THEN
            UPDATE employees SET tenant_id = new_uuid WHERE tenant_id = 1;
            RAISE NOTICE 'Updated employees table';
        END IF;
        
        -- sensitive_data_config
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'sensitive_data_config') THEN
            UPDATE sensitive_data_config SET tenant_id = new_uuid WHERE tenant_id = 1;
            RAISE NOTICE 'Updated sensitive_data_config table';
        END IF;
        
        -- role_permissions
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'role_permissions') THEN
            UPDATE role_permissions SET tenant_id = new_uuid WHERE tenant_id = 1;
            RAISE NOTICE 'Updated role_permissions table';
        END IF;
        
        -- Delete old tenant record
        DELETE FROM tenants WHERE id = 1;
        RAISE NOTICE 'Deleted old tenant record (id=1)';
        
        RAISE NOTICE 'Migration completed! New tenant UUID: %', new_uuid;
    END IF;
END $$;

-- Step 3: Display final tenant information
SELECT 
    id::text as tenant_uuid,
    name,
    location,
    country_code,
    currency,
    number_format,
    date_format,
    created_at,
    updated_at
FROM tenants 
WHERE name = 'Pradeep Farm'
ORDER BY created_at DESC
LIMIT 1;

-- Step 4: Verify no orphaned records (should return 0 rows)
SELECT 
    'tenant_users' as table_name,
    COUNT(*) as orphaned_count
FROM tenant_users tu
LEFT JOIN tenants t ON tu.tenant_id = t.id
WHERE t.id IS NULL
UNION ALL
SELECT 
    'transactions' as table_name,
    COUNT(*) as orphaned_count
FROM transactions tr
LEFT JOIN tenants t ON tr.tenant_id = t.id
WHERE t.id IS NULL
UNION ALL
SELECT 
    'price_history' as table_name,
    COUNT(*) as orphaned_count
FROM price_history ph
LEFT JOIN tenants t ON ph.tenant_id = t.id
WHERE t.id IS NULL;

