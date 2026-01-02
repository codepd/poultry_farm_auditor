#!/usr/bin/env python3
"""
Initialize the database schema.
Run this script to create all tables, enums, indexes, and triggers.
"""

import sys
import os

# Add parent directory to path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from database import init_database

if __name__ == '__main__':
    try:
        print("Initializing database schema...")
        init_database()
        print("✅ Database initialized successfully!")
    except Exception as e:
        print(f"❌ Error initializing database: {e}")
        sys.exit(1)

