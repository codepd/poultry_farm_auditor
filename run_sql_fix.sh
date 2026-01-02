#!/bin/bash
# Script to create missing database tables

echo "Creating missing database tables..."
echo ""

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL not set in .env file"
    exit 1
fi

# Try to run SQL using psql if available
if command -v psql &> /dev/null; then
    echo "Running SQL script using psql..."
    psql "$DATABASE_URL" -f fix_missing_tables.sql
    if [ $? -eq 0 ]; then
        echo "✓ Successfully created missing tables"
    else
        echo "✗ Error running SQL script"
        exit 1
    fi
else
    echo "psql not found. Please run the SQL script manually:"
    echo ""
    echo "  psql \$DATABASE_URL -f fix_missing_tables.sql"
    echo ""
    echo "Or connect to your database and run the contents of fix_missing_tables.sql"
    exit 1
fi



