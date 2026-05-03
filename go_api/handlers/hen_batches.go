package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"poultry-farm-api/models"
	"strconv"
	"strings"
	"time"

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

		batch.Notes = models.NullString{NullString: notes}

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

	batch.Notes = models.NullString{NullString: notes}

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
		BatchName    *string `json:"batch_name,omitempty"`
		CurrentCount *int    `json:"current_count,omitempty"`
		AgeWeeks     *int    `json:"age_weeks,omitempty"`
		AgeDays      *int    `json:"age_days,omitempty"`
		Notes        *string `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build update query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.BatchName != nil {
		updates = append(updates, fmt.Sprintf("batch_name = $%d", argIndex))
		args = append(args, *req.BatchName)
		argIndex++
	}
	if req.CurrentCount != nil {
		updates = append(updates, fmt.Sprintf("current_count = $%d", argIndex))
		args = append(args, *req.CurrentCount)
		argIndex++
	}
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

// DeleteHenBatch deletes a hen batch
func DeleteHenBatch(w http.ResponseWriter, r *http.Request) {
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

	// Verify batch belongs to tenant
	var batchTenantID uuid.UUID
	err = database.DB.QueryRow(`
		SELECT tenant_id FROM hen_batches WHERE id = $1
	`, batchID).Scan(&batchTenantID)

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

	// Delete the batch (cascade will handle related records if configured)
	_, err = database.DB.Exec(`
		DELETE FROM hen_batches WHERE id = $1 AND tenant_id = $2
	`, batchID, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete hen batch")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Hen batch deleted successfully",
	})
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
			"id":             mortalityID,
			"batch_id":       req.BatchID,
			"mortality_date": req.MortalityDate,
			"count":          req.Count,
		},
	})
}

// GetMortalityHistory returns mortality history for a batch
func GetMortalityHistory(w http.ResponseWriter, r *http.Request) {
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

	// Verify batch belongs to tenant
	var batchTenantID uuid.UUID
	err = database.DB.QueryRow(`
		SELECT tenant_id FROM hen_batches WHERE id = $1
	`, batchID).Scan(&batchTenantID)

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

	// Get mortality history
	rows, err := database.DB.Query(`
		SELECT id, batch_id, mortality_date, count, reason, notes, recorded_by_user_id, created_at
		FROM hen_mortality
		WHERE batch_id = $1
		ORDER BY mortality_date DESC, created_at DESC
	`, batchID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch mortality history")
		return
	}
	defer rows.Close()

	var mortalityRecords []map[string]interface{}
	for rows.Next() {
		var record struct {
			ID               int
			BatchID          int
			MortalityDate    time.Time
			Count            int
			Reason           sql.NullString
			Notes            sql.NullString
			RecordedByUserID sql.NullInt64
			CreatedAt        time.Time
		}

		err := rows.Scan(
			&record.ID, &record.BatchID, &record.MortalityDate,
			&record.Count, &record.Reason, &record.Notes,
			&record.RecordedByUserID, &record.CreatedAt,
		)
		if err != nil {
			continue
		}

		mortalityRecord := map[string]interface{}{
			"id":             record.ID,
			"batch_id":       record.BatchID,
			"mortality_date": record.MortalityDate.Format("2006-01-02"),
			"count":          record.Count,
			"created_at":     record.CreatedAt.Format(time.RFC3339),
		}

		if record.Reason.Valid {
			mortalityRecord["reason"] = record.Reason.String
		}
		if record.Notes.Valid {
			mortalityRecord["notes"] = record.Notes.String
		}
		if record.RecordedByUserID.Valid {
			mortalityRecord["recorded_by_user_id"] = record.RecordedByUserID.Int64
		}

		mortalityRecords = append(mortalityRecords, mortalityRecord)
	}

	if err = rows.Err(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading mortality records")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    mortalityRecords,
	})
}

type HenBatchSaleCreateRequest struct {
	BatchID     int     `json:"batch_id"`
	SaleDate    string  `json:"sale_date"` // YYYY-MM-DD
	Count       int     `json:"count"`
	PricePerHen float64 `json:"price_per_hen"`
	Notes       *string `json:"notes,omitempty"`
}

// CreateHenBatchSale records a partial/full hen batch sale and inserts matching income transaction.
func CreateHenBatchSale(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req HenBatchSaleCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Count <= 0 {
		respondWithError(w, http.StatusBadRequest, "count must be greater than 0")
		return
	}
	if req.PricePerHen < 0 {
		respondWithError(w, http.StatusBadRequest, "price_per_hen cannot be negative")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	// Verify batch belongs to tenant and fetch batch name.
	var batchName string
	err = database.DB.QueryRow(`
		SELECT batch_name
		FROM hen_batches
		WHERE id = $1 AND tenant_id = $2
	`, req.BatchID, tenantID).Scan(&batchName)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Hen batch not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	saleDate, err := time.Parse("2006-01-02", req.SaleDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid sale date format. Use YYYY-MM-DD")
		return
	}

	totalAmount := float64(req.Count) * req.PricePerHen
	periodMonth := time.Date(saleDate.Year(), saleDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	tx, err := database.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to begin transaction")
		return
	}
	defer tx.Rollback()

	// Insert sale event (trigger updates current_count).
	var saleID int
	err = tx.QueryRow(`
		INSERT INTO hen_batch_sales (
			batch_id, sale_date, count, price_per_hen, total_amount, notes, recorded_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, req.BatchID, saleDate, req.Count, req.PricePerHen, totalAmount, req.Notes, user.UserID).Scan(&saleID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to record sale: %v", err))
		return
	}

	itemName := fmt.Sprintf("HEN BATCH SALE - %s", batchName)
	transactionNotes := req.Notes

	_, err = tx.Exec(`
		INSERT INTO transactions (
			tenant_id, transaction_date, transaction_type, category,
			item_name, quantity, unit, rate, amount, notes,
			payment_date, period_month
		)
		VALUES ($1, $2, 'INCOME', 'OTHER', $3, $4, 'NOS', $5, $6, $7, $8, $9)
	`, tenantID, saleDate, itemName, req.Count, req.PricePerHen, totalAmount, transactionNotes, saleDate, periodMonth)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create income transaction: %v", err))
		return
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to commit sale transaction")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Hen batch sale recorded successfully",
		"data": map[string]interface{}{
			"id":            saleID,
			"batch_id":      req.BatchID,
			"sale_date":     saleDate.Format("2006-01-02"),
			"count":         req.Count,
			"price_per_hen": req.PricePerHen,
			"total_amount":  totalAmount,
		},
	})
}

