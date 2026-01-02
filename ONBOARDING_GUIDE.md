# Onboarding Guide

## Overview

There are two ways to onboard users to a poultry farm:

1. **Direct Creation (Local Development)** - Create users directly with password
2. **Invitation Flow (Production)** - Send email invitation, user sets password via link

## Method 1: Direct Creation (Local Development)

For local development, you can directly create an owner using the admin endpoint:

### Step 1: Create a Tenant (if not exists)

First, you need a tenant. You can create one via the API or directly in the database.

### Step 2: Create Owner Directly

```bash
curl -X POST http://localhost:8080/api/admin/create-owner \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: local-dev-admin-key" \
  -d '{
    "email": "owner@example.com",
    "password": "your-password-here",
    "full_name": "Owner Name",
    "tenant_id": "your-tenant-uuid"
  }'
```

**Note**: The password can be anything you choose (minimum 6 characters recommended).

### Step 3: Login

The user can now login at `http://localhost:4300/login` with:
- Email: `owner@example.com`
- Password: `your-password-here`

## Method 2: Invitation Flow (Production)

### Step 1: Invite User

An existing owner/admin invites a user:

```bash
curl -X POST http://localhost:8080/api/users/invite \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -d '{
    "email": "newuser@example.com",
    "role": "OWNER"
  }'
```

### Step 2: Response

The API returns an invitation link:
```json
{
  "success": true,
  "message": "Invitation sent",
  "data": {
    "invitation_id": 1,
    "email": "newuser@example.com",
    "token": "abc123...",
    "expires_at": "2025-01-15T10:00:00Z",
    "invitation_link": "http://localhost:4300/accept-invite?token=abc123..."
  }
}
```

### Step 3: Send Email (Manual for Local)

For local development, you can:
1. Copy the `invitation_link` from the response
2. Send it manually via email or share it directly
3. Or open it in a browser to test

### Step 4: User Accepts Invitation

When the user clicks the link:
1. They are taken to `/accept-invite?token=abc123...`
2. They enter their full name and set a password
3. Account is activated and they're added to the tenant
4. They can then login

## Creating the First Owner

Since you need an owner to invite others, use Method 1 (Direct Creation) for the first owner.

### Quick Setup Script

Create a file `setup_first_owner.sh`:

```bash
#!/bin/bash

# First, get or create a tenant UUID
# You can get this from the database or create one via API

TENANT_ID="your-tenant-uuid-here"
EMAIL="owner@example.com"
PASSWORD="ChangeMe123!"
FULL_NAME="Farm Owner"

curl -X POST http://localhost:8080/api/admin/create-owner \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: local-dev-admin-key" \
  -d "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\",
    \"full_name\": \"$FULL_NAME\",
    \"tenant_id\": \"$TENANT_ID\"
  }"

echo ""
echo "Owner created!"
echo "Email: $EMAIL"
echo "Password: $PASSWORD"
echo "Login at: http://localhost:4300/login"
```

## Environment Variables

For the invitation flow to work properly, set:

```bash
# In go_api/.env or environment
FRONTEND_URL=http://localhost:4300  # For local development
# FRONTEND_URL=https://yourdomain.com  # For production
```

## Database Setup

Before onboarding users, ensure:

1. Database is initialized:
   ```bash
   cd python_backend
   python3 cli/init_database.py
   ```

2. At least one tenant exists (create via API or directly in DB)

## Security Notes

- **Local Development**: The admin endpoint uses `X-Admin-Key: local-dev-admin-key` for convenience
- **Production**: Remove or secure the admin endpoint, use only invitation flow
- **Passwords**: Minimum 6 characters, but recommend 8+ with complexity
- **Invitation Tokens**: Expire after 7 days by default


