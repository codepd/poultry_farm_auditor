package handlers

import (
	"encoding/json"
	"net/http"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"poultry-farm-api/models"

	"github.com/google/uuid"
)

// GetSensitiveDataConfig returns sensitive data configuration for a tenant
func GetSensitiveDataConfig(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanManageUsers {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, tenant_id, data_type, is_sensitive, created_at, updated_at
		FROM sensitive_data_config
		WHERE tenant_id = $1
		ORDER BY data_type
	`, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch sensitive data config")
		return
	}
	defer rows.Close()

	var configs []models.SensitiveDataConfig
	for rows.Next() {
		var config models.SensitiveDataConfig
		err := rows.Scan(
			&config.ID, &config.TenantID, &config.DataType,
			&config.IsSensitive, &config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			continue
		}
		configs = append(configs, config)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    configs,
	})
}

type SensitiveDataConfigUpdateRequest struct {
	DataType    string `json:"data_type"`
	IsSensitive bool   `json:"is_sensitive"`
}

// UpdateSensitiveDataConfig updates sensitive data configuration
func UpdateSensitiveDataConfig(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanManageUsers {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	var req SensitiveDataConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Upsert configuration
	_, err = database.DB.Exec(`
		INSERT INTO sensitive_data_config (tenant_id, data_type, is_sensitive)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, data_type)
		DO UPDATE SET is_sensitive = $3, updated_at = CURRENT_TIMESTAMP
	`, tenantID, req.DataType, req.IsSensitive)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update sensitive data config")
		return
	}

	// Return updated config
	var config models.SensitiveDataConfig
	err = database.DB.QueryRow(`
		SELECT id, tenant_id, data_type, is_sensitive, created_at, updated_at
		FROM sensitive_data_config
		WHERE tenant_id = $1 AND data_type = $2
	`, tenantID, req.DataType).Scan(
		&config.ID, &config.TenantID, &config.DataType,
		&config.IsSensitive, &config.CreatedAt, &config.UpdatedAt,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch updated config")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    config,
	})
}

