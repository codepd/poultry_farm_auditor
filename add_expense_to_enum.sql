-- Add EXPENSE to transaction_type_enum if it doesn't exist
DO $$ 
BEGIN
    -- Check if EXPENSE already exists in the enum
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum 
        WHERE enumlabel = 'EXPENSE' 
        AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'transaction_type_enum')
    ) THEN
        -- Add EXPENSE to the enum
        ALTER TYPE transaction_type_enum ADD VALUE 'EXPENSE';
    END IF;
END $$;
