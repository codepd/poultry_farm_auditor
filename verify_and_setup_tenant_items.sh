#!/bin/bash

# Script to verify and setup tenant items

echo "Checking tenant items configuration..."

# Check if we can connect to the database
cd "$(dirname "$0")"

# Load environment variables
if [ -f .env ]; then
    source .env
    export $(cat .env | grep -v '^#' | xargs)
fi

# Run the SQL script
echo "Setting up tenant items..."
psql "$DATABASE_URL" -f setup_tenant_items.sql

if [ $? -eq 0 ]; then
    echo "✓ Tenant items setup complete!"
    echo ""
    echo "Verifying items..."
    psql "$DATABASE_URL" -c "SELECT t.name, ti.category, COUNT(*) as item_count FROM tenant_items ti JOIN tenants t ON ti.tenant_id = t.id GROUP BY t.name, ti.category ORDER BY t.name, ti.category;"
else
    echo "✗ Failed to setup tenant items. Please run the SQL script manually."
    echo "SQL file: setup_tenant_items.sql"
fi




