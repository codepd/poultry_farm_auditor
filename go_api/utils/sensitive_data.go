package utils

import (
	"database/sql"
	"poultry-farm-api/models"

	"github.com/google/uuid"
)

// CheckSensitiveDataConfig checks if a data type is sensitive for a given tenant.
// Checks tenant config, then walks up the hierarchy to parent tenants.
// Returns (isSensitive, found)
func CheckSensitiveDataConfig(db *sql.DB, tenantID uuid.UUID, dataType string) (bool, bool, error) {
	var config models.SensitiveDataConfig

	// Check tenant config first
	err := db.QueryRow(`
		SELECT id, tenant_id, data_type, is_sensitive
		FROM sensitive_data_config
		WHERE tenant_id = $1 AND data_type = $2
	`, tenantID, dataType).Scan(
		&config.ID, &config.TenantID, &config.DataType, &config.IsSensitive,
	)

	if err == nil {
		return config.IsSensitive, true, nil
	} else if err != sql.ErrNoRows {
		return false, false, err
	}

	// If not found, check parent tenant (walk up hierarchy)
	var parentID *uuid.UUID
	err = db.QueryRow(`
		SELECT parent_id FROM tenants WHERE id = $1
	`, tenantID).Scan(&parentID)

	if err == nil && parentID != nil {
		// Recursively check parent
		return CheckSensitiveDataConfig(db, *parentID, dataType)
	}

	// No config found in hierarchy, default to sensitive for safety
	return true, false, nil
}

// IsDataSensitive checks if data should be hidden based on sensitive config and user permissions.
func IsDataSensitive(db *sql.DB, tenantID uuid.UUID, dataType string, canViewSensitive bool) (bool, error) {
	if canViewSensitive {
		return false, nil // User can view sensitive data
	}

	isSensitive, _, err := CheckSensitiveDataConfig(db, tenantID, dataType)
	return isSensitive, err
}
