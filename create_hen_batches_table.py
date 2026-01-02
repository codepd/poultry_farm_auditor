#!/usr/bin/env python3
"""
Create hen_batches and hen_mortality tables in the database.
"""

import os
import sys

# Add python_backend to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'python_backend'))

from database.connection import get_db_connection, return_db_connection

def create_tables():
    """Create hen_batches and hen_mortality tables."""
    conn = None
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        
        # Read SQL file
        sql_file = os.path.join(os.path.dirname(__file__), 'create_hen_batches_table.sql')
        with open(sql_file, 'r') as f:
            sql_script = f.read()
        
        # Execute SQL script
        cur.execute(sql_script)
        conn.commit()
        
        print("✓ Successfully created hen_batches and hen_mortality tables")
        
        # Verify tables exist
        cur.execute("""
            SELECT table_name 
            FROM information_schema.tables 
            WHERE table_schema = 'public' 
            AND table_name IN ('hen_batches', 'hen_mortality')
        """)
        tables = cur.fetchall()
        print(f"✓ Verified tables exist: {[t[0] for t in tables]}")
        
        cur.close()
        
    except Exception as e:
        print(f"✗ Error creating tables: {e}")
        import traceback
        traceback.print_exc()
        if conn:
            conn.rollback()
        sys.exit(1)
    finally:
        if conn:
            return_db_connection(conn)

if __name__ == '__main__':
    create_tables()

