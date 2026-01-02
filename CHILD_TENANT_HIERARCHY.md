# Child Tenant Hierarchy Design

## Overview
The system now uses `child_tenants` instead of `sub_tenants` to support a hierarchical structure where a child tenant can itself be a parent of another child tenant.

## Database Schema

### `child_tenants` Table
```sql
CREATE TABLE child_tenants (
    id SERIAL PRIMARY KEY,
    -- Hierarchical structure: can be child of tenant or another child_tenant
    parent_tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
    parent_child_tenant_id INTEGER REFERENCES child_tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    capacity INTEGER, -- Farm capacity in number of hens
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- Ensure exactly one parent is set
    CHECK (
        (parent_tenant_id IS NOT NULL AND parent_child_tenant_id IS NULL) OR
        (parent_tenant_id IS NULL AND parent_child_tenant_id IS NOT NULL)
    )
);
```

## Hierarchy Structure

### Levels
1. **Top Level**: `tenants` (e.g., "ABC Poultry Farm")
2. **Level 1**: Direct children of tenants (e.g., "Farm Location A")
3. **Level 2+**: Children of child tenants (e.g., "Building 1" under "Farm Location A")

### Example Hierarchy
```
Tenant: "ABC Poultry Farm"
├── Child Tenant: "Farm Location A" (parent_tenant_id = 1)
│   ├── Child Tenant: "Building 1" (parent_child_tenant_id = 1)
│   └── Child Tenant: "Building 2" (parent_child_tenant_id = 1)
└── Child Tenant: "Farm Location B" (parent_tenant_id = 1)
    └── Child Tenant: "Building 1" (parent_child_tenant_id = 3)
```

## Key Features

### 1. Flexible Hierarchy
- A child tenant can be a direct child of a tenant (`parent_tenant_id` set)
- A child tenant can be a child of another child tenant (`parent_child_tenant_id` set)
- Exactly one parent must be set (enforced by CHECK constraint)

### 2. Data Association
All data can be associated with any level:
- **Transactions**: Can be associated with tenant or any child tenant
- **Hen Batches**: Can be associated with tenant or any child tenant
- **Employees**: Can be associated with tenant or any child tenant
- **Sensitive Data Config**: Can be set at tenant level or any child tenant level

### 3. Sensitive Data Configuration
- **Tenant Level**: Default configuration for all child tenants
- **Child Tenant Level**: Can override tenant-level configuration
- **Hierarchy**: Checks child tenant config first, then parent, then tenant

### 4. Capacity Management
- Each child tenant can have its own `capacity` (number of hens)
- Useful for tracking capacity at different hierarchy levels

## Usage Examples

### Creating a Top-Level Child Tenant
```sql
INSERT INTO child_tenants (parent_tenant_id, name, location, capacity)
VALUES (1, 'Farm Location A', 'City A', 45000);
```

### Creating a Nested Child Tenant
```sql
INSERT INTO child_tenants (parent_child_tenant_id, name, location, capacity)
VALUES (1, 'Building 1', 'City A - Building 1', 20000);
```

### Querying Child Tenants by Level
```sql
-- Get all direct children of a tenant
SELECT * FROM child_tenants WHERE parent_tenant_id = 1;

-- Get all children of a child tenant
SELECT * FROM child_tenants WHERE parent_child_tenant_id = 1;

-- Get full hierarchy (requires recursive query)
WITH RECURSIVE tenant_hierarchy AS (
    -- Base case: direct children of tenant
    SELECT id, name, parent_tenant_id, parent_child_tenant_id, 1 as level
    FROM child_tenants
    WHERE parent_tenant_id = 1
    
    UNION ALL
    
    -- Recursive case: children of child tenants
    SELECT ct.id, ct.name, ct.parent_tenant_id, ct.parent_child_tenant_id, th.level + 1
    FROM child_tenants ct
    INNER JOIN tenant_hierarchy th ON ct.parent_child_tenant_id = th.id
)
SELECT * FROM tenant_hierarchy ORDER BY level, name;
```

## Migration Notes

### Renamed References
- `sub_tenants` → `child_tenants`
- `sub_tenant_id` → `child_tenant_id`
- All related indexes and foreign keys updated

### Backward Compatibility
- Old `sub_tenant_id` references need to be migrated to `child_tenant_id`
- Database migration script should handle this

## Benefits

1. **Flexibility**: Support for complex organizational structures
2. **Scalability**: Can handle farms with multiple locations and buildings
3. **Granular Control**: Sensitive data config can be set at any level
4. **Capacity Tracking**: Track capacity at different hierarchy levels
5. **Data Organization**: Better organization of transactions, batches, employees by location/building

## Go API Models

### ChildTenant Model
```go
type ChildTenant struct {
    ID                  int       `json:"id"`
    ParentTenantID      *int      `json:"parent_tenant_id,omitempty"`
    ParentChildTenantID *int      `json:"parent_child_tenant_id,omitempty"`
    Name                string    `json:"name"`
    Location            string    `json:"location,omitempty"`
    Capacity            *int      `json:"capacity,omitempty"`
    CreatedAt           time.Time `json:"created_at"`
    UpdatedAt           time.Time `json:"updated_at"`
}
```

### Helper Methods
- `GetParentID()`: Returns the parent ID (either tenant or child tenant)
- `IsTopLevel()`: Returns true if this is a direct child of a tenant

