-- Fix missing tables: hen_batches and verify tenant_items exists

-- Create hen_batches table if it doesn't exist
CREATE TABLE IF NOT EXISTS hen_batches (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    batch_name VARCHAR(255) NOT NULL,
    initial_count INTEGER NOT NULL,
    current_count INTEGER NOT NULL,
    age_weeks INTEGER DEFAULT 0,
    age_days INTEGER DEFAULT 0,
    date_added DATE NOT NULL,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create hen_mortality table if it doesn't exist
CREATE TABLE IF NOT EXISTS hen_mortality (
    id SERIAL PRIMARY KEY,
    batch_id INTEGER NOT NULL REFERENCES hen_batches(id) ON DELETE CASCADE,
    mortality_date DATE NOT NULL,
    count INTEGER NOT NULL,
    reason VARCHAR(255),
    notes TEXT,
    recorded_by_user_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_hen_batches_tenant_id ON hen_batches(tenant_id);
CREATE INDEX IF NOT EXISTS idx_hen_batches_date_added ON hen_batches(date_added);
CREATE INDEX IF NOT EXISTS idx_hen_mortality_batch_id ON hen_mortality(batch_id);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_hen_batches_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_hen_batches_updated_at ON hen_batches;
CREATE TRIGGER trigger_update_hen_batches_updated_at
    BEFORE UPDATE ON hen_batches
    FOR EACH ROW
    EXECUTE FUNCTION update_hen_batches_updated_at();

-- Create trigger to automatically update current_count when mortality is recorded
CREATE OR REPLACE FUNCTION update_batch_count_on_mortality()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE hen_batches
    SET current_count = current_count - NEW.count,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.batch_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_batch_count_on_mortality ON hen_mortality;
CREATE TRIGGER trigger_update_batch_count_on_mortality
    AFTER INSERT ON hen_mortality
    FOR EACH ROW
    EXECUTE FUNCTION update_batch_count_on_mortality();

-- Verify tenant_items table exists (should already exist, but check)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM information_schema.tables 
                   WHERE table_schema = 'public' 
                   AND table_name = 'tenant_items') THEN
        RAISE EXCEPTION 'tenant_items table does not exist. Please run the tenant_items setup script first.';
    END IF;
END $$;

-- Verify category_enum exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_type WHERE typname = 'category_enum') THEN
        CREATE TYPE category_enum AS ENUM ('EGG', 'FEED', 'MEDICINE', 'OTHER', 'CHICK', 'GROWER', 'MANURE', 'EMPLOYEE');
    END IF;
END $$;

