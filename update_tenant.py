#!/usr/bin/env python3
"""
Simple script to update tenant name to "Pradeep Farm"
Direct database connection (no module dependencies)
"""

import os
import sys

try:
    import psycopg2
except ImportError:
    print("Error: psycopg2 not installed")
    print("Install it with: pip install psycopg2-binary")
    print("Or: pip3 install psycopg2-binary")
    sys.exit(1)

def get_db_connection():
    """Get database connection from environment or defaults"""
    host = os.getenv("DB_HOST", "localhost")
    port = os.getenv("DB_PORT", "5432")
    database = os.getenv("DB_NAME", "poultry_farm")
    user = os.getenv("DB_USER", "postgres")
    password = os.getenv("DB_PASSWORD", "postgres")
    
    try:
        conn = psycopg2.connect(
            host=host,
            port=port,
            database=database,
            user=user,
            password=password
        )
        return conn
    except Exception as e:
        print(f"Error connecting to database: {e}")
        print(f"\nConnection details:")
        print(f"  Host: {host}")
        print(f"  Port: {port}")
        print(f"  Database: {database}")
        print(f"  User: {user}")
        print(f"\nMake sure PostgreSQL is running and credentials are correct")
        sys.exit(1)

def update_tenant_name():
    """Update tenant name to 'Pradeep Farm'"""
    conn = None
    try:
        print("Connecting to database...")
        conn = get_db_connection()
        cur = conn.cursor()
        
        # First, check what tenants exist
        print("\nChecking existing tenants...")
        cur.execute("""
            SELECT id, name, 
                   pg_typeof(id)::text as id_type
            FROM tenants 
            WHERE id = 1 OR name LIKE '%Pradeep%' OR name LIKE '%Farm%'
            LIMIT 5
        """)
        
        tenants = cur.fetchall()
        
        if not tenants:
            print("❌ No tenant found with id=1 or name containing 'Pradeep'/'Farm'")
            print("\nAll tenants in database:")
            cur.execute("SELECT id, name FROM tenants LIMIT 10")
            all_tenants = cur.fetchall()
            if all_tenants:
                for tenant_id, tenant_name in all_tenants:
                    print(f"  ID: {tenant_id}, Name: {tenant_name}")
            else:
                print("  (No tenants found)")
            return
        
        print("\nFound tenant(s):")
        for tenant_id, tenant_name, id_type in tenants:
            print(f"  ID: {tenant_id} ({id_type})")
            print(f"  Name: {tenant_name}")
        
        # Use the first tenant found (or id=1 if exists)
        tenant_id = None
        for tid, tname, _ in tenants:
            # Check if tid is 1 (could be int or UUID string "1")
            if tid == 1 or str(tid) == "1" or (isinstance(tid, int) and tid == 1):
                tenant_id = tid
                break
        
        if tenant_id is None:
            tenant_id = tenants[0][0]
        
        print(f"\nUpdating tenant ID: {tenant_id}")
        
        # Update tenant name
        cur.execute("""
            UPDATE tenants 
            SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
            WHERE id = %s
            RETURNING id, name, pg_typeof(id)::text
        """, (tenant_id,))
        
        updated = cur.fetchone()
        conn.commit()
        
        if updated:
            updated_id, updated_name, id_type = updated
            print(f"\n✅ Successfully updated tenant!")
            print(f"   ID: {updated_id} ({id_type})")
            print(f"   Name: {updated_name}")
            
            # Get UUID string format
            cur.execute("SELECT id::text FROM tenants WHERE id = %s", (updated_id,))
            uuid_str = cur.fetchone()[0]
            
            print(f"\n   UUID (string): {uuid_str}")
            print(f"\n" + "="*50)
            print("Next steps:")
            print("="*50)
            print(f"1. Start Go API: cd go_api && go run main.go")
            print(f"2. Create owner:")
            print(f"   ./setup_first_owner.sh {uuid_str} owner@example.com password123 'Owner Name'")
            print("="*50)
        else:
            print("❌ Update failed - no rows updated")
            
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        if conn:
            conn.rollback()
        sys.exit(1)
    finally:
        if conn:
            cur.close()
            conn.close()

if __name__ == "__main__":
    print("="*50)
    print("Update Tenant Name to 'Pradeep Farm'")
    print("="*50)
    print()
    update_tenant_name()


