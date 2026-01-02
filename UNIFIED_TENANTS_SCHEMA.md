# Unified Tenants Schema with UUID

## Overview
The `tenants` and `child_tenants` tables have been merged into a single `tenants` table with UUID primary keys and hierarchical support via `parent_id`.

## Key Changes

### 1. Single Table for All Tenants
- **Before**: Separate `tenants` and `child_tenants` tables
- **After**: Single `tenants` table with `parent_id` for hierarchy

### 2. UUID Primary Keys
- **Before**: `SERIAL` (INTEGER) primary keys
- **After**: `UUID` primary keys using `uuid_generate_v4()`
- **Requirement**: UUID extension enabled (`uuid-ossp`)

### 3. Hierarchical Structure
- `parent_id UUID REFERENCES tenants(id)` - NULL for top-level tenants
- Self-referential foreign key allows unlimited nesting
- `UNIQUE(parent_id, name)` ensures unique names within same parent

### 4. Removed `child_tenant_id` References
- All tables now reference `tenant_id` only (UUID)
- No distinction between tenant and child tenant in foreign keys
- Sensitive data config is per tenant (any level in hierarchy)

## Database Schema

### Unified Tenants Table
```sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL for top-level
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    country_code VARCHAR(3) DEFAULT 'IND',
    currency VARCHAR(3) DEFAULT 'INR',
    number_format VARCHAR(20) DEFAULT 'lakhs',
    date_format VARCHAR(20) DEFAULT 'DD-MM-YYYY',
    capacity INTEGER, -- Farm capacity
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(parent_id, name) -- Unique name within same parent
);
```

## Updated Foreign Keys

All tables now use `UUID` for `tenant_id`:
- `tenant_users.tenant_id` → `UUID`
- `invitations.tenant_id` → `UUID`
- `role_permissions.tenant_id` → `UUID`
- `sensitive_data_config.tenant_id` → `UUID`
- `transactions.tenant_id` → `UUID`
- `price_history.tenant_id` → `UUID`
- `ledger_parses.tenant_id` → `UUID`
- `hen_batches.tenant_id` → `UUID`
- `employees.tenant_id` → `UUID`

## Hierarchy Example

```
Tenant: "ABC Poultry Farm" (id: uuid-1, parent_id: NULL)
├── Tenant: "Farm Location A" (id: uuid-2, parent_id: uuid-1)
│   ├── Tenant: "Building 1" (id: uuid-3, parent_id: uuid-2)
│   └── Tenant: "Building 2" (id: uuid-4, parent_id: uuid-2)
└── Tenant: "Farm Location B" (id: uuid-5, parent_id: uuid-1)
    └── Tenant: "Building 1" (id: uuid-6, parent_id: uuid-5)
```

## Querying Hierarchy

### Get All Children of a Tenant
```sql
WITH RECURSIVE tenant_hierarchy AS (
    -- Base case: direct children
    SELECT id, name, parent_id, 1 as level
    FROM tenants
    WHERE parent_id = 'uuid-1'
    
    UNION ALL
    
    -- Recursive case: children of children
    SELECT t.id, t.name, t.parent_id, th.level + 1
    FROM tenants t
    INNER JOIN tenant_hierarchy th ON t.parent_id = th.id
)
SELECT * FROM tenant_hierarchy ORDER BY level, name;
```

### Get All Ancestors of a Tenant
```sql
WITH RECURSIVE tenant_ancestors AS (
    -- Base case: start with the tenant
    SELECT id, name, parent_id, 0 as level
    FROM tenants
    WHERE id = 'uuid-3'
    
    UNION ALL
    
    -- Recursive case: get parent
    SELECT t.id, t.name, t.parent_id, ta.level + 1
    FROM tenants t
    INNER JOIN tenant_ancestors ta ON t.id = ta.parent_id
)
SELECT * FROM tenant_ancestors ORDER BY level DESC;
```

## Benefits

1. **Simplified Schema**: Single table instead of two
2. **UUID Benefits**: 
   - Globally unique identifiers
   - No sequence conflicts in distributed systems
   - Better for replication and sharding
3. **Flexible Hierarchy**: Unlimited nesting levels
4. **Consistent References**: All tables use same `tenant_id` type (UUID)
5. **Unique Constraint**: `UNIQUE(parent_id, name)` ensures no duplicate names within same parent

## Migration Notes

### Required Changes
1. Enable UUID extension: `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`
2. Convert all `INTEGER` tenant_id references to `UUID`
3. Update trigger functions to use `UUID` type
4. Remove `child_tenant_id` columns from all tables
5. Update sensitive data config to use single `tenant_id` (no separate child_tenant_id)

### Backward Compatibility
- Existing data will need migration script
- UUIDs can be generated for existing INTEGER IDs
- Application code needs to handle UUID instead of INTEGER

## Go API Models

### Updated Tenant Model
```go
type Tenant struct {
    ID          uuid.UUID  `json:"id"`
    ParentID    *uuid.UUID `json:"parent_id,omitempty"`
    Name        string     `json:"name"`
    Location    string     `json:"location,omitempty"`
    CountryCode string     `json:"country_code"`
    Currency    string     `json:"currency"`
    NumberFormat string    `json:"number_format"`
    DateFormat  string     `json:"date_format"`
    Capacity    *int       `json:"capacity,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Helper Methods
- `IsTopLevel()`: Returns true if `parent_id` is NULL
- `GetAncestors()`: Returns all parent tenants up to root
- `GetChildren()`: Returns all direct children
- `GetDescendants()`: Returns all descendants (recursive)

