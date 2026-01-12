-- Add age category configuration columns to tenants table
-- Standard industry ranges: Chick: 0-6w, Grower: 6-18w, Pre-layer: 18-22w, Layer: 22+w

ALTER TABLE tenants 
ADD COLUMN IF NOT EXISTS age_category_chick_max_weeks INTEGER DEFAULT 6,
ADD COLUMN IF NOT EXISTS age_category_grower_max_weeks INTEGER DEFAULT 18,
ADD COLUMN IF NOT EXISTS age_category_prelayer_max_weeks INTEGER DEFAULT 22;

-- Update existing tenants to have default values
UPDATE tenants 
SET 
  age_category_chick_max_weeks = 6,
  age_category_grower_max_weeks = 18,
  age_category_prelayer_max_weeks = 22
WHERE age_category_chick_max_weeks IS NULL;

-- Add comments
COMMENT ON COLUMN tenants.age_category_chick_max_weeks IS 'Maximum weeks for Chick category (default: 6)';
COMMENT ON COLUMN tenants.age_category_grower_max_weeks IS 'Maximum weeks for Grower category (default: 18)';
COMMENT ON COLUMN tenants.age_category_prelayer_max_weeks IS 'Maximum weeks for Pre-Layer category (default: 22)';

