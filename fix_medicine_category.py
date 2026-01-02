#!/usr/bin/env python3
"""
Fix medicine items that are incorrectly categorized as FEED in the database.
This script will:
1. Check for medicine items in FEED category
2. Update them to MEDICINE category
3. Show summary of changes
"""

import os
import sys

# Add python_backend to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'python_backend'))

try:
    from database.connection import get_db_connection, return_db_connection
except ImportError as e:
    print(f"Error: Could not import database connection module: {e}")
    print("Make sure you're running this from the project root directory")
    sys.exit(1)

# Medicine keywords to identify medicine items
MEDICINE_KEYWORDS = [
    'D3', 'VETMULIN', 'OXYCYCLINE', 'TIAZIN', 'BPPS', 'CTC', 'SHELL GRIT',
    'ROVIMIX', 'CHOLIMARIN', 'ZAGROMIN', 'G PRO NATURO', 'NECROVET', 'TOXOL',
    'FRA C12', 'FRA C 12', 'CALCI', 'CALDLIV', 'RESPAFEED', 'VENTRIM', 'VITAL',
    'MEDICINE', 'MEDIC', 'VITAMIN', 'SUPPLEMENT', 'GRIT', 'VET', 'NECRO', 'TOX'
]

def build_medicine_condition():
    """Build SQL condition to match medicine items"""
    conditions = []
    for keyword in MEDICINE_KEYWORDS:
        conditions.append(f"UPPER(item_name) LIKE '%{keyword}%'")
    return " OR ".join(conditions)

def check_medicine_in_feed(conn):
    """Check for medicine items incorrectly categorized as FEED"""
    cur = conn.cursor()
    
    condition = build_medicine_condition()
    query = f"""
        SELECT 
            id,
            tenant_id,
            transaction_date,
            item_name,
            category,
            transaction_type,
            amount,
            quantity,
            unit
        FROM transactions
        WHERE category = 'FEED'
            AND transaction_type IN ('PURCHASE', 'EXPENSE')
            AND ({condition})
        ORDER BY transaction_date DESC, tenant_id
        LIMIT 100
    """
    
    cur.execute(query)
    results = cur.fetchall()
    cur.close()
    return results

def get_summary_by_month(conn):
    """Get summary of medicine items in FEED category by month"""
    cur = conn.cursor()
    
    condition = build_medicine_condition()
    query = f"""
        SELECT 
            tenant_id,
            EXTRACT(YEAR FROM transaction_date)::int as year,
            EXTRACT(MONTH FROM transaction_date)::int as month,
            COUNT(*) as medicine_count,
            SUM(amount) as total_medicine_amount
        FROM transactions
        WHERE category = 'FEED'
            AND transaction_type IN ('PURCHASE', 'EXPENSE')
            AND ({condition})
        GROUP BY tenant_id, EXTRACT(YEAR FROM transaction_date), EXTRACT(MONTH FROM transaction_date)
        ORDER BY year DESC, month DESC, tenant_id
    """
    
    cur.execute(query)
    results = cur.fetchall()
    cur.close()
    return results

def fix_medicine_category(conn, dry_run=True):
    """Update medicine items from FEED to MEDICINE category"""
    cur = conn.cursor()
    
    condition = build_medicine_condition()
    query = f"""
        UPDATE transactions
        SET category = 'MEDICINE'
        WHERE category = 'FEED'
            AND transaction_type IN ('PURCHASE', 'EXPENSE')
            AND ({condition})
    """
    
    if dry_run:
        # Count how many would be updated
        count_query = f"""
            SELECT COUNT(*) as count, SUM(amount) as total
            FROM transactions
            WHERE category = 'FEED'
                AND transaction_type IN ('PURCHASE', 'EXPENSE')
                AND ({condition})
        """
        cur.execute(count_query)
        result = cur.fetchone()
        cur.close()
        return result
    else:
        cur.execute(query)
        updated_count = cur.rowcount
        
        # Get total amount updated
        cur.execute(f"""
            SELECT SUM(amount) as total
            FROM transactions
            WHERE category = 'MEDICINE'
                AND transaction_type IN ('PURCHASE', 'EXPENSE')
                AND ({condition})
        """)
        total_result = cur.fetchone()
        cur.close()
        return (updated_count, total_result[0] if total_result else 0)

def main():
    print("=" * 60)
    print("Medicine Category Fix Script")
    print("=" * 60)
    
    try:
        conn = get_db_connection()
        print("✓ Connected to database")
        
        # Step 1: Check for medicine items in FEED category
        print("\n1. Checking for medicine items incorrectly categorized as FEED...")
        items = check_medicine_in_feed(conn)
        
        if not items:
            print("   ✓ No medicine items found in FEED category. Database is correct!")
            conn.close()
            return
        
        print(f"   Found {len(items)} medicine items in FEED category")
        
        # Show sample items
        print("\n   Sample items to be fixed:")
        for item in items[:5]:
            print(f"   - {item['item_name']} ({item['transaction_date']}): ₹{item['amount']:.2f}")
        
        # Step 2: Show summary by month
        print("\n2. Summary by month:")
        summary = get_summary_by_month(conn)
        for row in summary:
            print(f"   {row['year']}-{row['month']:02d}: {row['medicine_count']} items, ₹{row['total_medicine_amount']:.2f}")
        
        # Step 3: Dry run to see what would be updated
        print("\n3. Dry run (checking what would be updated)...")
        dry_run_result = fix_medicine_category(conn, dry_run=True)
        print(f"   Would update: {dry_run_result[0]} transactions")
        print(f"   Total amount: ₹{dry_run_result[1]:.2f}" if dry_run_result[1] else "   Total amount: ₹0.00")
        
        # Step 4: Ask for confirmation
        print("\n4. Ready to fix?")
        response = input("   Enter 'yes' to apply the fix, or 'no' to cancel: ").strip().lower()
        
        if response == 'yes':
            print("\n5. Applying fix...")
            conn.rollback()  # Make sure we start fresh
            fix_result = fix_medicine_category(conn, dry_run=False)
            conn.commit()
            print(f"   ✓ Updated {fix_result[0]} transactions")
            print(f"   ✓ Total amount moved: ₹{fix_result[1]:.2f}")
            print("\n   ✓ Fix applied successfully!")
        else:
            print("\n   ✗ Fix cancelled. No changes made.")
            conn.rollback()
        
        return_db_connection(conn)
        
    except Exception as e:
        print(f"\n✗ Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == '__main__':
    main()

