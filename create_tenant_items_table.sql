-- Create tenant_items table if it doesn't exist
-- Run this FIRST before running setup_tenant_items_quick.sql

-- Create the table
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

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_tenant_items_tenant_category ON tenant_items(tenant_id, category);
CREATE INDEX IF NOT EXISTS idx_tenant_items_active ON tenant_items(tenant_id, category, is_active) WHERE is_active = TRUE;

-- Verify table was created
SELECT 
    table_name, 
    column_name, 
    data_type 
FROM information_schema.columns 
WHERE table_name = 'tenant_items' 
ORDER BY ordinal_position;




