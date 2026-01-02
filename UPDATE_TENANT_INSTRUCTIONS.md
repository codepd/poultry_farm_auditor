# Update Tenant Name and Migrate to UUID - Instructions

## 🎯 What This Does

This guide will help you:
1. **Migrate tenant_id from INTEGER (1) to UUID** ✅
2. **Update tenant name to "Pradeep Farm"** ✅
3. **Update all foreign key references** in related tables ✅

## ⚡ Quick Start (Choose One)

- **Have a database GUI tool?** → Use **Option 1** (Easiest)
- **Can install psql?** → Use **Option 2** (Recommended)
- **Prefer Python?** → Use **Option 3**
- **Using Docker?** → Use **Option 4**
- **Want manual control?** → Use **Option 5**

---

## Detailed Instructions

Since `psql` is not in your PATH and `psycopg2` installation has issues, here are alternative ways:

---

## Option 1: Use Database GUI Tool (Recommended - Easiest)

If you have a database GUI tool (like **pgAdmin**, **DBeaver**, **TablePlus**, **Postico**, etc.):

### Steps:

1. **Connect to your PostgreSQL database**
   - Host: `localhost`
   - Port: `5432`
   - Database: `poultry_farm`
   - Username: `postgres`
   - Password: `postgres`

2. **Check current state first:**
   - Open the SQL editor
   - Run this query to see if migration is needed:
   ```sql
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
   ```

3. **If it shows "ALREADY UUID":**
   - Just update the name:
   ```sql
   UPDATE tenants 
   SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
   WHERE id IN (
       SELECT id FROM tenants 
       WHERE id::text = '1' 
          OR name LIKE '%Pradeep%' 
          OR name LIKE '%Farm%'
       LIMIT 1
   );
   ```

4. **If it shows "NEEDS MIGRATION":**
   - Use the **simpler script** `migrate_tenant_simple.sql` (works better with GUI tools)
   - Or follow **Option 5** below for manual step-by-step instructions

5. **Get the final tenant UUID:**
   ```sql
   SELECT id::text as tenant_uuid, name 
   FROM tenants 
   WHERE name = 'Pradeep Farm';
   ```

6. **Copy the tenant UUID** and use it to create the owner:
   ```bash
   ./setup_first_owner.sh <tenant-uuid> owner@example.com password123 "Owner Name"
   ```

---

## Option 2: Install PostgreSQL Client Tools (psql)

### On macOS with Homebrew:

```bash
# Install PostgreSQL (includes psql)
brew install postgresql@14

# Or if you have PostgreSQL installed but not in PATH:
brew link postgresql@14

# Add to PATH (add to ~/.zshrc or ~/.bash_profile):
export PATH="/usr/local/opt/postgresql@14/bin:$PATH"

# Reload shell
source ~/.zshrc
```

### Then run the migration:

```bash
# Run the complete migration script
psql -U postgres -d poultry_farm -f migrate_tenant_to_uuid.sql

# Or if prompted for password:
PGPASSWORD=postgres psql -U postgres -d poultry_farm -f migrate_tenant_to_uuid.sql
```

The script will:
- ✅ Check current tenant structure
- ✅ Migrate from INTEGER to UUID (if needed)
- ✅ Update all foreign key references
- ✅ Update tenant name to "Pradeep Farm"
- ✅ Display the final tenant UUID

---

## Option 3: Use Python Migration Script

If you can get `psycopg2` working:

```bash
# Install PostgreSQL development libraries first
brew install postgresql

# Then install psycopg2
pip3 install psycopg2-binary

# Or if that fails, try:
pip3 install --upgrade pip
pip3 install psycopg2-binary
```

Then run:
```bash
cd python_backend/cli
python3 migrate_tenant_to_uuid.py
```

This script will:
- Check if migration is needed
- Ask for confirmation before migrating
- Update all foreign key references
- Update tenant name
- Display the final UUID

---

## Option 4: Direct SQL via Docker (if using Docker)

If PostgreSQL is running in Docker:

```bash
# Find container
docker ps | grep postgres

# Copy migration script into container and run it
docker cp migrate_tenant_to_uuid.sql <container-name>:/tmp/
docker exec -it <container-name> psql -U postgres -d poultry_farm -f /tmp/migrate_tenant_to_uuid.sql

# Or run SQL directly
docker exec -it <container-name> psql -U postgres -d poultry_farm < migrate_tenant_to_uuid.sql
```

---

## Option 5: Manual Step-by-Step SQL (No DO blocks - Works with all tools)

