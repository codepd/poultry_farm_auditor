# Quick Fix: Setup Tenant Items

## The Problem
You're seeing "Failed to load items" because the `tenant_items` table doesn't have data for your tenant.

## Easiest Solution: Use SQL Script

**Step 1:** Open your database tool (pgAdmin, DBeaver, TablePlus, etc.)

**Step 2:** Run this SQL script:

```sql
-- Get your tenant UUID first
SELECT id, name FROM tenants;

-- Then run this (replace 'YOUR_TENANT_UUID' with your actual tenant UUID from above)
-- Or just run the full script below which works for ALL tenants

-- Default egg items for ALL tenants
INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'LARGE EGG', 1, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'MEDIUM EGG', 2, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'SMALL EGG', 3, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'BROKEN EGG', 4, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'DOUBLE YOLK', 5, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'DIRT EGG', 6, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'CORRECT EGG', 7, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'EXPORT EGG', 8, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

-- Default feed items for ALL tenants
INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'LAYER MASH', 1, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'GROWER MASH', 2, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'PRE LAYER MASH', 3, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'LAYER MASH BULK', 4, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'GROWER MASH BULK', 5, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'PRE LAYER MASH BULK', 6, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO UPDATE SET is_active = TRUE;
```

**Step 3:** Verify it worked:
```sql
SELECT 
    t.name as tenant_name,
    ti.category,
    COUNT(*) as item_count
FROM tenant_items ti
JOIN tenants t ON ti.tenant_id = t.id
WHERE ti.is_active = TRUE
GROUP BY t.name, ti.category;
```

You should see your tenant with 8 EGG items and 6 FEED items.

**Step 4:** Refresh your browser - the dropdowns should now work!

## Alternative: Install Python Dependencies

If you prefer using the Python script:

```bash
cd python_backend
source venv/bin/activate
pip install psycopg2-binary
python3 cli/setup_tenant_items.py
```

But the SQL script is easier and doesn't require Python dependencies!




