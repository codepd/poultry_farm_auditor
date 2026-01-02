"""
Database connection management for PostgreSQL.
"""

import os
import psycopg2
from psycopg2 import pool
from contextlib import contextmanager
from database.schema import get_all_schema_sql

# Connection pool
_connection_pool = None

def get_db_config():
    """Get database configuration from environment variables."""
    return {
        'host': os.getenv('DB_HOST', 'localhost'),
        'port': os.getenv('DB_PORT', '5432'),
        'database': os.getenv('DB_NAME', 'poultry_farm'),
        'user': os.getenv('DB_USER', 'postgres'),
        'password': os.getenv('DB_PASSWORD', 'postgres'),
    }

def init_connection_pool(minconn=1, maxconn=10):
    """Initialize the connection pool."""
    global _connection_pool
    if _connection_pool is None:
        config = get_db_config()
        _connection_pool = psycopg2.pool.SimpleConnectionPool(
            minconn, maxconn,
            **config
        )
    return _connection_pool

def get_db_connection():
    """Get a database connection from the pool."""
    pool = init_connection_pool()
    return pool.getconn()

def return_db_connection(conn):
    """Return a connection to the pool."""
    pool = init_connection_pool()
    pool.putconn(conn)

@contextmanager
def get_db_cursor():
    """Context manager for database cursor."""
    conn = get_db_connection()
    try:
        cursor = conn.cursor()
        yield cursor
        conn.commit()
    except Exception as e:
        conn.rollback()
        raise e
    finally:
        cursor.close()
        return_db_connection(conn)

def init_database():
    """Initialize the database schema."""
    with get_db_cursor() as cursor:
        for sql in get_all_schema_sql():
            cursor.execute(sql)
        print("Database schema initialized successfully.")

def close_pool():
    """Close all connections in the pool."""
    global _connection_pool
    if _connection_pool:
        _connection_pool.closeall()
        _connection_pool = None

