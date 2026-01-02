#!/usr/bin/env python3
"""
Simple script to update tenant name to "Pradeep Farm"
Uses the database connection module (no psql needed)
"""

import sys
import os

# Add python_backend to path
script_dir = os.path.dirname(os.path.abspath(__file__))
python_backend_dir = os.path.dirname(script_dir)
project_root = os.path.dirname(python_backend_dir)
sys.path.insert(0, python_backend_dir)

try:
    from database.connection import get_db_connection
except ImportError as e:
    print(f"Error: Could not import database connection module: {e}")
    print(f"Script dir: {script_dir}")
    print(f"Python backend dir: {python_backend_dir}")
    print(f"Project root: {project_root}")
    print(f"Python path: {sys.path}")
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
            if tid == 1 or (isinstance(tid, int) and tid == 1):
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

