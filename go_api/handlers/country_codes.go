package handlers

import (
	"encoding/json"
	"net/http"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CountryCode struct {
	ID          int    `json:"id"`
	TenantID    string `json:"tenant_id"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
}

type SetCountryCodesRequest struct {
	CountryCodes []CountryCodeEntry `json:"country_codes"`
}

type CountryCodeEntry struct {
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
}

// GetTenantCountryCodes returns allowed country codes for a tenant.
// Public endpoint — used by login page to show country code picker.
func GetTenantCountryCodes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantIDStr := vars["id"]

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, tenant_id, country_code, country_name
		FROM tenant_country_codes
		WHERE tenant_id = $1
		ORDER BY country_code
	`, tenantID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	codes := []CountryCode{}
	for rows.Next() {
		var cc CountryCode
		if err := rows.Scan(&cc.ID, &cc.TenantID, &cc.CountryCode, &cc.CountryName); err != nil {
			continue
		}
		codes = append(codes, cc)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    codes,
	})
}

// SetTenantCountryCodes replaces all country codes for a tenant (admin only).
func SetTenantCountryCodes(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	tenantIDStr := vars["id"]

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	if user.Role != "ADMIN" && user.Role != "OWNER" {
		respondWithError(w, http.StatusForbidden, "Only admins and owners can manage country codes")
		return
	}

	var req SetCountryCodesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.CountryCodes) == 0 {
		respondWithError(w, http.StatusBadRequest, "At least one country code is required")
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer tx.Rollback()

	// Clear existing
	_, err = tx.Exec("DELETE FROM tenant_country_codes WHERE tenant_id = $1", tenantID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update country codes")
		return
	}

	// Insert new
	for _, cc := range req.CountryCodes {
		_, err = tx.Exec(`
			INSERT INTO tenant_country_codes (tenant_id, country_code, country_name)
			VALUES ($1, $2, $3)
		`, tenantID, cc.CountryCode, cc.CountryName)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to insert country code")
			return
		}
	}

	if err = tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to commit changes")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Country codes updated",
	})
}

// GetAllCountryCodes returns a distinct list of all country codes across tenants (public, for login page).
func GetAllCountryCodes(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
		SELECT DISTINCT country_code, country_name
		FROM tenant_country_codes
		ORDER BY country_code
	`)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	type SimpleCode struct {
		CountryCode string `json:"country_code"`
		CountryName string `json:"country_name"`
	}
	codes := []SimpleCode{}
	for rows.Next() {
		var cc SimpleCode
		if err := rows.Scan(&cc.CountryCode, &cc.CountryName); err != nil {
			continue
		}
		codes = append(codes, cc)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    codes,
	})
}
