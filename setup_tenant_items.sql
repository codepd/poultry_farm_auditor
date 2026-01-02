-- Script to populate default tenant items for existing tenants
-- This ensures all tenants have the standard egg and feed items configured

-- Default egg items
INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'LARGE EGG', 1, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'MEDIUM EGG', 2, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'SMALL EGG', 3, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'BROKEN EGG', 4, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'DOUBLE YOLK', 5, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'DIRT EGG', 6, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'CORRECT EGG', 7, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'EGG', 'EXPORT EGG', 8, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

-- Default feed items
INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'LAYER MASH', 1, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'GROWER MASH', 2, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'PRE LAYER MASH', 3, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'LAYER MASH BULK', 4, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'GROWER MASH BULK', 5, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
SELECT id, 'FEED', 'PRE LAYER MASH BULK', 6, TRUE FROM tenants
ON CONFLICT (tenant_id, category, item_name) DO NOTHING;

-- Verify the items were created
SELECT 
    t.name as tenant_name,
    ti.category,
    ti.item_name,
    ti.display_order,
    ti.is_active
FROM tenant_items ti
JOIN tenants t ON ti.tenant_id = t.id
ORDER BY t.name, ti.category, ti.display_order;




