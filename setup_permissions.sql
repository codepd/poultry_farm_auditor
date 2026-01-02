-- Setup role permissions for Pradeep Farm tenant
-- This script creates the role_permissions table and sets default permissions

-- Step 1: Create role_permissions table if it doesn't exist
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

-- Step 2: Get tenant UUID for Pradeep Farm
DO $$
DECLARE
    tenant_uuid UUID;
BEGIN
    -- Get tenant UUID
    SELECT id INTO tenant_uuid FROM tenants WHERE name = 'Pradeep Farm' LIMIT 1;
    
    IF tenant_uuid IS NULL THEN
        RAISE EXCEPTION 'Tenant "Pradeep Farm" not found!';
    END IF;
    
    RAISE NOTICE 'Setting up permissions for tenant: %', tenant_uuid;
    
    -- Insert permissions for OWNER (full access)
    INSERT INTO role_permissions (
        tenant_id, role, can_view_sensitive_data, can_edit_transactions,
        can_approve_transactions, can_manage_users, can_view_charts
    )
    VALUES (
        tenant_uuid, 'OWNER', TRUE, TRUE, TRUE, TRUE, TRUE
    )
    ON CONFLICT (tenant_id, role) DO UPDATE
    SET can_view_sensitive_data = EXCLUDED.can_view_sensitive_data,
        can_edit_transactions = EXCLUDED.can_edit_transactions,
        can_approve_transactions = EXCLUDED.can_approve_transactions,
        can_manage_users = EXCLUDED.can_manage_users,
        can_view_charts = EXCLUDED.can_view_charts,
        updated_at = CURRENT_TIMESTAMP;
    
    -- Insert permissions for CO_OWNER (full access)
    INSERT INTO role_permissions (
        tenant_id, role, can_view_sensitive_data, can_edit_transactions,
        can_approve_transactions, can_manage_users, can_view_charts
    )
    VALUES (
        tenant_uuid, 'CO_OWNER', TRUE, TRUE, TRUE, TRUE, TRUE
    )
    ON CONFLICT (tenant_id, role) DO UPDATE
    SET can_view_sensitive_data = EXCLUDED.can_view_sensitive_data,
        can_edit_transactions = EXCLUDED.can_edit_transactions,
        can_approve_transactions = EXCLUDED.can_approve_transactions,
        can_manage_users = EXCLUDED.can_manage_users,
        can_view_charts = EXCLUDED.can_view_charts,
        updated_at = CURRENT_TIMESTAMP;
    
    -- Insert permissions for MANAGER (can edit transactions and view charts, but cannot approve or manage users)
    INSERT INTO role_permissions (
        tenant_id, role, can_view_sensitive_data, can_edit_transactions,
        can_approve_transactions, can_manage_users, can_view_charts
    )
    VALUES (
        tenant_uuid, 'MANAGER', TRUE, TRUE, FALSE, FALSE, TRUE
    )
    ON CONFLICT (tenant_id, role) DO UPDATE
    SET can_view_sensitive_data = EXCLUDED.can_view_sensitive_data,
        can_edit_transactions = EXCLUDED.can_edit_transactions,
        can_approve_transactions = EXCLUDED.can_approve_transactions,
        can_manage_users = EXCLUDED.can_manage_users,
        can_view_charts = EXCLUDED.can_view_charts,
        updated_at = CURRENT_TIMESTAMP;
    
    -- Insert permissions for AUDITOR (view only, can see charts)
    INSERT INTO role_permissions (
        tenant_id, role, can_view_sensitive_data, can_edit_transactions,
        can_approve_transactions, can_manage_users, can_view_charts
    )
    VALUES (
        tenant_uuid, 'AUDITOR', TRUE, FALSE, FALSE, FALSE, TRUE
    )
    ON CONFLICT (tenant_id, role) DO UPDATE
    SET can_view_sensitive_data = EXCLUDED.can_view_sensitive_data,
        can_edit_transactions = EXCLUDED.can_edit_transactions,
        can_approve_transactions = EXCLUDED.can_approve_transactions,
        can_manage_users = EXCLUDED.can_manage_users,
        can_view_charts = EXCLUDED.can_view_charts,
        updated_at = CURRENT_TIMESTAMP;
    
    -- Insert permissions for OTHER_USER (limited access, no charts)
    INSERT INTO role_permissions (
        tenant_id, role, can_view_sensitive_data, can_edit_transactions,
        can_approve_transactions, can_manage_users, can_view_charts
    )
    VALUES (
        tenant_uuid, 'OTHER_USER', FALSE, TRUE, FALSE, FALSE, FALSE
    )
    ON CONFLICT (tenant_id, role) DO UPDATE
    SET can_view_sensitive_data = EXCLUDED.can_view_sensitive_data,
        can_edit_transactions = EXCLUDED.can_edit_transactions,
        can_approve_transactions = EXCLUDED.can_approve_transactions,
        can_manage_users = EXCLUDED.can_manage_users,
        can_view_charts = EXCLUDED.can_view_charts,
        updated_at = CURRENT_TIMESTAMP;
    
    -- Insert permissions for ADMIN (full access)
    INSERT INTO role_permissions (
        tenant_id, role, can_view_sensitive_data, can_edit_transactions,
        can_approve_transactions, can_manage_users, can_view_charts
    )
    VALUES (
        tenant_uuid, 'ADMIN', TRUE, TRUE, TRUE, TRUE, TRUE
    )
    ON CONFLICT (tenant_id, role) DO UPDATE
    SET can_view_sensitive_data = EXCLUDED.can_view_sensitive_data,
        can_edit_transactions = EXCLUDED.can_edit_transactions,
        can_approve_transactions = EXCLUDED.can_approve_transactions,
        can_manage_users = EXCLUDED.can_manage_users,
        can_view_charts = EXCLUDED.can_view_charts,
        updated_at = CURRENT_TIMESTAMP;
    
    RAISE NOTICE '✅ Permissions setup complete for tenant: %', tenant_uuid;
END $$;

-- Step 3: Verify permissions were created
SELECT 
    role, 
    can_view_charts, 
    can_view_sensitive_data, 
    can_edit_transactions,
    can_approve_transactions,
    can_manage_users
FROM role_permissions
WHERE tenant_id = (SELECT id FROM tenants WHERE name = 'Pradeep Farm' LIMIT 1)
ORDER BY role;