If you prefer to run SQL commands manually or your tool doesn't support DO blocks:

### Step 1: Check current state
```sql
-- Check if tenant table uses integer or UUID
SELECT data_type 
FROM information_schema.columns 
WHERE table_name = 'tenants' AND column_name = 'id';

-- Check current tenant
SELECT id, name, pg_typeof(id) as id_type 
FROM tenants 
WHERE id = 1 OR name LIKE '%Pradeep%';
```

### Step 2A: If tenant table already uses UUID
```sql
-- Just update the name
UPDATE tenants 
SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
WHERE id IN (
    SELECT id FROM tenants 
    WHERE id::text = '1' 
       OR name LIKE '%Pradeep%' 
       OR name LIKE '%Farm%'
    LIMIT 1
);

-- Get the UUID
SELECT id::text as tenant_uuid, name 
FROM tenants 
WHERE name = 'Pradeep Farm';
```

### Step 2B: If tenant table uses INTEGER (needs full migration)

**Step 2B.1: Enable UUID extension**
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

**Step 2B.2: Generate a new UUID (copy this value!)**
```sql
SELECT uuid_generate_v4() as new_tenant_uuid;
```
**⚠️ IMPORTANT: Copy the UUID value from the result! You'll need it in the next steps.**

**Step 2B.3: Create new tenant with UUID**
```sql
-- Replace '<NEW_UUID_HERE>' with the UUID from Step 2B.2
INSERT INTO tenants (
    id, name, location, country_code, currency, 
    number_format, date_format, capacity, 
    created_at, updated_at
)
SELECT 
    '<NEW_UUID_HERE>'::uuid,  -- Replace with actual UUID
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
WHERE id = 1;
```

**Step 2B.4: Update all foreign key references**
```sql
-- Replace '<NEW_UUID_HERE>' with the UUID from Step 2B.2
-- Run each UPDATE statement separately:

UPDATE tenant_users SET tenant_id = '<NEW_UUID_HERE>'::uuid WHERE tenant_id = 1;
UPDATE invitations SET tenant_id = '<NEW_UUID_HERE>'::uuid WHERE tenant_id = 1;
UPDATE transactions SET tenant_id = '<NEW_UUID_HERE>'::uuid WHERE tenant_id = 1;
UPDATE price_history SET tenant_id = '<NEW_UUID_HERE>'::uuid WHERE tenant_id = 1;
UPDATE ledger_parses SET tenant_id = '<NEW_UUID_HERE>'::uuid WHERE tenant_id = 1;
UPDATE hen_batches SET tenant_id = '<NEW_UUID_HERE>'::uuid WHERE tenant_id = 1;
UPDATE employees SET tenant_id = '<NEW_UUID_HERE>'::uuid WHERE tenant_id = 1;
UPDATE sensitive_data_config SET tenant_id = '<NEW_UUID_HERE>'::uuid WHERE tenant_id = 1;
UPDATE role_permissions SET tenant_id = '<NEW_UUID_HERE>'::uuid WHERE tenant_id = 1;
```

**Step 2B.5: Delete old tenant**
```sql
DELETE FROM tenants WHERE id = 1;
```

**Step 2B.6: Get the new UUID**
```sql
SELECT id::text as tenant_uuid, name 
FROM tenants 
WHERE name = 'Pradeep Farm';
```

---

## Quick Check: Is PostgreSQL Running?

```bash
# Check if PostgreSQL is running
pg_isready -h localhost -p 5432

# Or check Docker
docker ps | grep postgres

# Test connection
psql -U postgres -d poultry_farm -c "SELECT 1;"
```

---

## After Migration

Once migration is complete, you'll get a tenant UUID. Use it to create the owner:

```bash
./setup_first_owner.sh <tenant-uuid> owner@example.com password123 "Owner Name"
```

Or manually:
```sql
-- Get tenant UUID
SELECT id::text FROM tenants WHERE name = 'Pradeep Farm';

-- Then create owner (replace <tenant-uuid> with actual UUID)
-- See setup_first_owner.sh for the full process
```

---

## Troubleshooting

### "relation does not exist" errors
- Make sure you've run the schema initialization first
- Check that tables exist: `\dt` in psql

### "permission denied" errors
- Make sure you're using the `postgres` user
- Check database connection: `psql -U postgres -d poultry_farm -c "SELECT 1;"`

### "column does not exist" errors
- The schema might be different - check with: `\d tenants` in psql
- You may need to run schema updates first


