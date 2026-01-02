# Implementation Plan - Poultry Farm Management System

## Overview
This document outlines the implementation plan for the new requirements and TODO items.

## Requirements Summary

### 1. Multiple Owners per Tenant ✅
- **Database**: `tenant_users` table supports multiple owners via `is_owner` flag
- **Implementation**: Multiple users can have `is_owner = TRUE` for the same tenant

### 2. Sub-Tenants (Location-based) ✅
- **Database**: `sub_tenants` table with `tenant_id` and `location` fields
- **Implementation**: Transactions can be associated with `sub_tenant_id`

### 3. Miscellaneous Expenses/Income ✅
- **Database**: Extended `category_enum` to include 'CHICK', 'GROWER', 'MANURE'
- **Database**: Extended `transaction_type_enum` to include 'EXPENSE', 'INCOME'
- **Implementation**: Transactions can now track purchases of chicks/growers and sales of manure

### 4. App Name Change ✅
- **Frontend**: Changed app name to "Poultry Farm" in `index.html`

### 5. Additional Egg Types ✅
- **Database**: Support for BROKEN EGG, DOUBLE YOLK EGG, DIRT EGG in `price_history` and normalization
- **Python**: Updated `egg_normalizer.py` to handle new egg types
- **Database**: `ledger_breakdowns` tracks EGG_BROKEN, EGG_DOUBLE_YOLK, EGG_DIRT

### 6. Sensitive Data Classification ✅
- **Database**: `sensitive_data_config` table defines what data is sensitive
- **Default**: INCOME, EXPENSE, NET_PROFIT are marked as sensitive

### 7. Role-based Display ✅
- **Database**: `role_permissions` table controls what each role can see/do
- **Implementation**: `can_view_sensitive_data` flag controls access

### 8. Configurable Access ✅
- **Database**: `role_permissions` table allows per-tenant, per-role configuration
- **Default Permissions**:
  - ADMIN, OWNER, CO_OWNER: Can view sensitive data, edit, approve, manage users
  - AUDITOR: Can view sensitive data, view charts (read-only)
  - OTHER_USER: Cannot view sensitive data, can edit transactions (submit for approval)

### 9. Multi-Category Entry Form ⏳
- **Status**: Pending implementation in React frontend
- **Requirement**: Add button to enter multiple egg/feed categories before submitting

### 10. Receipt Upload ⏳
- **Database**: `receipts` table created ✅
- **Status**: Pending implementation in Go API and React frontend
- **Requirement**: Upload receipts for egg sales and feed purchases

## Database Schema Changes

### New Tables
1. `sub_tenants` - Location-based sub-tenants
2. `users` - User accounts
3. `tenant_users` - Many-to-many relationship between tenants and users with roles
4. `invitations` - Email invitations for user onboarding
5. `role_permissions` - Configurable permissions per role per tenant
6. `sensitive_data_config` - Configuration for what data is sensitive
7. `receipts` - File storage for transaction receipts

### Extended Enums
- `transaction_type_enum`: Added 'EXPENSE', 'INCOME'
- `category_enum`: Added 'CHICK', 'GROWER', 'MANURE'
- `user_role_enum`: ADMIN, OWNER, CO_OWNER, OTHER_USER, AUDITOR
- `transaction_status_enum`: DRAFT, SUBMITTED, APPROVED, REJECTED

### Extended Tables
- `transactions`: Added `sub_tenant_id`, `status`, `submitted_by_user_id`, `approved_by_user_id`, `approved_at`
- `tenants`: Added `country_code`, `currency`, `number_format`, `date_format`

## Next Steps

### High Priority
1. **Go API Implementation**
   - User authentication (JWT)
   - Role-based access control middleware
   - Transaction approval endpoints
   - Receipt upload endpoint
   - Sensitive data filtering in analytics endpoints

2. **React Frontend**
   - Multi-category entry form with add button
   - Receipt upload component
   - User management UI
   - Invitation acceptance flow
   - Role-based UI (hide/show sensitive data)
   - Transaction approval workflow UI

3. **Python Backend**
   - Update parsers to handle new egg types
   - Update importers to handle new categories

### Medium Priority
1. Email service for invitations
2. File storage service for receipts
3. Tenant configuration UI
4. Role permissions management UI

## Implementation Status

- ✅ Database schema design and SQL
- ✅ Python database connection and schema initialization
- ✅ Egg normalizer updated for new types
- ✅ App name changed to "Poultry Farm"
- ⏳ Go API implementation (in progress)
- ⏳ React frontend implementation (pending)
- ⏳ Receipt upload functionality (pending)
- ⏳ Multi-category entry form (pending)

