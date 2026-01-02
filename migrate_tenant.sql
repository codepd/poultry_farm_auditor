-- Migration script to:
-- 1. Update tenant name to "Pradeep Farm"
-- 2. Handle UUID conversion if needed

-- Step 1: Check current tenant
SELECT id, name, 
       pg_typeof(id) as id_type 
FROM tenants 
WHERE id = 1 OR name LIKE '%Pradeep%' 
LIMIT 5;

-- Step 2: Update tenant name to "Pradeep Farm"
-- If tenant uses integer ID (old schema):
UPDATE tenants 
SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- If tenant uses UUID (new schema), find it first:
-- SELECT id FROM tenants WHERE id::text LIKE '%1%' OR name LIKE '%Pradeep%';
-- Then update: UPDATE tenants SET name = 'Pradeep Farm' WHERE id = '<uuid-here>';

-- Step 3: Verify update
SELECT id, name, updated_at 
FROM tenants 
WHERE name = 'Pradeep Farm';

-- Step 4: Display tenant UUID for use in setup script
SELECT id::text as tenant_uuid, name 
FROM tenants 
WHERE name = 'Pradeep Farm';


