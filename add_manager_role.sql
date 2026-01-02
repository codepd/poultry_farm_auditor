-- Add MANAGER role to user_role_enum
-- This script safely adds 'MANAGER' to the existing enum type

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
        RAISE NOTICE 'MANAGER role added to user_role_enum';
    ELSE
        RAISE NOTICE 'MANAGER role already exists in user_role_enum';
    END IF;
END $$;

-- Verify the enum now includes MANAGER
SELECT enumlabel as role_name 
FROM pg_enum 
WHERE enumtypid = (SELECT oid FROM pg_type WHERE typname = 'user_role_enum')
ORDER BY enumsortorder;

