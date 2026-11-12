package database

import (
	"database/sql"
	"fmt"
	"log"
	"poultry-farm-api/config"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init(cfg *config.Config) error {
	var err error
	DB, err = sql.Open("postgres", cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	runMigrations()

	return nil
}

func runMigrations() {
	migrations := []string{
		"ALTER TABLE users ALTER COLUMN email DROP NOT NULL",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20) UNIQUE",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS preferred_language VARCHAR(10) DEFAULT 'en'",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS must_set_password BOOLEAN DEFAULT FALSE",
		`CREATE TABLE IF NOT EXISTS invitations (
			id SERIAL PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			invited_by_user_id INTEGER NOT NULL REFERENCES users(id),
			email VARCHAR(255),
			phone VARCHAR(20),
			role user_role_enum NOT NULL,
			token VARCHAR(255) UNIQUE NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			accepted_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS otp_codes (
			id SERIAL PRIMARY KEY,
			phone VARCHAR(20) NOT NULL,
			code VARCHAR(255) NOT NULL,
			tenant_id UUID NOT NULL,
			verified BOOLEAN DEFAULT FALSE,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		"ALTER TABLE otp_codes ALTER COLUMN code TYPE VARCHAR(255)",
		"ALTER TABLE otp_codes ADD COLUMN IF NOT EXISTS encrypted_code TEXT",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS parent_id UUID",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS location VARCHAR(255)",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS country_code VARCHAR(10)",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS currency VARCHAR(10)",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS number_format VARCHAR(50)",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS date_format VARCHAR(50)",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS capacity INTEGER",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS refresh_ttl_without_remember_hours INTEGER DEFAULT 12",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS refresh_ttl_with_remember_days INTEGER DEFAULT 30",
		`CREATE TABLE IF NOT EXISTS auth_sessions (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			refresh_token_hash VARCHAR(64) UNIQUE NOT NULL,
			remember_me BOOLEAN NOT NULL DEFAULT FALSE,
			user_agent TEXT,
			ip_address TEXT,
			device_id VARCHAR(128),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			revoked_at TIMESTAMP,
			revoked_reason TEXT
		)`,
		"CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_auth_sessions_tenant_id ON auth_sessions(tenant_id)",
		"CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at ON auth_sessions(expires_at)",
		"ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS device_id VARCHAR(128)",
		`CREATE TABLE IF NOT EXISTS document_uploads (
			id SERIAL PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			s3_bucket VARCHAR(255) NOT NULL,
			s3_key TEXT NOT NULL,
			s3_version_id VARCHAR(255),
			file_name VARCHAR(255) NOT NULL,
			file_type VARCHAR(50) NOT NULL,
			content_type VARCHAR(100),
			file_size BIGINT,
			sha256_hash VARCHAR(64),
			document_year INTEGER,
			document_month INTEGER,
			upload_source VARCHAR(50) DEFAULT 'WEB_PORTAL',
			upload_status VARCHAR(50) DEFAULT 'UPLOADED',
			parse_result JSONB,
			error_message TEXT,
			uploaded_by_user_id INTEGER REFERENCES users(id),
			ledger_parse_id INTEGER REFERENCES ledger_parses(id) ON DELETE SET NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, s3_key)
		)`,
		"CREATE INDEX IF NOT EXISTS idx_document_uploads_tenant_id ON document_uploads(tenant_id)",
		"CREATE INDEX IF NOT EXISTS idx_document_uploads_status ON document_uploads(tenant_id, upload_status)",
		"CREATE INDEX IF NOT EXISTS idx_document_uploads_period ON document_uploads(tenant_id, document_year, document_month)",
	}
	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			log.Printf("Migration (may already be applied): %s — %v", m, err)
		} else {
			log.Printf("Migration applied: %s", m)
		}
	}
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
