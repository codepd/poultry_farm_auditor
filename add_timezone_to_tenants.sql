-- Add timezone column to tenants table
-- Default timezone is 'Asia/Kolkata' (IST) for Indian tenants

ALTER TABLE tenants 
ADD COLUMN IF NOT EXISTS timezone VARCHAR(50) DEFAULT 'Asia/Kolkata';

-- Update existing tenants to have default timezone if NULL
UPDATE tenants 
SET timezone = 'Asia/Kolkata' 
WHERE timezone IS NULL;

-- Add comment
COMMENT ON COLUMN tenants.timezone IS 'IANA timezone identifier (e.g., Asia/Kolkata, America/New_York)';

