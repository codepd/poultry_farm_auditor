package database

import (
	"database/sql"
	"fmt"
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

	return nil
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
