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
			code VARCHAR(6) NOT NULL,
			tenant_id UUID NOT NULL,
			verified BOOLEAN DEFAULT FALSE,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
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
