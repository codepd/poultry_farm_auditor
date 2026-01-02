#!/usr/bin/env python3
"""Verify user credentials and test login endpoint"""

import psycopg2
import os
import bcrypt
import requests
import json

# Database connection
conn = psycopg2.connect(
    host=os.getenv('DB_HOST', 'localhost'),
    port=os.getenv('DB_PORT', '5432'),
    database=os.getenv('DB_NAME', 'poultry_farm'),
    user=os.getenv('DB_USER', 'postgres'),
    password=os.getenv('DB_PASSWORD', 'postgres')
)
cur = conn.cursor()

print('=' * 60)
print('VERIFYING USER CREDENTIALS')
print('=' * 60)
print()

email = 'ppradeep0610@gmail.com'
password = 'P@sswd123!'

# Check user exists
cur.execute("SELECT id, email, password_hash, full_name, is_active FROM users WHERE email = %s", (email,))
user = cur.fetchone()

if not user:
    print('❌ USER NOT FOUND!')
    print(f'   Email: {email}')
    print()
    print('Available users:')
    cur.execute("SELECT email, full_name FROM users LIMIT 5")
    for u in cur.fetchall():
        print(f'   - {u[0]} ({u[1]})')
else:
    user_id, user_email, password_hash, full_name, is_active = user
    print('✅ USER FOUND:')
    print(f'   ID: {user_id}')
    print(f'   Email: {user_email}')
    print(f'   Full Name: {full_name}')
    print(f'   Is Active: {is_active}')
    print()
    
    # Check password hash
    if password_hash:
        print('✅ Password hash exists')
        print(f'   Hash (first 50 chars): {password_hash[:50]}...')
        
        # Verify password
        try:
            if bcrypt.checkpw(password.encode('utf-8'), password_hash.encode('utf-8')):
                print('✅ PASSWORD VERIFICATION: SUCCESS')
            else:
                print('❌ PASSWORD VERIFICATION: FAILED')
                print('   The password does not match the hash in database')
        except Exception as e:
            print(f'⚠️  Password verification error: {e}')
    else:
        print('❌ No password hash found!')
    
    print()
    
    # Check tenant relationship
    cur.execute("""
        SELECT tu.tenant_id, t.name, tu.role, tu.is_owner 
        FROM tenant_users tu
        JOIN tenants t ON t.id = tu.tenant_id
        WHERE tu.user_id = %s
    """, (user_id,))
    tenant_user = cur.fetchone()
    
    if tenant_user:
        tenant_id, tenant_name, role, is_owner = tenant_user
        print('✅ TENANT RELATIONSHIP:')
        print(f'   Tenant ID: {tenant_id}')
        print(f'   Tenant Name: {tenant_name}')
        print(f'   Role: {role}')
        print(f'   Is Owner: {is_owner}')
    else:
        print('❌ No tenant relationship found!')
        print('   User cannot login without tenant access')

print()
print('=' * 60)
print('TESTING LOGIN ENDPOINT')
print('=' * 60)

# Test login endpoint
try:
    response = requests.post(
        'http://localhost:8080/api/auth/login',
        json={'email': email, 'password': password},
        headers={'Content-Type': 'application/json'},
        timeout=5
    )
    print(f'Status Code: {response.status_code}')
    
    if response.status_code == 200:
        data = response.json()
        if data.get('success'):
            print('✅ LOGIN ENDPOINT: SUCCESS')
            login_data = data.get('data', {})
            print(f'   Token received: {len(login_data.get("token", ""))} characters')
            print(f'   User ID: {login_data.get("user_id")}')
            print(f'   Email: {login_data.get("email")}')
            print(f'   Full Name: {login_data.get("full_name")}')
            print(f'   Tenants: {len(login_data.get("tenants", []))} tenant(s)')
        else:
            print('❌ LOGIN ENDPOINT: Failed')
            print(f'   Error: {data.get("error", "Unknown error")}')
            print(f'   Full response: {json.dumps(data, indent=2)}')
    else:
        print(f'❌ LOGIN ENDPOINT: HTTP {response.status_code}')
        try:
            error_data = response.json()
            print(f'   Error: {error_data.get("error", "Unknown error")}')
        except:
            print(f'   Response: {response.text[:200]}')
except requests.exceptions.ConnectionError:
    print('❌ LOGIN ENDPOINT: Connection error')
    print('   Make sure the API is running on http://localhost:8080')
    print('   Check: curl http://localhost:8080/health')
except Exception as e:
    print(f'❌ LOGIN ENDPOINT: Error - {e}')

cur.close()
conn.close()

