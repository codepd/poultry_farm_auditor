# Timezone Configuration for Tenants

This document describes how timezone configuration works for tenants in the Poultry Farm Management System.

## Overview

Each tenant can now have a configurable timezone (IANA timezone identifier, e.g., `Asia/Kolkata`, `America/New_York`). This allows the system to display dates and times according to the tenant's local timezone.

## Database Migration

To add timezone support to existing databases, run the migration script:

```bash
psql -U your_user -d your_database -f add_timezone_to_tenants.sql
```

Or manually execute:

```sql
ALTER TABLE tenants 
ADD COLUMN IF NOT EXISTS timezone VARCHAR(50) DEFAULT 'Asia/Kolkata';

UPDATE tenants 
SET timezone = 'Asia/Kolkata' 
WHERE timezone IS NULL;
```

## Default Timezone

- **Default**: `Asia/Kolkata` (IST - Indian Standard Time)
- This is set automatically for new tenants if not specified
- Existing tenants are updated to `Asia/Kolkata` during migration

## Backend Changes

### Go API

1. **Tenant Model** (`go_api/models/tenant.go`):
   - Added `Timezone string` field to `Tenant` struct
   - Added `Timezone` to `TenantCreateRequest` and `TenantUpdateRequest`

2. **Tenant Handlers** (`go_api/handlers/tenants.go`):
   - All queries now include `timezone` in SELECT statements
   - `CreateTenant` sets default timezone to `Asia/Kolkata` if not provided
   - `UpdateTenant` allows updating timezone

3. **API Endpoints**:
   - `GET /api/tenants/{id}` - Returns tenant with timezone
   - `PUT /api/tenants/{id}` - Allows updating timezone

## Frontend Changes

### New Files

1. **`react_frontend/src/utils/dateUtils.ts`**:
   - `formatDateForTenant()` - Format dates according to tenant's timezone and date format
   - `getCurrentDateInTimezone()` - Get current date in tenant's timezone
   - `toISODateInTimezone()` - Convert date to ISO string in tenant's timezone
   - `formatDateTimeForTenant()` - Format date and time for display

2. **`react_frontend/src/hooks/useTenant.ts`**:
   - `useTenant()` hook - Fetches and provides tenant details including timezone
   - Returns: `{ tenant, loading, error, timezone, dateFormat }`

### Updated Files

1. **`react_frontend/src/services/api.ts`**:
   - Added `Tenant` interface with `timezone` field
   - Added `tenantsAPI` with `getTenant()`, `getTenants()`, and `updateTenant()`

## Usage Examples

### Using the useTenant Hook

```typescript
import { useTenant } from '../hooks/useTenant';
import { formatDateForTenant } from '../utils/dateUtils';

function MyComponent() {
  const { timezone, dateFormat, loading } = useTenant();

  if (loading) return <div>Loading...</div>;

  const formattedDate = formatDateForTenant(
    new Date(),
    timezone,
    dateFormat
  );

  return <div>Date: {formattedDate}</div>;
}
```

### Formatting Transaction Dates

```typescript
import { useTenant } from '../hooks/useTenant';
import { formatDateForTenant } from '../utils/dateUtils';

function TransactionList({ transactions }) {
  const { timezone, dateFormat } = useTenant();

  return (
    <ul>
      {transactions.map(txn => (
        <li key={txn.id}>
          {formatDateForTenant(txn.transaction_date, timezone, dateFormat)}
        </li>
      ))}
    </ul>
  );
}
```

### Getting Current Date in Tenant Timezone

```typescript
import { useTenant } from '../hooks/useTenant';
import { getCurrentDateInTimezone } from '../utils/dateUtils';

function DateInput() {
  const { timezone } = useTenant();
  const today = getCurrentDateInTimezone(timezone);
  
  return <input type="date" defaultValue={today.toISOString().split('T')[0]} />;
}
```

## Supported Timezones

Any valid IANA timezone identifier is supported. Common examples:

- `Asia/Kolkata` - Indian Standard Time (IST)
- `America/New_York` - Eastern Time (ET)
- `Europe/London` - Greenwich Mean Time (GMT)
- `Asia/Dubai` - Gulf Standard Time (GST)
- `Australia/Sydney` - Australian Eastern Time (AET)

For a complete list, see: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones

## Updating Tenant Timezone

### Via API

```bash
curl -X PUT http://localhost:8080/api/tenants/{tenant_id} \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "timezone": "America/New_York"
  }'
```

### Via Frontend (Future Enhancement)

A tenant settings page can be created to allow users to update their timezone preference.

## Notes

- All dates stored in the database remain in UTC
- Timezone conversion happens at display time in the frontend
- The backend can also use timezone information for server-side date formatting if needed
- Date format (e.g., `DD-MM-YYYY`) is separate from timezone and can be configured independently


