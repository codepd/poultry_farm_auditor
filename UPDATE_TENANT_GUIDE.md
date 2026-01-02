# Update Tenant to "Pradeep Farm" Guide

Since you already have tenant_id=1 with all your data, here's how to:
1. Rename it to "Pradeep Farm"
2. Get its UUID (if it's already UUID) or convert it

## Option 1: Direct SQL Update (Recommended)

### Step 1: Update Tenant Name

```sql
-- Connect to database
psql -U postgres -d poultry_farm

-- Update tenant name
UPDATE tenants 
SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
WHERE id = 1;
```

### Step 2: Get Tenant UUID

```sql
-- Get the UUID (works whether id is integer or UUID)
SELECT id::text as tenant_uuid, name 
FROM tenants 
WHERE name = 'Pradeep Farm';
```

Copy the UUID from the output.

### Step 3: Create Owner Using UUID

```bash
./setup_first_owner.sh <tenant-uuid-from-step-2> owner@example.com password123 "Owner Name"
```

## Option 2: Using Go API (if running)

### Step 1: Update via API

```bash
# First, you need to be authenticated, but for now use admin endpoint
# Or update directly in database (Option 1 is easier)
```

### Step 2: Get Tenant Info

```bash
# If API is running and you have auth token
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/tenants?tenant_id=1
```

## Option 3: Using Helper Script

```bash
# Get tenant info
./get_tenant_info.sh

# This will show you the tenant UUID
# Then use it to create owner:
./setup_first_owner.sh <uuid> owner@example.com password123 "Owner Name"
```

## Quick One-Liner

If tenant already uses UUID schema:

```bash
# Get UUID and create owner in one go
TENANT_UUID=$(psql -U postgres -d poultry_farm -t -c "SELECT id::text FROM tenants WHERE id = 1 OR name LIKE '%Farm%' LIMIT 1;" | xargs)
psql -U postgres -d poultry_farm -c "UPDATE tenants SET name = 'Pradeep Farm' WHERE id = '$TENANT_UUID';"
echo "Tenant UUID: $TENANT_UUID"
./setup_first_owner.sh "$TENANT_UUID" owner@example.com password123 "Owner Name"
```

## Check Current Tenant Status

```sql
-- Check if tenant uses UUID or integer
SELECT 
    id,
    pg_typeof(id) as id_type,
    name,
    created_at
FROM tenants 
WHERE id = 1 
   OR name LIKE '%Pradeep%' 
   OR name LIKE '%Farm%';
```

## Notes

- If your tenant table already uses UUID (from the new schema), the `id` column is already UUID
- If it's still integer, you may need to migrate (but this is complex with existing data)
- For now, just update the name and use the UUID directly - the Go API expects UUID format
- All foreign keys in other tables should already reference UUID if you've run the latest schema

## After Update

Once tenant is renamed and you have the UUID:

1. **Create Owner:**
   ```bash
   ./setup_first_owner.sh <tenant-uuid> owner@example.com mypassword123 "Pradeep"
   ```

2. **Login:**
   - Go to: http://localhost:4300/login
   - Email: owner@example.com
   - Password: mypassword123


