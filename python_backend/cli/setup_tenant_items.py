#!/usr/bin/env python3
"""
Script to populate default tenant items for existing tenants.
This ensures all tenants have the standard egg and feed items configured.
"""

import sys
import os

# Add parent directory to path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from database.connection import get_db_connection

def setup_tenant_items():
    """Populate default items for all existing tenants."""
    conn = get_db_connection()
    if not conn:
        print("ERROR: Failed to connect to database")
        return False
    
    cur = conn.cursor()
    
    try:
        # First, ensure the table exists
        cur.execute("""
            CREATE TABLE IF NOT EXISTS tenant_items (
                id SERIAL PRIMARY KEY,
                tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
                category category_enum NOT NULL,
                item_name VARCHAR(255) NOT NULL,
                display_order INTEGER DEFAULT 0,
                is_active BOOLEAN DEFAULT TRUE,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                UNIQUE(tenant_id, category, item_name)
            )
        """)
        
        # Create indexes
        cur.execute("""
            CREATE INDEX IF NOT EXISTS idx_tenant_items_tenant_category 
            ON tenant_items(tenant_id, category)
        """)
        
        cur.execute("""
            CREATE INDEX IF NOT EXISTS idx_tenant_items_active 
            ON tenant_items(tenant_id, category, is_active) 
            WHERE is_active = TRUE
        """)
        
        conn.commit()
        print("✓ Created tenant_items table (if it didn't exist)")
        
    except Exception as e:
        conn.rollback()
        print(f"  Warning: Failed to create table: {e}")
        # Continue anyway - table might already exist
    
    try:
        # Default egg items
        egg_items = [
            ('LARGE EGG', 1),
            ('MEDIUM EGG', 2),
            ('SMALL EGG', 3),
            ('BROKEN EGG', 4),
            ('DOUBLE YOLK', 5),
            ('DIRT EGG', 6),
            ('CORRECT EGG', 7),
            ('EXPORT EGG', 8),
        ]
        
        # Default feed items
        feed_items = [
            ('LAYER MASH', 1),
            ('GROWER MASH', 2),
            ('PRE LAYER MASH', 3),
            ('LAYER MASH BULK', 4),
            ('GROWER MASH BULK', 5),
            ('PRE LAYER MASH BULK', 6),
        ]
        
        # Get all tenant IDs
        cur.execute("SELECT id FROM tenants")
        tenant_ids = cur.fetchall()
        
        if not tenant_ids:
            print("No tenants found in database")
            return False
        
        print(f"Found {len(tenant_ids)} tenant(s). Setting up items...")
        
        inserted_count = 0
        skipped_count = 0
        
        for (tenant_id,) in tenant_ids:
            # Insert egg items
            for item_name, display_order in egg_items:
                try:
                    cur.execute("""
                        INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
                        VALUES (%s, 'EGG', %s, %s, TRUE)
                        ON CONFLICT (tenant_id, category, item_name) DO NOTHING
                    """, (str(tenant_id), item_name, display_order))
                    if cur.rowcount > 0:
                        inserted_count += 1
                    else:
                        skipped_count += 1
                except Exception as e:
                    print(f"  Warning: Failed to insert {item_name} for tenant {tenant_id}: {e}")
            
            # Insert feed items
            for item_name, display_order in feed_items:
                try:
                    cur.execute("""
                        INSERT INTO tenant_items (tenant_id, category, item_name, display_order, is_active)
                        VALUES (%s, 'FEED', %s, %s, TRUE)
                        ON CONFLICT (tenant_id, category, item_name) DO NOTHING
                    """, (str(tenant_id), item_name, display_order))
                    if cur.rowcount > 0:
                        inserted_count += 1
                    else:
                        skipped_count += 1
                except Exception as e:
                    print(f"  Warning: Failed to insert {item_name} for tenant {tenant_id}: {e}")
        
        conn.commit()
        print(f"\n✓ Setup complete!")
        print(f"  - Inserted: {inserted_count} items")
        print(f"  - Skipped (already exist): {skipped_count} items")
        return True
        
    except Exception as e:
        conn.rollback()
        print(f"ERROR: {e}")
        return False
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    print("Setting up default tenant items...")
    success = setup_tenant_items()
    sys.exit(0 if success else 1)

