"""
Database schema definitions for Poultry Farm Management System.
Includes tables for tenants, users, roles, permissions, transactions, and receipts.
"""

# Enable UUID extension
CREATE_UUID_EXTENSION = """
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
"""

# ENUM Types
CREATE_ENUMS = """
-- Transaction types
DO $$ BEGIN
    CREATE TYPE transaction_type_enum AS ENUM ('SALE', 'PURCHASE', 'PAYMENT', 'TDS', 'DISCOUNT', 'EXPENSE', 'INCOME');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Categories
DO $$ BEGIN
    CREATE TYPE category_enum AS ENUM ('EGG', 'FEED', 'MEDICINE', 'OTHER', 'CHICK', 'GROWER', 'MANURE', 'EMPLOYEE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Price types
DO $$ BEGIN
    CREATE TYPE price_type_enum AS ENUM ('EGG', 'FEED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- User roles
DO $$ BEGIN
    CREATE TYPE user_role_enum AS ENUM ('ADMIN', 'OWNER', 'CO_OWNER', 'MANAGER', 'OTHER_USER', 'AUDITOR');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Transaction status
DO $$ BEGIN
    CREATE TYPE transaction_status_enum AS ENUM ('DRAFT', 'SUBMITTED', 'APPROVED', 'REJECTED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;
"""

# Core Tables - Unified Tenants Table
CREATE_TENANTS_TABLE = """
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL for top-level tenants
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    country_code VARCHAR(3) DEFAULT 'IND',
    currency VARCHAR(3) DEFAULT 'INR',
    number_format VARCHAR(20) DEFAULT 'lakhs', -- 'lakhs' or 'millions'
    date_format VARCHAR(20) DEFAULT 'DD-MM-YYYY',
    capacity INTEGER, -- Farm capacity in number of hens (e.g., 45000)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- Ensure unique name within same parent (siblings can't have same name)
    UNIQUE(parent_id, name)
);
"""

CREATE_USERS_TABLE = """
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- NULL until user accepts invite
    full_name VARCHAR(255),
    is_active BOOLEAN DEFAULT FALSE, -- Activated after accepting invite
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
"""

CREATE_TENANT_USERS_TABLE = """
CREATE TABLE IF NOT EXISTS tenant_users (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role user_role_enum NOT NULL,
    is_owner BOOLEAN DEFAULT FALSE, -- Multiple owners allowed
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, user_id)
);
"""

CREATE_INVITATIONS_TABLE = """
CREATE TABLE IF NOT EXISTS invitations (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invited_by_user_id INTEGER NOT NULL REFERENCES users(id),
    email VARCHAR(255) NOT NULL,
    role user_role_enum NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    accepted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
"""

# Tenant Items Configuration Table
CREATE_TENANT_ITEMS_TABLE = """
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
"""

CREATE_ROLE_PERMISSIONS_TABLE = """
CREATE TABLE IF NOT EXISTS role_permissions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role user_role_enum NOT NULL,
    can_view_sensitive_data BOOLEAN DEFAULT FALSE,
    can_edit_transactions BOOLEAN DEFAULT FALSE,
    can_approve_transactions BOOLEAN DEFAULT FALSE,
    can_manage_users BOOLEAN DEFAULT FALSE,
    can_view_charts BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, role)
);
"""

CREATE_SENSITIVE_DATA_CONFIG_TABLE = """
CREATE TABLE IF NOT EXISTS sensitive_data_config (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    data_type VARCHAR(50) NOT NULL, -- 'EGGS_SOLD', 'FEED_PURCHASED', 'NET_PROFIT', 'INCOME', 'EXPENSE', etc.
    is_sensitive BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, data_type)
);
"""

CREATE_TRANSACTIONS_TABLE = """
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    transaction_date DATE NOT NULL,
    transaction_type transaction_type_enum NOT NULL,
    category category_enum NOT NULL,
    item_name VARCHAR(255),
    quantity DECIMAL(12, 3),
    unit VARCHAR(50),
    rate DECIMAL(12, 2),
    amount DECIMAL(12, 2) NOT NULL,
    notes TEXT,
    status transaction_status_enum DEFAULT 'DRAFT',
    submitted_by_user_id INTEGER REFERENCES users(id),
    approved_by_user_id INTEGER REFERENCES users(id),
    approved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
"""

CREATE_RECEIPTS_TABLE = """
CREATE TABLE IF NOT EXISTS receipts (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size INTEGER,
    mime_type VARCHAR(100),
    uploaded_by_user_id INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
"""

CREATE_PRICE_HISTORY_TABLE = """
CREATE TABLE IF NOT EXISTS price_history (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    price_date DATE NOT NULL,
    price_type price_type_enum NOT NULL,
    item_name VARCHAR(255) NOT NULL, -- e.g., "LAYER MASH", "LARGE EGG", "BROKEN EGG", "DOUBLE YOLK", "DIRT EGG"
    price DECIMAL(12, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, price_date, price_type, item_name)
);
"""

CREATE_LEDGER_PARSES_TABLE = """
CREATE TABLE IF NOT EXISTS ledger_parses (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    pdf_filename VARCHAR(255),
    parse_date DATE,
    month INTEGER CHECK (month >= 1 AND month <= 12),
    year INTEGER,
    opening_balance DECIMAL(12, 2),
    closing_balance DECIMAL(12, 2),
    total_eggs DECIMAL(12, 3),
    total_feeds DECIMAL(12, 3),
    total_medicines DECIMAL(12, 2),
    net_profit DECIMAL(12, 2),
    parsed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, year, month, pdf_filename)
);
"""

