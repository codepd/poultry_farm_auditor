#!/bin/bash

# Simple test script to debug owner creation
TENANT_ID="8d7939f7-b716-4eb0-98d4-544c18c8dfb8"
EMAIL="ppradeep0610@gmail.com"
PASSWORD="P@sswd123!"
FULL_NAME="Pradeep"

echo "Testing API connection..."
curl -s http://localhost:8080/health && echo " ✅ API is running" || echo " ❌ API is not running"

echo ""
echo "Testing create-owner endpoint..."
echo ""

# Test with verbose output
curl -v -X POST "http://localhost:8080/api/admin/create-owner" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: local-dev-admin-key" \
  -d "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\",
    \"full_name\": \"$FULL_NAME\",
    \"tenant_id\": \"$TENANT_ID\"
  }" 2>&1

echo ""
echo ""
echo "Done."

