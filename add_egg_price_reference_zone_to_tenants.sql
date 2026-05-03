ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS egg_price_reference_zone VARCHAR(100) DEFAULT 'Namakkal';

UPDATE tenants
SET egg_price_reference_zone = 'Namakkal'
WHERE egg_price_reference_zone IS NULL OR trim(egg_price_reference_zone) = '';

COMMENT ON COLUMN tenants.egg_price_reference_zone IS
'Preferred NECC zone for tenant egg price import (e.g., Namakkal, Hyderabad).';
