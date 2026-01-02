#!/bin/bash

# Script to create the first owner for a poultry farm
# Usage: ./setup_first_owner.sh <tenant_uuid> <email> <password> <full_name>

if [ "$#" -lt 4 ]; then
    echo "Usage: $0 <tenant_uuid> <email> <password> <full_name>"
    echo ""
    echo "Example:"
    echo "  $0 550e8400-e29b-41d4-a716-446655440000 owner@farm.com MyPassword123 'John Doe'"
    exit 1
fi

TENANT_ID=$1
EMAIL=$2
PASSWORD=$3
FULL_NAME=$4

API_URL="http://localhost:8080/api"

echo "Creating owner..."
echo "  Tenant ID: $TENANT_ID"
echo "  Email: $EMAIL"
echo "  Full Name: $FULL_NAME"
echo ""

# Check if API is running first
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "❌ API server is not running on http://localhost:8080"
    echo "   Please start the API server first:"
    echo "   cd go_api && go run main.go"
    exit 1
fi

echo "Calling API endpoint..."

# Properly escape JSON values (handle special characters)
# Use jq if available, otherwise use a simple approach
if command -v jq > /dev/null 2>&1; then
    json_payload=$(jq -n \
      --arg email "$EMAIL" \
      --arg password "$PASSWORD" \
      --arg full_name "$FULL_NAME" \
      --arg tenant_id "$TENANT_ID" \
      '{email: $email, password: $password, full_name: $full_name, tenant_id: $tenant_id}')
else
    # Fallback: manually escape quotes and backslashes
    EMAIL_ESC=$(echo "$EMAIL" | sed 's/"/\\"/g')
    PASSWORD_ESC=$(echo "$PASSWORD" | sed 's/"/\\"/g' | sed 's/\\/\\\\/g')
    FULL_NAME_ESC=$(echo "$FULL_NAME" | sed 's/"/\\"/g')
    json_payload="{\"email\":\"$EMAIL_ESC\",\"password\":\"$PASSWORD_ESC\",\"full_name\":\"$FULL_NAME_ESC\",\"tenant_id\":\"$TENANT_ID\"}"
fi

response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$API_URL/admin/create-owner" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: local-dev-admin-key" \
  -d "$json_payload")

# Extract HTTP code and response body
http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d: -f2)
response_body=$(echo "$response" | sed '/HTTP_CODE:/d')

if [ -z "$http_code" ]; then
    echo "❌ Failed to connect to API server"
    echo "   Make sure the API is running on http://localhost:8080"
    echo "   Response: $response"
    exit 1
fi

if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
    if echo "$response_body" | grep -q '"success":true'; then
        echo "✅ Owner created successfully!"
        echo ""
        echo "Login credentials:"
        echo "  Email: $EMAIL"
        echo "  Password: $PASSWORD"
        echo ""
        echo "Login at: http://localhost:4300/login"
        exit 0
    fi
fi

echo "❌ Failed to create owner (HTTP $http_code)"
echo "Response: $response_body"
exit 1


