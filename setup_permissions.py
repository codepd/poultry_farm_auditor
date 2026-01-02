#!/usr/bin/env python3
"""Setup default role permissions for the tenant"""

import psycopg2
import os
import sys

def setup_permissions():
    conn = psycopg2.connect(
        host=os.getenv('DB_HOST', 'localhost'),
        port=os.getenv('DB_PORT', '5432'),
        database=os.getenv('DB_NAME', 'poultry_farm'),
        user=os.getenv('DB_USER', 'postgres'),
        password=os.getenv('DB_PASSWORD', 'postgres')
    )
    cur = conn.cursor()
    
    try:
        print('Setting up role permissions...')
        print('=' * 50)
        
        # Check if table exists
        cur.execute("""
            SELECT COUNT(*) 
            FROM information_schema.tables 
            WHERE table_name = 'role_permissions'
        """)
        table_exists = cur.fetchone()[0] > 0
        
        if not table_exists:
            print('Creating role_permissions table...')
            cur.execute("""
                CREATE TABLE IF NOT EXISTS role_permissions (
                    id SERIAL PRIMARY KEY,
                    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
                    role user_role_enum NOT NULL,
                    can_view_sensitive_data BOOLEAN DEFAULT FALSE,
                    can_edit_transactions BOOLEAN DEFAULT FALSE,
                    can_approve_transactions BOOLEAN DEFAULT FALSE,
                    can_manage_users BOOLEAN DEFAULT FALSE,
                    can_view_charts BOOLEAN DEFAULT FALSE,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    UNIQUE(tenant_id, role)
                );
            """)
            print('✅ Table created')
        else:
            print('✅ Table already exists')
        
        # Get tenant UUID
        cur.execute("SELECT id FROM tenants WHERE name = 'Pradeep Farm' LIMIT 1")
        tenant_row = cur.fetchone()
        
        if not tenant_row:
            print('❌ Tenant "Pradeep Farm" not found!')
            sys.exit(1)
        
        tenant_id = tenant_row[0]
        print(f'Found tenant: {tenant_id}')
        print()
        
        # Define default permissions for each role
        role_permissions = {
            'OWNER': {
                'can_view_sensitive_data': True,
                'can_edit_transactions': True,
                'can_approve_transactions': True,
                'can_manage_users': True,
                'can_view_charts': True,
            },
            'CO_OWNER': {
                'can_view_sensitive_data': True,
                'can_edit_transactions': True,
                'can_approve_transactions': True,
                'can_manage_users': True,
                'can_view_charts': True,
            },
            'MANAGER': {
                'can_view_sensitive_data': True,
                'can_edit_transactions': True,
                'can_approve_transactions': False,  # Managers can edit but not approve
                'can_manage_users': False,  # Managers cannot manage users
                'can_view_charts': True,
            },
            'AUDITOR': {
                'can_view_sensitive_data': True,
                'can_edit_transactions': False,
                'can_approve_transactions': False,
                'can_manage_users': False,
                'can_view_charts': True,
            },
            'OTHER_USER': {
                'can_view_sensitive_data': False,
                'can_edit_transactions': True,
                'can_approve_transactions': False,
                'can_manage_users': False,
                'can_view_charts': False,
            },
            'ADMIN': {
                'can_view_sensitive_data': True,
                'can_edit_transactions': True,
                'can_approve_transactions': True,
                'can_manage_users': True,
                'can_view_charts': True,
            },
        }
        
        # Insert or update permissions for each role
        for role, perms in role_permissions.items():
            cur.execute("""
                INSERT INTO role_permissions (
                    tenant_id, role, can_view_sensitive_data, can_edit_transactions,
                    can_approve_transactions, can_manage_users, can_view_charts
                )
                VALUES (%s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (tenant_id, role) DO UPDATE
                SET can_view_sensitive_data = EXCLUDED.can_view_sensitive_data,
                    can_edit_transactions = EXCLUDED.can_edit_transactions,
                    can_approve_transactions = EXCLUDED.can_approve_transactions,
                    can_manage_users = EXCLUDED.can_manage_users,
                    can_view_charts = EXCLUDED.can_view_charts,
                    updated_at = CURRENT_TIMESTAMP
            """, (
                tenant_id, role,
                perms['can_view_sensitive_data'],
                perms['can_edit_transactions'],
                perms['can_approve_transactions'],
                perms['can_manage_users'],
                perms['can_view_charts']
            ))
            print(f'✅ Set permissions for {role}')
        
        conn.commit()
        
        print()
        print('=' * 50)
        print('✅ Permissions setup complete!')
        print()
        print('Verifying permissions:')
        cur.execute("""
            SELECT role, can_view_charts, can_view_sensitive_data, can_edit_transactions
            FROM role_permissions
            WHERE tenant_id = %s
            ORDER BY role
        """, (tenant_id,))
        
        for row in cur.fetchall():
            role, charts, sensitive, edit = row
            print(f'  {role}: Charts={charts}, Sensitive={sensitive}, Edit={edit}')
        
    except Exception as e:
        conn.rollback()
        print(f'❌ Error: {e}')
        import traceback
        traceback.print_exc()
        sys.exit(1)
    finally:
        cur.close()
        conn.close()

if __name__ == '__main__':
    setup_permissions()



