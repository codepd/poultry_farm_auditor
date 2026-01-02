# Additional Requirements Implementation

## Overview
This document outlines the implementation of additional requirements for hen management, employee management, and enhanced sensitive data configuration.

## ✅ Implemented Features

### 1. Sensitive Data Configuration (Tenant & Sub-Tenant Level)
- **Database**: `sensitive_data_config` table now supports `sub_tenant_id`
- **Logic**: Sub-tenant level config takes precedence over tenant level config
- **Default Sensitive Items**:
  - `EGGS_SOLD` - Eggs sold transactions
  - `FEED_PURCHASED` - Feed purchase transactions
  - `NET_PROFIT` - Net profit calculations
  - `INCOME` - Income transactions
  - `EXPENSE` - Expense transactions
- **Helper Function**: `CheckSensitiveDataConfig()` in Go API checks sub-tenant first, then tenant

### 2. Hen Management
- **Hen Batches Table** (`hen_batches`):
  - Batch name (e.g., "Batch 1", "Batch A")
  - Initial count and current count
  - Age tracking: `age_weeks` and `age_days` (format: "16W 2D")
  - Date added
  - Associated with tenant and optionally sub-tenant
  - **Non-sensitive data** (always visible)

- **Hen Mortality Table** (`hen_mortality`):
  - Daily mortality tracking
  - Automatic count reduction via database trigger
  - Reason and notes for mortality
  - Recorded by user

- **Database Trigger**: Automatically updates `current_count` in `hen_batches` when mortality is recorded

### 3. Employee Management
- **Employees Table** (`employees`):
  - Full name
  - Contact details: Phone, Email, Address
  - Designation (job title/role)
  - Active status
  - Associated with tenant and optionally sub-tenant

- **Employee Expenses**: Added `EMPLOYEE` to `category_enum` for tracking employee costs

### 4. Farm Capacity
- **Sub-Tenants Table**: Added `capacity` field (e.g., 45000 hens)
- Capacity can be set per sub-tenant (location)

## Database Schema Changes

### New Tables
1. `hen_batches` - Hen batch management with age tracking
2. `hen_mortality` - Daily mortality records with automatic count updates
3. `employees` - Employee contact details and information

### Updated Tables
1. `sensitive_data_config` - Added `sub_tenant_id` for sub-tenant level configuration
2. `sub_tenants` - Added `capacity` field for farm capacity

### New Database Functions
1. `update_hen_batch_count_on_mortality()` - Automatically updates batch count on mortality insert/update/delete

### New Triggers
1. `trigger_update_batch_on_mortality_insert` - Updates count on mortality insert
2. `trigger_update_batch_on_mortality_update` - Updates count on mortality update
3. `trigger_update_batch_on_mortality_delete` - Updates count on mortality delete

## Go API Models

### New Models
- `HenBatch` - With `AgeString()` method for "16W 2D" format
- `HenMortality` - Mortality tracking
- `Employee` - Employee management
- `SensitiveDataConfig` - Configuration model

### New Utilities
- `utils/sensitive_data.go` - Helper functions for checking sensitive data config
  - `CheckSensitiveDataConfig()` - Checks sub-tenant first, then tenant
  - `IsDataSensitive()` - Determines if data should be hidden based on user permissions

## Default Sensitive Data Configuration

At tenant level (applies to all sub-tenants unless overridden):
- ✅ `EGGS_SOLD` - Sensitive
- ✅ `FEED_PURCHASED` - Sensitive
- ✅ `NET_PROFIT` - Sensitive
- ✅ `INCOME` - Sensitive
- ✅ `EXPENSE` - Sensitive

## Hen Age Format

Age is stored as:
- `age_weeks` (INTEGER) - Weeks (e.g., 16)
- `age_days` (INTEGER) - Additional days (e.g., 2)

Display format: "16W 2D" (via `HenBatch.AgeString()` method)

## Example Usage

### Creating a Hen Batch
```sql
INSERT INTO hen_batches (tenant_id, batch_name, initial_count, current_count, age_weeks, age_days, date_added)
VALUES (1, 'Batch 1', 20000, 20000, 16, 2, '2025-01-15');
```

### Recording Mortality
```sql
INSERT INTO hen_mortality (batch_id, mortality_date, count, reason)
VALUES (1, '2025-12-01', 5, 'Natural causes');
-- This automatically decreases current_count in hen_batches by 5
```

### Sub-Tenant Sensitive Data Override
```sql
-- Tenant level: EGGS_SOLD is sensitive
-- Sub-tenant level: Make EGGS_SOLD non-sensitive for a specific location
INSERT INTO sensitive_data_config (tenant_id, sub_tenant_id, data_type, is_sensitive)
VALUES (1, 2, 'EGGS_SOLD', FALSE);
```

## Future Enhancements (ML Recommendations)

### Planned Feature
- **ML-based Recommendations**: Based on hen age, suggest:
  - Antibiotics
  - Proteins
  - Shell GRIT
  - Other supplements

This will be implemented later as a separate feature.

## Next Steps

1. **Go API Endpoints**:
   - Hen batch CRUD operations
   - Mortality entry endpoint
   - Employee CRUD operations
   - Sensitive data config management (tenant & sub-tenant)

2. **React Frontend**:
   - Hen batch management UI
   - Mortality entry form
   - Employee management page
   - Sensitive data configuration UI

3. **Age Calculation**:
   - Automatic age calculation based on `date_added`
   - Age progression tracking

4. **ML Integration** (Future):
   - Age-based recommendation engine
   - Integration with medicine/supplement tracking

