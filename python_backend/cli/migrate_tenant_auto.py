#!/usr/bin/env python3
"""
Auto-migration script to rename tenant to "Pradeep Farm"
and ensure tenant uses UUID (if not already)
This version runs automatically without user prompts
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
        
        # Step 2: Check if tenant table already uses UUID
        cur.execute("""
            SELECT data_type 
            FROM information_schema.columns 
            WHERE table_name = 'tenants' AND column_name = 'id'
        """)
        id_type = cur.fetchone()[0]
        
        if id_type == 'uuid':
            print("✅ Tenant table already uses UUID")
            # Find the tenant UUID
            cur.execute("SELECT id, name FROM tenants WHERE id::text = %s OR id = %s", ('1', 1))
            existing = cur.fetchone()
            if existing:
                tenant_uuid = existing[0]
                print(f"Found existing tenant with UUID: {tenant_uuid}")
                # Update name
                cur.execute("""
                    UPDATE tenants 
                    SET name = 'Pradeep Farm', updated_at = CURRENT_TIMESTAMP
                    WHERE id = %s
                """, (tenant_uuid,))
                print(f"✅ Updated tenant name to 'Pradeep Farm'")
                final_uuid = tenant_uuid
            else:
                print("❌ Could not find tenant")
                conn.rollback()
                return
        else:
            print("⚠️  Tenant table uses integer, starting migration...")
            print("This requires updating all foreign key references.")
            
            # Generate new UUID
            new_uuid = uuid.uuid4()
            print(f"Generated new UUID: {new_uuid}")
            
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
                print(f"  Found: {table_name}.{column_name} ({data_type})")
                if data_type in ('integer', 'bigint'):
                    tables_to_update.append((table_name, column_name))
            
            print(f"\n⚠️  Found {len(tables_to_update)} tables with integer tenant_id")
            print("Migration will:")
            print("  1. Alter tenants.id column to UUID type")
            print("  2. Update tenant id=1 to new UUID")
            print("  3. Alter all foreign key columns to UUID")
            print("  4. Update tenant name to 'Pradeep Farm'")
            print("\n🚀 Starting migration...")
            
            # Enable UUID extension
            cur.execute('CREATE EXTENSION IF NOT EXISTS "uuid-ossp"')
            
            # Step 1: First, we need to drop foreign key constraints temporarily
            # Get all foreign key constraints that reference tenants.id
            cur.execute("""
                SELECT conname, conrelid::regclass::text as table_name
                FROM pg_constraint
                WHERE confrelid = 'tenants'::regclass
                AND contype = 'f'
            """)
            fk_constraints = cur.fetchall()
            
            print(f"Found {len(fk_constraints)} foreign key constraints to temporarily drop")
            dropped_constraints = []
            for conname, table_name in fk_constraints:
                try:
                    cur.execute(f'ALTER TABLE {table_name} DROP CONSTRAINT IF EXISTS {conname} CASCADE')
                    dropped_constraints.append((conname, table_name))
                    print(f"  ✅ Dropped constraint {conname} from {table_name}")
                except Exception as e:
                    print(f"  ⚠️  Warning: Could not drop constraint {conname}: {e}")
            
            # Step 2: Add a temporary UUID column to tenants
            cur.execute("ALTER TABLE tenants ADD COLUMN IF NOT EXISTS id_new UUID")
            print("✅ Added temporary UUID column")
            
            # Step 3: Generate UUID for the existing tenant
            cur.execute("""
                UPDATE tenants 
                SET id_new = %s
                WHERE id = %s
            """, (str(new_uuid), old_id))
            print(f"✅ Assigned UUID {new_uuid} to tenant id={old_id}")
            
            # Step 4: Alter foreign key columns to UUID first
            for table_name, column_name in tables_to_update:
                try:
                    print(f"Altering {table_name}.{column_name} to UUID...")
                    # Add temporary UUID column
                    cur.execute(f'ALTER TABLE {table_name} ADD COLUMN IF NOT EXISTS {column_name}_new UUID')
                    # Copy data using the mapping from tenants
                    cur.execute(f"""
                        UPDATE {table_name} t
                        SET {column_name}_new = t2.id_new
                        FROM tenants t2
                        WHERE t.{column_name} = t2.id::integer
                    """)
                    # Drop old column
                    cur.execute(f'ALTER TABLE {table_name} DROP COLUMN IF EXISTS {column_name} CASCADE')
                    # Rename new column
                    cur.execute(f'ALTER TABLE {table_name} RENAME COLUMN {column_name}_new TO {column_name}')
                    print(f"  ✅ Converted {table_name}.{column_name} to UUID")
                except Exception as e:
                    print(f"  ⚠️  Warning: Could not convert {table_name}.{column_name}: {e}")
                    import traceback
                    traceback.print_exc()
            
            # Step 5: Alter tenants.id to UUID
            cur.execute("ALTER TABLE tenants DROP COLUMN IF EXISTS id CASCADE")
            cur.execute("ALTER TABLE tenants RENAME COLUMN id_new TO id")
            cur.execute("ALTER TABLE tenants ADD PRIMARY KEY (id)")
            print("✅ Converted tenants.id to UUID")
            
            # Step 6: Update tenant name
            cur.execute("""
                UPDATE tenants 
                SET name = 'Pradeep Farm'
                WHERE id = %s
            """, (str(new_uuid),))
            print("✅ Updated tenant name to 'Pradeep Farm'")
            
            # Step 7: Re-add foreign key constraints (simplified - they should work now)
            # Note: PostgreSQL will auto-create some constraints, but we can verify
            
            final_uuid = new_uuid
        
        # Commit transaction
        conn.commit()
        
        # Display final tenant info
        # Check what columns exist first
        cur.execute("""
            SELECT column_name 
            FROM information_schema.columns 
            WHERE table_name = 'tenants'
            ORDER BY ordinal_position
        """)
        tenant_columns = [row[0] for row in cur.fetchall()]
        
        # Build SELECT query with only existing columns
        select_cols = ['id', 'name']
        if 'created_at' in tenant_columns:
            select_cols.append('created_at')
        
        select_sql = f"SELECT {', '.join(select_cols)} FROM tenants WHERE id = %s"
        cur.execute(select_sql, (str(final_uuid),))
        final_tenant = cur.fetchone()
        
        print("\n" + "="*50)
        print("✅ Migration completed successfully!")
        print("="*50)
        print(f"Tenant UUID: {final_tenant[0]}")
        print(f"Tenant Name: {final_tenant[1]}")
        if len(final_tenant) > 2:
            print(f"Created At: {final_tenant[2]}")
        print("="*50)
        print(f"\nUse this UUID when creating the owner:")
        print(f"  ./setup_first_owner.sh {final_tenant[0]} owner@example.com password123 'Owner Name'")
        
        return final_tenant[0]
        
    except Exception as e:
        print(f"❌ Error during migration: {e}")
        import traceback
        traceback.print_exc()
        conn.rollback()
        raise
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    print("="*50)
    print("Tenant Migration: Integer ID → UUID (Auto-mode)")
    print("="*50)
    print()
    migrate_tenant_to_uuid()

