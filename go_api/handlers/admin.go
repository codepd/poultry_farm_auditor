package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"poultry-farm-api/database"
	"poultry-farm-api/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateOwnerDirectly creates an owner user directly (for local development)
// This bypasses the invitation flow
type CreateOwnerRequest struct {
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	FullName  string    `json:"full_name"`
	TenantID  uuid.UUID `json:"tenant_id"`
}

// CreateOwnerDirectly creates an owner directly (for local development)
func CreateOwnerDirectly(w http.ResponseWriter, r *http.Request) {
	// For local development, we can skip auth or use a special admin key
	// In production, this should be protected
	adminKey := r.Header.Get("X-Admin-Key")
	if adminKey != "local-dev-admin-key" {
		respondWithError(w, http.StatusUnauthorized, "Admin key required")
		return
	}

	var req CreateOwnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Check if user exists
	var userID int
	err = database.DB.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&userID)

	if err == sql.ErrNoRows {
		// Create new user
		err = database.DB.QueryRow(`
			INSERT INTO users (email, password_hash, full_name, is_active)
			VALUES ($1, $2, $3, TRUE)
			RETURNING id
		`, req.Email, passwordHash, req.FullName).Scan(&userID)

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	} else {
		// Update existing user
		_, err = database.DB.Exec(`
			UPDATE users
			SET password_hash = $1, full_name = $2, is_active = TRUE, updated_at = CURRENT_TIMESTAMP
			WHERE id = $3
		`, passwordHash, req.FullName, userID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to update user")
			return
		}
	}

	// Add user to tenant as OWNER
	_, err = database.DB.Exec(`
		INSERT INTO tenant_users (tenant_id, user_id, role, is_owner)
		VALUES ($1, $2, 'OWNER', TRUE)
		ON CONFLICT (tenant_id, user_id) DO UPDATE
		SET role = 'OWNER', is_owner = TRUE, updated_at = CURRENT_TIMESTAMP
	`, req.TenantID, userID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to add user to tenant")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Owner created successfully",
		"data": map[string]interface{}{
			"user_id":   userID,
			"email":     req.Email,
			"full_name": req.FullName,
			"tenant_id": req.TenantID,
		},
	})
}

// CreateTenantRequest for admin endpoint
type CreateTenantRequest struct {
	Name         string  `json:"name"`
	Location     *string `json:"location,omitempty"`
	CountryCode  string  `json:"country_code"`
	Currency     string  `json:"currency"`
	NumberFormat string  `json:"number_format"`
	DateFormat   string  `json:"date_format"`
	Capacity     *int    `json:"capacity,omitempty"`
	ParentID     *string `json:"parent_id,omitempty"`
}

// CreateTenantDirectly creates a tenant directly (for local development)
func CreateTenantDirectly(w http.ResponseWriter, r *http.Request) {
	// For local development, we can skip auth or use a special admin key
	adminKey := r.Header.Get("X-Admin-Key")
	if adminKey != "local-dev-admin-key" {
		respondWithError(w, http.StatusUnauthorized, "Admin key required")
		return
	}

	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set defaults
	if req.CountryCode == "" {
		req.CountryCode = "IND"
	}
	if req.Currency == "" {
		req.Currency = "INR"
	}
	if req.NumberFormat == "" {
		req.NumberFormat = "lakhs"
	}
	if req.DateFormat == "" {
		req.DateFormat = "DD-MM-YYYY"
	}

	var tenantID uuid.UUID
	var parentID *uuid.UUID

	if req.ParentID != nil {
		parsed, err := uuid.Parse(*req.ParentID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid parent_id")
			return
		}
		parentID = &parsed
	}

	err := database.DB.QueryRow(`
		INSERT INTO tenants (parent_id, name, location, country_code, currency, number_format, date_format, capacity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, parentID, req.Name, req.Location, req.CountryCode, req.Currency, req.NumberFormat, req.DateFormat, req.Capacity).Scan(&tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create tenant")
		return
	}

	// Return created tenant
	var tenant models.Tenant
	var capacity sql.NullInt64
	err = database.DB.QueryRow(`
		SELECT id, parent_id, name, location, country_code, currency, number_format, date_format, capacity, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`, tenantID).Scan(
		&tenant.ID, &tenant.ParentID, &tenant.Name, &tenant.Location,
		&tenant.CountryCode, &tenant.Currency, &tenant.NumberFormat,
		&tenant.DateFormat, &capacity, &tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch created tenant")
		return
	}

	if capacity.Valid {
		cap := int(capacity.Int64)
		tenant.Capacity = &cap
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Tenant created successfully",
		"data":    tenant,
	})
}

