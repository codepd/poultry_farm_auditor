-- COMPLETE SETUP: Create table and populate tenant items
-- Run this entire script in your database

-- Step 1: Create the tenant_items table
CREATE TABLE IF NOT EXISTS tenant_items (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    category category_enum NOT NULL, -- 'EGG' or 'FEED'
    item_name VARCHAR(255) NOT NULL,
    display_order INTEGER DEFAULT 0, -- Order in dropdown
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, category, item_name)
);

-- Step 2: Create indexes
CREATE INDEX IF NOT EXISTS idx_tenant_items_tenant_category ON tenant_items(tenant_id, category);
CREATE INDEX IF NOT EXISTS idx_tenant_items_active ON tenant_items(tenant_id, category, is_active) WHERE is_active = TRUE;

-- Step 3: Insert default egg items for all tenants
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

-- Step 4: Insert default feed items for all tenants
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

-- Step 5: Verify the setup
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