CREATE_LEDGER_BREAKDOWNS_TABLE = """
CREATE TABLE IF NOT EXISTS ledger_breakdowns (
    id SERIAL PRIMARY KEY,
    ledger_parse_id INTEGER NOT NULL REFERENCES ledger_parses(id) ON DELETE CASCADE,
    breakdown_type VARCHAR(50) NOT NULL, -- e.g., "EGG_LARGE", "EGG_MEDIUM", "EGG_SMALL", "EGG_BROKEN", "EGG_DOUBLE_YOLK", "EGG_DIRT", "FEED_LAYER_MASH", etc.
    quantity DECIMAL(12, 3),
    amount DECIMAL(12, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
"""

CREATE_HEN_BATCHES_TABLE = """
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
"""

CREATE_MORTALITY_TABLE = """
CREATE TABLE IF NOT EXISTS mortality (
    id SERIAL PRIMARY KEY,
    hen_batch_id INTEGER NOT NULL REFERENCES hen_batches(id) ON DELETE CASCADE,
    mortality_date DATE NOT NULL,
    count INTEGER NOT NULL,
    reason VARCHAR(255),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
"""

CREATE_EMPLOYEES_TABLE = """
CREATE TABLE IF NOT EXISTS employees (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    email VARCHAR(255),
    address TEXT,
    designation VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
"""

# Indexes
CREATE_INDEXES = """
-- Transaction indexes
CREATE INDEX IF NOT EXISTS idx_transactions_tenant_date ON transactions(tenant_id, transaction_date);
CREATE INDEX IF NOT EXISTS idx_transactions_tenant_category ON transactions(tenant_id, category, transaction_date);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);

-- Price history indexes
CREATE INDEX IF NOT EXISTS idx_price_history_tenant_type_date ON price_history(tenant_id, price_type, price_date);
CREATE INDEX IF NOT EXISTS idx_price_history_item ON price_history(tenant_id, item_name);

-- Ledger parses indexes
CREATE INDEX IF NOT EXISTS idx_ledger_parses_tenant_year_month ON ledger_parses(tenant_id, year, month);

-- Tenant items indexes
CREATE INDEX IF NOT EXISTS idx_tenant_items_tenant_category ON tenant_items(tenant_id, category);
CREATE INDEX IF NOT EXISTS idx_tenant_items_active ON tenant_items(tenant_id, category, is_active) WHERE is_active = TRUE;
"""

# Initialize default role permissions for all tenants
INIT_ROLE_PERMISSIONS = """
INSERT INTO role_permissions (tenant_id, role, can_view_sensitive_data, can_edit_transactions, can_approve_transactions, can_manage_users, can_view_charts)
SELECT id, 'ADMIN', TRUE, TRUE, TRUE, TRUE, TRUE FROM tenants
ON CONFLICT (tenant_id, role) DO NOTHING;

INSERT INTO role_permissions (tenant_id, role, can_view_sensitive_data, can_edit_transactions, can_approve_transactions, can_manage_users, can_view_charts)
SELECT id, 'OWNER', TRUE, TRUE, TRUE, TRUE, TRUE FROM tenants
ON CONFLICT (tenant_id, role) DO NOTHING;

INSERT INTO role_permissions (tenant_id, role, can_view_sensitive_data, can_edit_transactions, can_approve_transactions, can_manage_users, can_view_charts)
SELECT id, 'CO_OWNER', TRUE, TRUE, TRUE, TRUE, TRUE FROM tenants
ON CONFLICT (tenant_id, role) DO NOTHING;

INSERT INTO role_permissions (tenant_id, role, can_view_sensitive_data, can_edit_transactions, can_approve_transactions, can_manage_users, can_view_charts)
SELECT id, 'AUDITOR', TRUE, FALSE, FALSE, FALSE, TRUE FROM tenants
ON CONFLICT (tenant_id, role) DO NOTHING;

INSERT INTO role_permissions (tenant_id, role, can_view_sensitive_data, can_edit_transactions, can_approve_transactions, can_manage_users, can_view_charts)
SELECT id, 'OTHER_USER', FALSE, TRUE, FALSE, FALSE, FALSE FROM tenants
ON CONFLICT (tenant_id, role) DO NOTHING;
"""

# Initialize default items for all tenants
INIT_TENANT_ITEMS = """
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
"""

def get_all_schema_sql():
    """Return all SQL statements to initialize the database schema."""
    return [
        CREATE_UUID_EXTENSION,
        CREATE_ENUMS,
        CREATE_TENANTS_TABLE,
        CREATE_USERS_TABLE,
        CREATE_TENANT_USERS_TABLE,
        CREATE_INVITATIONS_TABLE,
        CREATE_TENANT_ITEMS_TABLE,
        CREATE_ROLE_PERMISSIONS_TABLE,
        CREATE_SENSITIVE_DATA_CONFIG_TABLE,
        CREATE_TRANSACTIONS_TABLE,
        CREATE_RECEIPTS_TABLE,
        CREATE_PRICE_HISTORY_TABLE,
        CREATE_LEDGER_PARSES_TABLE,
        CREATE_LEDGER_BREAKDOWNS_TABLE,
        CREATE_HEN_BATCHES_TABLE,
        CREATE_MORTALITY_TABLE,
        CREATE_EMPLOYEES_TABLE,
        CREATE_INDEXES,
        INIT_ROLE_PERMISSIONS,
        INIT_TENANT_ITEMS,
    ]
