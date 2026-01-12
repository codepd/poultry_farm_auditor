# Database Schema Files

This directory contains database schema files for the poultry farm management system.

## Files

- **schema.sql**: Complete database schema dump including:
  - All table definitions
  - Enum types (category_enum, transaction_type_enum, user_role_enum, etc.)
  - Functions and triggers
  - Indexes and constraints

## Usage

### Recreate Database Schema

To recreate the database schema from scratch:

```bash
# Create a new database
createdb poultry_farm

# Restore schema
psql -U postgres -d poultry_farm < database/schema.sql
```

Or using Docker:

```bash
docker exec -i poultry_farm_db_new psql -U postgres -d poultry_farm < database/schema.sql
```

### Update Schema

The schema file is generated using:

```bash
docker exec -i poultry_farm_db_new pg_dump -U postgres -d poultry_farm --schema-only --no-owner --no-acl > database/schema.sql
```

## Migration Files

Migration files for specific schema changes are located in the root directory:

- `add_payment_period_fields.sql` - Adds payment period tracking fields to transactions
- `add_expense_to_enum.sql` - Adds EXPENSE and INCOME to transaction_type_enum
- `add_timezone_to_tenants.sql` - Adds timezone configuration to tenants table

## Notes

- The schema file does not include data, only structure
- The schema file does not include ownership information (--no-owner)
- The schema file does not include access control lists (--no-acl)
- Always backup your database before applying schema changes
