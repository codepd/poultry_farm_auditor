#!/bin/bash

# Script to create a tenant and first owner
# Usage: ./create_tenant_and_owner.sh <tenant_name> <owner_email> <owner_password> <owner_full_name>

if [ "$#" -lt 4 ]; then
    echo "Usage: $0 <tenant_name> <owner_email> <owner_password> <owner_full_name>"
    echo ""
    echo "Example:"
    echo "  $0 'My Poultry Farm' owner@farm.com MyPassword123 'John Doe'"
    exit 1
fi

TENANT_NAME=$1
OWNER_EMAIL=$2
OWNER_PASSWORD=$3
OWNER_FULL_NAME=$4

API_URL="http://localhost:8080/api"

echo "Step 1: Creating tenant..."
echo "  Name: $TENANT_NAME"
echo ""

# First, we need to login as an admin or use a special endpoint
# For now, let's assume we can create tenant directly via admin endpoint
# In production, this would require authentication

# Create tenant
tenant_response=$(curl -s -X POST "$API_URL/admin/create-tenant" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: local-dev-admin-key" \
  -d "{
    \"name\": \"$TENANT_NAME\",
    \"location\": \"\",
    \"country_code\": \"IND\",
    \"currency\": \"INR\",
    \"number_format\": \"lakhs\",
    \"date_format\": \"DD-MM-YYYY\"
  }" 2>/dev/null)

# Extract tenant ID from response
if echo "$tenant_response" | grep -q '"success":true'; then
    # Try to extract UUID from response (format: "id":"uuid-here")
    TENANT_ID=$(echo "$tenant_response" | grep -oE '"id":"[a-f0-9-]{36}"' | cut -d'"' -f4)
    if [ -z "$TENANT_ID" ]; then
        # Try alternative format
        TENANT_ID=$(echo "$tenant_response" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
    fi
    echo "✅ Tenant created with ID: $TENANT_ID"
else
    echo "⚠️  Tenant creation via API failed"
    echo "Response: $tenant_response"
    echo ""
    echo "   To get a tenant ID from database, run:"
    echo "   psql -U postgres -d poultry_farm -c \"SELECT id, name FROM tenants LIMIT 5;\""
    echo ""
    read -p "Enter tenant UUID (or press Enter to exit): " TENANT_ID
    if [ -z "$TENANT_ID" ]; then
        exit 1
    fi
fi

if [ -z "$TENANT_ID" ]; then
    echo "❌ Cannot proceed without tenant ID"
    exit 1
fi

echo ""
echo "Step 2: Creating owner..."
echo "  Email: $OWNER_EMAIL"
echo "  Full Name: $OWNER_FULL_NAME"
echo "  Tenant ID: $TENANT_ID"
echo ""

owner_response=$(curl -s -X POST "$API_URL/admin/create-owner" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: local-dev-admin-key" \
  -d "{
    \"email\": \"$OWNER_EMAIL\",
    \"password\": \"$OWNER_PASSWORD\",
    \"full_name\": \"$OWNER_FULL_NAME\",
    \"tenant_id\": \"$TENANT_ID\"
  }")

if echo "$owner_response" | grep -q '"success":true'; then
    echo "✅ Owner created successfully!"
    echo ""
    echo "=========================================="
    echo "Login Credentials:"
    echo "=========================================="
    echo "  Email: $OWNER_EMAIL"
    echo "  Password: $OWNER_PASSWORD"
    echo "  Tenant: $TENANT_NAME"
    echo ""
    echo "Login at: http://localhost:4300/login"
    echo "=========================================="
else
    echo "❌ Failed to create owner"
    echo "Response: $owner_response"
    exit 1
fi

