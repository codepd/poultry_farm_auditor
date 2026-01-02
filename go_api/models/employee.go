package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Employee struct {
	ID          int            `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	FullName    string         `json:"full_name"`
	Phone       sql.NullString `json:"phone,omitempty"`
	Email       sql.NullString `json:"email,omitempty"`
	Address     sql.NullString `json:"address,omitempty"`
	Designation sql.NullString `json:"designation,omitempty"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
