package models

import (
	"time"

	"github.com/google/uuid"
)

type SensitiveDataConfig struct {
	ID          int       `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	DataType    string    `json:"data_type"` // 'EGGS_SOLD', 'FEED_PURCHASED', 'NET_PROFIT', etc.
	IsSensitive bool      `json:"is_sensitive"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
