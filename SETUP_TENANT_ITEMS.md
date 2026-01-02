# Setup Tenant Items Configuration

## Problem
If you see "Failed to load items" or empty dropdowns in the Quick Entry form, it means the tenant items haven't been configured in the database.

## Solution

### Option 1: Using SQL Script (Recommended)

1. Open your database GUI tool (pgAdmin, DBeaver, etc.) or use `psql`
2. Run the SQL script: `setup_tenant_items_quick.sql`
3. This will populate default egg and feed items for all existing tenants

### Option 2: Using Python Script

```bash
cd python_backend
source ../.env
python3 cli/setup_tenant_items.py
```

### Option 3: Manual SQL

If you prefer to run SQL manually, execute:

```sql
-- For each tenant, insert default items
-- Replace 'YOUR_TENANT_UUID' with your actual tenant UUID

-- Egg items
INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
VALUES 
  ('YOUR_TENANT_UUID', 'EGG', 'LARGE EGG', 1, TRUE),
  ('YOUR_TENANT_UUID', 'EGG', 'MEDIUM EGG', 2, TRUE),
  ('YOUR_TENANT_UUID', 'EGG', 'SMALL EGG', 3, TRUE),
  ('YOUR_TENANT_UUID', 'EGG', 'BROKEN EGG', 4, TRUE),
  ('YOUR_TENANT_UUID', 'EGG', 'DOUBLE YOLK', 5, TRUE),
  ('YOUR_TENANT_UUID', 'EGG', 'DIRT EGG', 6, TRUE),
  ('YOUR_TENANT_UUID', 'EGG', 'CORRECT EGG', 7, TRUE),
  ('YOUR_TENANT_UUID', 'EGG', 'EXPORT EGG', 8, TRUE)
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

-- Feed items
INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
VALUES 
  ('YOUR_TENANT_UUID', 'FEED', 'LAYER MASH', 1, TRUE),
  ('YOUR_TENANT_UUID', 'FEED', 'GROWER MASH', 2, TRUE),
  ('YOUR_TENANT_UUID', 'FEED', 'PRE LAYER MASH', 3, TRUE),
  ('YOUR_TENANT_UUID', 'FEED', 'LAYER MASH BULK', 4, TRUE),
  ('YOUR_TENANT_UUID', 'FEED', 'GROWER MASH BULK', 5, TRUE),
  ('YOUR_TENANT_UUID', 'FEED', 'PRE LAYER MASH BULK', 6, TRUE)
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;
```

## Verify Setup

After running the script, verify items were created:

```sql
SELECT 
    t.name as tenant_name,
    ti.category,
    COUNT(*) as item_count
FROM tenant_items ti
JOIN tenants t ON ti.tenant_id = t.id
WHERE ti.is_active = TRUE
GROUP BY t.name, ti.category
ORDER BY t.name, ti.category;
```

You should see:
- Each tenant should have 8 EGG items
- Each tenant should have 6 FEED items

## Customizing Items

You can add, edit, or disable items by directly updating the `tenant_items` table:

```sql
-- Add a new item
INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
VALUES ('YOUR_TENANT_UUID', 'EGG', 'NEW EGG TYPE', 9, TRUE);

-- Disable an item
UPDATE tenant_items 
SET is_active = FALSE 
WHERE tenant_id = 'YOUR_TENANT_UUID' AND item_name = 'ITEM_NAME';

-- Change display order
UPDATE tenant_items 
SET display_order = 1 
WHERE tenant_id = 'YOUR_TENANT_UUID' AND item_name = 'ITEM_NAME';
```

## Notes

- Items are tenant-specific - each tenant can have different items
- The `display_order` field controls the order in the dropdown
- Set `is_active = FALSE` to hide an item without deleting it
- The frontend will automatically fetch and display active items




