#!/usr/bin/env python3
"""
Simple script to update tenant name to "Pradeep Farm"
Works with existing tenant_id (whether integer or UUID)
"""

import sys
import os

# Add parent directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from database.connection import get_db_connection

def update_tenant_name():
    """Update tenant name to 'Pradeep Farm'"""
    conn = None
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        
        # Check current tenant
        cur.execute("SELECT id, name FROM tenants WHERE id = 1 OR name LIKE '%Pradeep%' LIMIT 5")
        tenants = cur.fetchall()
        
        if not tenants:
            print("❌ No tenant found with id=1 or name containing 'Pradeep'")
            print("\nAvailable tenants:")
            cur.execute("SELECT id, name FROM tenants LIMIT 10")
            for row in cur.fetchall():
                print(f"  ID: {row[0]}, Name: {row[1]}")
            return
        
        print("Found tenant(s):")
        for tenant_id, tenant_name in tenants:
            print(f"  ID: {tenant_id}, Name: {tenant_name}")
        
        # Update tenant with id=1 (or first tenant found)
        tenant_id = tenants[0][0]
        
        cur.execute("""
            UPDATE tenants 
            SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
            WHERE id = %s
            RETURNING id, name
        """, (tenant_id,))
        
        updated = cur.fetchone()
        conn.commit()
        
        print(f"\n✅ Updated tenant:")
        print(f"   ID: {updated[0]}")
        print(f"   Name: {updated[1]}")
        print(f"\nUse this ID when creating owner:")
        print(f"   ./setup_first_owner.sh {updated[0]} owner@example.com password123 'Owner Name'")
        
    except Exception as e:
        print(f"❌ Error: {e}")
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


