# Fix Missing Database Tables

## Issue
The `hen_batches` table is missing from the database, causing a 500 error on `/api/hen-batches`.

## Solution

### Option 1: Using psql (Recommended)
```bash
# Load environment variables
source .env

# Run the SQL script
psql "$DATABASE_URL" -f fix_missing_tables.sql
```

### Option 2: Manual SQL Execution
Connect to your PostgreSQL database and run the contents of `fix_missing_tables.sql`.

### Option 3: Using Python (if psycopg2 is installed)
```bash
cd python_backend
python3 -c "
import os
import sys
sys.path.insert(0, '.')
from database.connection import get_db_connection, return_db_connection

conn = get_db_connection()
cur = conn.cursor()

with open('../fix_missing_tables.sql', 'r') as f:
    cur.execute(f.read())

conn.commit()
cur.close()
return_db_connection(conn)
print('Tables created successfully')
"
```

## What the script does:
1. Creates `hen_batches` table
2. Creates `hen_mortality` table  
3. Creates necessary indexes
4. Creates triggers for automatic timestamp updates and count management
5. Verifies `tenant_items` table exists
6. Ensures `category_enum` type exists

## After running the script:
1. Restart the Go backend (if not already restarted)
2. Test the endpoints:
   - `/api/hen-batches` should return 200 OK (or empty array if no batches)
   - `/api/tenants/items?category=EGG` should return 200 OK

## Verify tables exist:
```sql
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name IN ('hen_batches', 'hen_mortality', 'tenant_items');
```



