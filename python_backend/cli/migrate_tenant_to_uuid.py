#!/usr/bin/env python3
"""
Migration script to rename tenant to "Pradeep Farm"
and ensure tenant uses UUID (if not already)
"""

import sys
import os

# Add parent directory to path to import database module
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..'))

try:
    import psycopg2
    import uuid
except ImportError:
    print("Error: psycopg2 not installed. Install with: pip install psycopg2-binary")
    sys.exit(1)

def get_db_connection():
    """Get database connection"""
    import os
    try:
        conn = psycopg2.connect(
            host=os.getenv("DB_HOST", "localhost"),
            port=os.getenv("DB_PORT", "5432"),
            database=os.getenv("DB_NAME", "poultry_farm"),
            user=os.getenv("DB_USER", "postgres"),
            password=os.getenv("DB_PASSWORD", "postgres")
        )
        return conn
    except Exception as e:
        print(f"Error connecting to database: {e}")
        sys.exit(1)

def migrate_tenant_to_uuid():
    """Migrate tenant from integer ID to UUID"""
    conn = get_db_connection()
    cur = conn.cursor()
    
    try:
        # Start transaction
        conn.autocommit = False
        
        # Step 1: Check if tenant with id=1 exists
        cur.execute("SELECT id, name FROM tenants WHERE id = 1")
        tenant_row = cur.fetchone()
        
        if not tenant_row:
            print("❌ Tenant with id=1 not found!")
            print("Available tenants:")
            cur.execute("SELECT id, name FROM tenants LIMIT 10")
            for row in cur.fetchall():
                print(f"  ID: {row[0]}, Name: {row[1]}")
            conn.rollback()
            return
        
        old_id = tenant_row[0]
        old_name = tenant_row[1]
        print(f"Found tenant: ID={old_id}, Name={old_name}")
        
        # Step 2: Generate new UUID
        new_uuid = uuid.uuid4()
        print(f"Generated new UUID: {new_uuid}")
        
        # Step 3: Check if tenant table already uses UUID
        cur.execute("""
            SELECT data_type 
            FROM information_schema.columns 
            WHERE table_name = 'tenants' AND column_name = 'id'
        """)
        id_type = cur.fetchone()[0]
        
        if id_type == 'uuid':
            print("✅ Tenant table already uses UUID")
            # Check if tenant with this UUID already exists
            cur.execute("SELECT id, name FROM tenants WHERE id = %s", (str(new_uuid),))
            existing = cur.fetchone()
            if existing:
                print(f"⚠️  UUID {new_uuid} already exists, using existing tenant")
                new_uuid = uuid.UUID(existing[0])
            else:
                # Update existing tenant to use new UUID
                print("Updating tenant to use UUID...")
                cur.execute("""
                    UPDATE tenants 
                    SET id = %s, name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
                    WHERE id = 1
                """, (str(new_uuid),))
        else:
            print("⚠️  Tenant table uses integer, need to migrate...")
            print("This requires updating all foreign key references.")
            print("Tables that reference tenant_id:")
            
            # List all tables with tenant_id
            cur.execute("""
                SELECT table_name, column_name, data_type
                FROM information_schema.columns
                WHERE column_name LIKE '%tenant_id%'
                ORDER BY table_name
            """)
            
            tables_to_update = []
            for row in cur.fetchall():
                table_name, column_name, data_type = row
                print(f"  - {table_name}.{column_name} ({data_type})")
                if data_type == 'integer':
                    tables_to_update.append((table_name, column_name))
            
            if tables_to_update:
                print(f"\n⚠️  Found {len(tables_to_update)} tables with integer tenant_id")
                print("This migration will:")
                print("  1. Create a new tenant record with UUID")
                print("  2. Update all foreign key references")
                print("  3. Delete the old tenant record")
                
                response = input("\nProceed with migration? (yes/no): ")
                if response.lower() != 'yes':
                    print("Migration cancelled")
                    conn.rollback()
                    return
                
                # Create new tenant with UUID
                cur.execute("""
                    INSERT INTO tenants (id, name, location, country_code, currency, 
                                        number_format, date_format, capacity, created_at, updated_at)
                    SELECT %s, 'Pradeep Farm', location, country_code, currency,
                           number_format, date_format, capacity, created_at, CURRENT_TIMESTAMP
                    FROM tenants
                    WHERE id = %s
                    RETURNING id
                """, (str(new_uuid), old_id))
                
                new_tenant_id = cur.fetchone()[0]
                print(f"✅ Created new tenant with UUID: {new_tenant_id}")
                
                # Update all foreign key references
                for table_name, column_name in tables_to_update:
                    print(f"Updating {table_name}.{column_name}...")
                    cur.execute(f"""
                        UPDATE {table_name}
                        SET {column_name} = %s
                        WHERE {column_name} = %s
                    """, (str(new_uuid), old_id))
                    updated = cur.rowcount
                    print(f"  ✅ Updated {updated} rows")
                
                # Delete old tenant
                cur.execute("DELETE FROM tenants WHERE id = %s", (old_id,))
                print(f"✅ Deleted old tenant record (id={old_id})")
            else:
                print("✅ No tables need updating")
        
        # Step 4: Update tenant name to "Pradeep Farm" if not already
        cur.execute("SELECT name FROM tenants WHERE id = %s", (str(new_uuid),))
        current_name = cur.fetchone()[0]
        
        if current_name != "Pradeep Farm":
            cur.execute("""
                UPDATE tenants 
                SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
                WHERE id = %s
            """, (str(new_uuid),))
            print(f"✅ Updated tenant name to 'Pradeep Farm'")
        else:
            print(f"✅ Tenant name is already 'Pradeep Farm'")
        
        # Commit transaction
        conn.commit()
        
        # Display final tenant info
        cur.execute("""
            SELECT id, name, location, country_code, currency, created_at
            FROM tenants WHERE id = %s
        """, (str(new_uuid),))
        final_tenant = cur.fetchone()
        
        print("\n" + "="*50)
        print("✅ Migration completed successfully!")
        print("="*50)
        print(f"Tenant UUID: {final_tenant[0]}")
        print(f"Tenant Name: {final_tenant[1]}")
        print(f"Location: {final_tenant[2] or 'N/A'}")
        print(f"Country: {final_tenant[3]}")
        print(f"Currency: {final_tenant[4]}")
        print("="*50)
        print(f"\nUse this UUID when creating the owner:")
        print(f"  ./setup_first_owner.sh {final_tenant[0]} owner@example.com password123 'Owner Name'")
        
    except Exception as e:
        print(f"❌ Error during migration: {e}")
        conn.rollback()
        raise
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    print("="*50)
    print("Tenant Migration: Integer ID → UUID")
    print("="*50)
    print()
    migrate_tenant_to_uuid()

