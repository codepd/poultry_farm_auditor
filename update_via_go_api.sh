#!/bin/bash

# Script to update tenant name via Go API
# Requires Go API to be running

API_URL="http://localhost:8080/api"

echo "Updating tenant name via Go API..."
echo ""

# Check if API is running
if ! curl -s -o /dev/null -w "%{http_code}" "$API_URL/health" | grep -q "200"; then
    echo "❌ Go API is not running on $API_URL"
    echo ""
    echo "Please start the API first:"
    echo "  cd go_api && go run main.go"
    exit 1
fi

echo "✅ API is running"
echo ""

# For now, we need authentication to update tenant
# This script assumes you'll use the admin endpoint or have a token

echo "To update tenant name, you have two options:"
echo ""
echo "Option 1: Use admin endpoint (if available)"
echo "  curl -X PUT $API_URL/admin/tenants/1 \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'X-Admin-Key: local-dev-admin-key' \\"
echo "    -d '{\"name\": \"Pradeep Farm\"}'"
echo ""
echo "Option 2: Use authenticated endpoint"
echo "  1. Login first to get token"
echo "  2. Then update tenant with token"
echo ""
echo "For now, please use one of the other methods in UPDATE_TENANT_INSTRUCTIONS.md"