// GetHenBatchSales returns sale history for a batch.
func GetHenBatchSales(w http.ResponseWriter, r *http.Request) {
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

	// Verify batch belongs to tenant.
	var batchTenantID uuid.UUID
	err = database.DB.QueryRow(`
		SELECT tenant_id FROM hen_batches WHERE id = $1
	`, batchID).Scan(&batchTenantID)
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

	rows, err := database.DB.Query(`
		SELECT id, batch_id, sale_date, count, price_per_hen, total_amount,
		       notes, recorded_by_user_id, created_at
		FROM hen_batch_sales
		WHERE batch_id = $1
		ORDER BY sale_date DESC, created_at DESC
	`, batchID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch sale history")
		return
	}
	defer rows.Close()

	var sales []map[string]interface{}
	for rows.Next() {
		var (
			id               int
			bID              int
			saleDate         time.Time
			count            int
			pricePerHen      float64
			totalAmount      float64
			notes            sql.NullString
			recordedByUserID sql.NullInt64
			createdAt        time.Time
		)
		if err := rows.Scan(&id, &bID, &saleDate, &count, &pricePerHen, &totalAmount, &notes, &recordedByUserID, &createdAt); err != nil {
			continue
		}

		sale := map[string]interface{}{
			"id":            id,
			"batch_id":      bID,
			"sale_date":     saleDate.Format("2006-01-02"),
			"count":         count,
			"price_per_hen": pricePerHen,
			"total_amount":  totalAmount,
			"created_at":    createdAt.Format(time.RFC3339),
		}
		if notes.Valid {
			sale["notes"] = notes.String
		}
		if recordedByUserID.Valid {
			sale["recorded_by_user_id"] = recordedByUserID.Int64
		}
		sales = append(sales, sale)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    sales,
	})
}
