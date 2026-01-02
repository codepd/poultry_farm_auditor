#!/bin/bash

# Quick setup script that:
# 1. Updates tenant name to "Pradeep Farm"
# 2. Gets the tenant UUID
# 3. Creates an owner

if [ "$#" -lt 3 ]; then
    echo "Usage: $0 <owner_email> <owner_password> <owner_full_name>"
    echo ""
    echo "Example:"
    echo "  $0 owner@example.com password123 'Pradeep'"
    exit 1
fi

OWNER_EMAIL=$1
OWNER_PASSWORD=$2
OWNER_FULL_NAME=$3

echo "Step 1: Updating tenant name to 'Pradeep Farm'..."
echo ""

# Try to update tenant name
TENANT_UUID=$(psql -U postgres -d poultry_farm -t -c "
    UPDATE tenants 
    SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
    WHERE id = 1 OR name LIKE '%Farm%' OR name LIKE '%Pradeep%'
    RETURNING id::text;
" 2>/dev/null | xargs)

if [ -z "$TENANT_UUID" ]; then
    # Try to get existing tenant UUID
    TENANT_UUID=$(psql -U postgres -d poultry_farm -t -c "
        SELECT id::text FROM tenants 
        WHERE id = 1 OR name LIKE '%Farm%' OR name LIKE '%Pradeep%'
        LIMIT 1;
    " 2>/dev/null | xargs)
    
    if [ -z "$TENANT_UUID" ]; then
        echo "❌ Could not find tenant. Please check database connection."
        echo ""
        echo "Manual steps:"
        echo "  1. Connect to database: psql -U postgres -d poultry_farm"
        echo "  2. Run: SELECT id::text, name FROM tenants WHERE id = 1;"
        echo "  3. Update: UPDATE tenants SET name = 'Pradeep Farm' WHERE id = 1;"
        echo "  4. Get UUID: SELECT id::text FROM tenants WHERE name = 'Pradeep Farm';"
        exit 1
    fi
    
    # Update name separately
    psql -U postgres -d poultry_farm -c "
        UPDATE tenants 
        SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
        WHERE id::text = '$TENANT_UUID';
    " >/dev/null 2>&1
fi

echo "✅ Tenant updated: $TENANT_UUID"
echo ""

echo "Step 2: Creating owner..."
echo "  Email: $OWNER_EMAIL"
echo "  Full Name: $OWNER_FULL_NAME"
echo "  Tenant UUID: $TENANT_UUID"
echo ""

# Check if API is running
API_RUNNING=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/health 2>/dev/null)

if [ "$API_RUNNING" = "200" ]; then
    echo "API is running, creating owner via API..."
    response=$(curl -s -X POST "http://localhost:8080/api/admin/create-owner" \
      -H "Content-Type: application/json" \
      -H "X-Admin-Key: local-dev-admin-key" \
      -d "{
        \"email\": \"$OWNER_EMAIL\",
        \"password\": \"$OWNER_PASSWORD\",
        \"full_name\": \"$OWNER_FULL_NAME\",
        \"tenant_id\": \"$TENANT_UUID\"
      }")
    
    if echo "$response" | grep -q '"success":true'; then
        echo "✅ Owner created successfully!"
        echo ""
        echo "=========================================="
        echo "Login Credentials:"
        echo "=========================================="
        echo "  URL: http://localhost:4300/login"
        echo "  Email: $OWNER_EMAIL"
        echo "  Password: $OWNER_PASSWORD"
        echo "  Tenant: Pradeep Farm"
        echo "=========================================="
    else
        echo "❌ Failed to create owner via API"
        echo "Response: $response"
        exit 1
    fi
else
    echo "⚠️  API is not running on http://localhost:8080"
    echo ""
    echo "Please start the API first:"
    echo "  cd go_api && go run main.go"
    echo ""
    echo "Then run this script again, or create owner manually:"
    echo "  ./setup_first_owner.sh $TENANT_UUID $OWNER_EMAIL $OWNER_PASSWORD '$OWNER_FULL_NAME'"
    exit 1
fi


