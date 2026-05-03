ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS financial_year_start_month INTEGER DEFAULT 4;

UPDATE tenants
SET financial_year_start_month = 4
WHERE financial_year_start_month IS NULL
   OR financial_year_start_month < 1
   OR financial_year_start_month > 12;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'tenants_financial_year_start_month_check'
    ) THEN
        ALTER TABLE tenants
        ADD CONSTRAINT tenants_financial_year_start_month_check
        CHECK (financial_year_start_month BETWEEN 1 AND 12);
    END IF;
END $$;

COMMENT ON COLUMN tenants.financial_year_start_month IS
'Financial year start month (1-12). Example: 4 for April-March financial year.';
