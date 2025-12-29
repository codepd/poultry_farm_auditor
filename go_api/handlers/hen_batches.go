package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"poultry-farm-api/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GetHenBatches returns all hen batches for a tenant
func GetHenBatches(w http.ResponseWriter, r *http.Request) {
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

	rows, err := database.DB.Query(`
		SELECT id, tenant_id, batch_name, initial_count, current_count,
		       age_weeks, age_days, date_added, notes, created_at, updated_at
		FROM hen_batches
		WHERE tenant_id = $1
		ORDER BY date_added DESC, batch_name
	`, tenantID)

	if err != nil {
		fmt.Printf("Error fetching hen batches: %v\n", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch hen batches: %v", err))
		return
	}
	defer rows.Close()

	var batches []models.HenBatch
	for rows.Next() {
		var batch models.HenBatch
		var notes sql.NullString

		err := rows.Scan(
			&batch.ID, &batch.TenantID, &batch.BatchName, &batch.InitialCount,
			&batch.CurrentCount, &batch.AgeWeeks, &batch.AgeDays,
			&batch.DateAdded, &notes, &batch.CreatedAt, &batch.UpdatedAt,
		)
		if err != nil {
			fmt.Printf("Error scanning hen batch row: %v\n", err)
			continue
		}

		if notes.Valid {
			batch.Notes = notes
		}

		batches = append(batches, batch)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		fmt.Printf("Error iterating hen batch rows: %v\n", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process hen batches: %v", err))
		return
	}

	// Return empty array if no batches - this is valid (tenant has no batches yet)
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    batches,
		"message": func() string {
			if len(batches) == 0 {
				return "No hen batches found for this tenant"
			}
			return ""
		}(),
	})
}

// GetHenBatch returns a single hen batch
func GetHenBatch(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	batchID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	var batch models.HenBatch
	var notes sql.NullString

	err = database.DB.QueryRow(`
		SELECT id, tenant_id, batch_name, initial_count, current_count,
		       age_weeks, age_days, date_added, notes, created_at, updated_at
		FROM hen_batches
		WHERE id = $1 AND tenant_id = $2
	`, batchID, tenantID).Scan(
		&batch.ID, &batch.TenantID, &batch.BatchName, &batch.InitialCount,
		&batch.CurrentCount, &batch.AgeWeeks, &batch.AgeDays,
		&batch.DateAdded, &notes, &batch.CreatedAt, &batch.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Hen batch not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if notes.Valid {
		batch.Notes = notes
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    batch,
	})
}

type HenBatchCreateRequest struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	BatchName    string    `json:"batch_name"`
	InitialCount int       `json:"initial_count"`
	AgeWeeks     int       `json:"age_weeks"`
	AgeDays      int       `json:"age_days"`
	DateAdded    string    `json:"date_added"` // YYYY-MM-DD format
	Notes        *string   `json:"notes,omitempty"`
}

// CreateHenBatch creates a new hen batch
func CreateHenBatch(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req HenBatchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	if req.TenantID == uuid.Nil {
		req.TenantID = tenantID
	}

	// Verify access
	var hasAccess bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM tenant_users 
			WHERE user_id = $1 AND tenant_id = $2
		)
	`, user.UserID, req.TenantID).Scan(&hasAccess)

	if err != nil || !hasAccess {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Parse date
	dateAdded, err := time.Parse("2006-01-02", req.DateAdded)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
		return
	}

	// Insert batch
	var batchID int
	err = database.DB.QueryRow(`
		INSERT INTO hen_batches (
			tenant_id, batch_name, initial_count, current_count,
			age_weeks, age_days, date_added, notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, req.TenantID, req.BatchName, req.InitialCount, req.InitialCount,
		req.AgeWeeks, req.AgeDays, dateAdded, req.Notes).Scan(&batchID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create hen batch")
		return
	}

	// Return created batch
	vars := mux.Vars(r)
	vars["id"] = strconv.Itoa(batchID)
	GetHenBatch(w, r)
}

// UpdateHenBatch updates hen batch age
func UpdateHenBatch(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	batchID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	var req struct {
		AgeWeeks *int    `json:"age_weeks,omitempty"`
		AgeDays  *int    `json:"age_days,omitempty"`
		Notes    *string `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build update query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.AgeWeeks != nil {
		updates = append(updates, fmt.Sprintf("age_weeks = $%d", argIndex))
		args = append(args, *req.AgeWeeks)
		argIndex++
	}
	if req.AgeDays != nil {
		updates = append(updates, fmt.Sprintf("age_days = $%d", argIndex))
		args = append(args, *req.AgeDays)
		argIndex++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argIndex))
		args = append(args, *req.Notes)
		argIndex++
	}

	if len(updates) == 0 {
		respondWithError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, batchID, tenantID)

	query := fmt.Sprintf("UPDATE hen_batches SET %s WHERE id = $%d AND tenant_id = $%d",
		strings.Join(updates, ", "), argIndex, argIndex+1)

	_, err = database.DB.Exec(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update hen batch")
		return
	}

	GetHenBatch(w, r)
}

type MortalityCreateRequest struct {
	BatchID       int     `json:"batch_id"`
	MortalityDate string  `json:"mortality_date"` // YYYY-MM-DD
	Count         int     `json:"count"`
	Reason        *string `json:"reason,omitempty"`
	Notes         *string `json:"notes,omitempty"`
}

// CreateMortality records hen mortality
func CreateMortality(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req MortalityCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	// Verify batch belongs to tenant
	var batchTenantID uuid.UUID
	err = database.DB.QueryRow(`
		SELECT tenant_id FROM hen_batches WHERE id = $1
	`, req.BatchID).Scan(&batchTenantID)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Hen batch not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if batchTenantID != tenantID {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Parse date
	mortalityDate, err := time.Parse("2006-01-02", req.MortalityDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
		return
	}

	// Insert mortality (trigger will update batch count automatically)
	var mortalityID int
	err = database.DB.QueryRow(`
		INSERT INTO hen_mortality (
			batch_id, mortality_date, count, reason, notes, recorded_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, req.BatchID, mortalityDate, req.Count, req.Reason, req.Notes, user.UserID).Scan(&mortalityID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to record mortality")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Mortality recorded",
		"data": map[string]interface{}{
			"id":            mortalityID,
			"batch_id":      req.BatchID,
			"mortality_date": req.MortalityDate,
			"count":         req.Count,
		},
	})
}

