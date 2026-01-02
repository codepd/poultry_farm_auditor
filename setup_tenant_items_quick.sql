-- Quick setup script for tenant items
-- Run this in your database to populate default items for all tenants

-- First, ensure the tenant_items table exists (should already exist from schema)
-- If not, the schema.py should have created it

-- Insert default egg items for all tenants
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

-- Insert default feed items for all tenants
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

-- Show summary
SELECT 
    t.name as tenant_name,
    ti.category,
    COUNT(*) as item_count,
    STRING_AGG(ti.item_name, ', ' ORDER BY ti.display_order) as items
FROM tenant_items ti
JOIN tenants t ON ti.tenant_id = t.id
WHERE ti.is_active = TRUE
GROUP BY t.name, ti.category
ORDER BY t.name, ti.category;




