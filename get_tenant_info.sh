#!/bin/bash

# Script to get tenant information and UUID

echo "Fetching tenant information..."
echo ""

# Try to get tenant info via psql
psql -U postgres -d poultry_farm -c "
SELECT 
    id::text as tenant_uuid,
    name,
    location,
    country_code,
    currency,
    created_at
FROM tenants 
WHERE id = 1 OR name LIKE '%Pradeep%' OR name LIKE '%Farm%'
LIMIT 5;
" 2>/dev/null

if [ $? -ne 0 ]; then
    echo ""
    echo "⚠️  Could not connect to database via psql"
    echo ""
    echo "Alternative: Use the Go API to get tenant info:"
    echo "  curl http://localhost:8080/api/tenants?tenant_id=1"
    echo ""
    echo "Or update tenant name via SQL:"
    echo "  psql -U postgres -d poultry_farm -c \"UPDATE tenants SET name = 'Pradeep Farm' WHERE id = 1;\""
    echo ""
    echo "Then get the UUID:"
    echo "  psql -U postgres -d poultry_farm -c \"SELECT id::text FROM tenants WHERE name = 'Pradeep Farm';\""
fi


