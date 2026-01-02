-- Create test employee user with OTHER_USER role
-- Email: test.employee1@gmail.com
-- Name: Employee1
-- Role: OTHER_USER
-- Password: test123 (bcrypt hash)

-- Step 1: Get tenant UUID (assuming "Pradeep Farm" or first tenant)
DO $$
DECLARE
    tenant_uuid UUID;
    user_id_var INTEGER;
    -- bcrypt hash for password "test123" (cost 10, generated using Go bcrypt)
    password_hash_var VARCHAR(255) := '$2a$10$SWmT6ft4Xg5PaxIBUFdY/uuRr8fI/e563aIpllAzRiJIZsOqHiV0W';
BEGIN
    -- Get tenant UUID (use "Pradeep Farm" or first available tenant)
    SELECT id INTO tenant_uuid 
    FROM tenants 
    WHERE name = 'Pradeep Farm' 
    LIMIT 1;
    
    -- If "Pradeep Farm" doesn't exist, use first tenant
    IF tenant_uuid IS NULL THEN
        SELECT id INTO tenant_uuid FROM tenants LIMIT 1;
    END IF;
    
    IF tenant_uuid IS NULL THEN
        RAISE EXCEPTION 'No tenant found! Please create a tenant first.';
    END IF;
    
    RAISE NOTICE 'Using tenant: %', tenant_uuid;
    
    -- Step 2: Check if user already exists
    SELECT id INTO user_id_var 
    FROM users 
    WHERE email = 'test.employee1@gmail.com';
    
    IF user_id_var IS NOT NULL THEN
        RAISE NOTICE 'User already exists with ID: %. Updating...', user_id_var;
        
        -- Update existing user
        UPDATE users 
        SET full_name = 'Employee1',
            password_hash = password_hash_var,
            is_active = TRUE,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = user_id_var;
    ELSE
        -- Step 3: Create new user
        INSERT INTO users (email, password_hash, full_name, is_active)
        VALUES ('test.employee1@gmail.com', password_hash_var, 'Employee1', TRUE)
        RETURNING id INTO user_id_var;
        
        RAISE NOTICE 'Created new user with ID: %', user_id_var;
    END IF;
    
    -- Step 4: Link user to tenant with OTHER_USER role
    INSERT INTO tenant_users (tenant_id, user_id, role, is_owner)
    VALUES (tenant_uuid, user_id_var, 'OTHER_USER', FALSE)
    ON CONFLICT (tenant_id, user_id) DO UPDATE
    SET role = 'OTHER_USER',
        is_owner = FALSE,
        updated_at = CURRENT_TIMESTAMP;
    
    RAISE NOTICE '✅ User "Employee1" created/updated successfully!';
    RAISE NOTICE '   Email: test.employee1@gmail.com';
    RAISE NOTICE '   Password: test123';
    RAISE NOTICE '   Role: OTHER_USER';
    RAISE NOTICE '   User ID: %', user_id_var;
    RAISE NOTICE '   Tenant ID: %', tenant_uuid;
END $$;

-- Step 5: Verify the user was created
SELECT 
    u.id as user_id,
    u.email,
    u.full_name,
    u.is_active,
    t.name as tenant_name,
    tu.role,
    tu.is_owner
FROM users u
JOIN tenant_users tu ON tu.user_id = u.id
JOIN tenants t ON t.id = tu.tenant_id
WHERE u.email = 'test.employee1@gmail.com';

-- Step 6: Show OTHER_USER permissions
SELECT 
    'OTHER_USER Permissions:' as info,
    rp.can_view_sensitive_data,
    rp.can_edit_transactions,
    rp.can_approve_transactions,
    rp.can_manage_users,
    rp.can_view_charts
FROM role_permissions rp
JOIN tenant_users tu ON tu.role = rp.role
JOIN users u ON u.id = tu.user_id
WHERE u.email = 'test.employee1@gmail.com'
LIMIT 1;

SELECT '✅ Test employee user setup complete!' as status;

