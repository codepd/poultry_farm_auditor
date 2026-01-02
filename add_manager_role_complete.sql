-- Complete script to add MANAGER role and set up permissions
-- Run this with: psql -h localhost -U postgres -d poultry_farm -f add_manager_role_complete.sql

-- Step 1: Add MANAGER role to user_role_enum
DO $$ 
BEGIN
    -- Check if MANAGER already exists in the enum
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_enum 
        WHERE enumlabel = 'MANAGER' 
        AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'user_role_enum')
    ) THEN
        -- Add MANAGER to the enum
        ALTER TYPE user_role_enum ADD VALUE 'MANAGER';
        RAISE NOTICE '✅ MANAGER role added to user_role_enum';
    ELSE
        RAISE NOTICE 'ℹ️  MANAGER role already exists in user_role_enum';
    END IF;
END $$;

-- Step 2: Verify the enum now includes MANAGER
SELECT 'Current roles in enum:' as info;
SELECT enumlabel as role_name 
FROM pg_enum 
WHERE enumtypid = (SELECT oid FROM pg_type WHERE typname = 'user_role_enum')
ORDER BY enumsortorder;

-- Step 3: Ensure role_permissions table exists
CREATE TABLE IF NOT EXISTS role_permissions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role user_role_enum NOT NULL,
    can_view_sensitive_data BOOLEAN DEFAULT FALSE,
    can_edit_transactions BOOLEAN DEFAULT FALSE,
    can_approve_transactions BOOLEAN DEFAULT FALSE,
    can_manage_users BOOLEAN DEFAULT FALSE,
    can_view_charts BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, role)
);

-- Step 4: Add MANAGER permissions for all existing tenants
DO $$
DECLARE
    tenant_rec RECORD;
BEGIN
    FOR tenant_rec IN SELECT id, name FROM tenants LOOP
        -- Insert or update MANAGER permissions
        INSERT INTO role_permissions (
            tenant_id, role, can_view_sensitive_data, can_edit_transactions,
            can_approve_transactions, can_manage_users, can_view_charts
        )
        VALUES (
            tenant_rec.id, 'MANAGER', TRUE, TRUE, FALSE, FALSE, TRUE
        )
        ON CONFLICT (tenant_id, role) DO UPDATE
        SET can_view_sensitive_data = EXCLUDED.can_view_sensitive_data,
            can_edit_transactions = EXCLUDED.can_edit_transactions,
            can_approve_transactions = EXCLUDED.can_approve_transactions,
            can_manage_users = EXCLUDED.can_manage_users,
            can_view_charts = EXCLUDED.can_view_charts,
            updated_at = CURRENT_TIMESTAMP;
        
        RAISE NOTICE '✅ Set MANAGER permissions for tenant: % (%)', tenant_rec.name, tenant_rec.id;
    END LOOP;
END $$;

-- Step 5: Verify permissions were set
SELECT 'MANAGER permissions summary:' as info;
SELECT 
    t.name as tenant_name,
    rp.role,
    rp.can_view_charts, 
    rp.can_view_sensitive_data, 
    rp.can_edit_transactions,
    rp.can_approve_transactions,
    rp.can_manage_users
FROM role_permissions rp
JOIN tenants t ON t.id = rp.tenant_id
WHERE rp.role = 'MANAGER'
ORDER BY t.name;

SELECT '✅ MANAGER role setup complete!' as status;


