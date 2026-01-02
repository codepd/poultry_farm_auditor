package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"poultry-farm-api/models"
	"poultry-farm-api/utils"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GetTransactions returns transactions with filters
func GetTransactions(w http.ResponseWriter, r *http.Request) {
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

	// Get permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get permissions")
		return
	}

	// Parse query parameters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	category := r.URL.Query().Get("category")
	// status := r.URL.Query().Get("status") // Status column doesn't exist in transactions table
	transactionType := r.URL.Query().Get("transaction_type")

	// Build query - exclude columns that may not exist in the actual database table
	// (submitted_by_user_id, approved_by_user_id, approved_at, status)
	query := `
		SELECT id, tenant_id, transaction_date, transaction_type, category,
		       item_name, quantity, unit, rate, amount, notes,
		       created_at, updated_at
		FROM transactions
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIndex := 2

	if startDate != "" {
		query += fmt.Sprintf(" AND transaction_date >= $%d", argIndex)
		args = append(args, startDate)
		argIndex++
	}
	if endDate != "" {
		query += fmt.Sprintf(" AND transaction_date <= $%d", argIndex)
		args = append(args, endDate)
		argIndex++
	}
	if category != "" {
		// Cast to category_enum to ensure type safety
		query += fmt.Sprintf(" AND category = $%d::category_enum", argIndex)
		args = append(args, category)
		argIndex++
	}
	// Note: status column doesn't exist in transactions table, so we skip status filter
	// if status != "" {
	// 	query += fmt.Sprintf(" AND status = $%d", argIndex)
	// 	args = append(args, status)
	// 	argIndex++
	// }
	if transactionType != "" {
		// Handle special case: EXPENSE might not be in the database enum
		// For OTHER category, EXPENSE transactions are typically stored as PURCHASE
		actualTransactionType := transactionType
		if transactionType == "EXPENSE" {
			if category == "OTHER" {
				// OTHER expenses are stored as PURCHASE transactions
				actualTransactionType = "PURCHASE"
			} else {
				// For other categories, try EXPENSE but it might fail if not in enum
				// We'll let the database error if EXPENSE doesn't exist
			}
		}

		// Cast to transaction_type_enum to ensure type safety
		query += fmt.Sprintf(" AND transaction_type = $%d::transaction_type_enum", argIndex)
		args = append(args, actualTransactionType)
		argIndex++
	}

	query += " ORDER BY transaction_date DESC, created_at DESC LIMIT 1000"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		// Log the actual error for debugging
		fmt.Printf("Error querying transactions: %v\n", err)
		fmt.Printf("Query: %s\n", query)
		fmt.Printf("Args: %v\n", args)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch transactions: %v", err))
		return
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var txn models.Transaction
		var itemName, unit, notes sql.NullString
		var quantity, rate sql.NullFloat64

		err := rows.Scan(
			&txn.ID, &txn.TenantID, &txn.TransactionDate, &txn.TransactionType,
			&txn.Category, &itemName, &quantity, &unit, &rate, &txn.Amount,
			&notes, &txn.CreatedAt, &txn.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Set default values for columns that don't exist in the database
		txn.Status = "APPROVED"
		txn.SubmittedByUserID = nil
		txn.ApprovedByUserID = nil
		txn.ApprovedAt = nil

		if itemName.Valid {
			txn.ItemName = itemName
		}
		if quantity.Valid {
			txn.Quantity = quantity
		}
		if unit.Valid {
			txn.Unit = unit
		}
		if rate.Valid {
			txn.Rate = rate
		}
		if notes.Valid {
			txn.Notes = notes
		}

		// Filter sensitive data based on permissions
		shouldHide, _ := utils.IsDataSensitive(database.DB, tenantID, "EGGS_SOLD", perms.CanViewSensitiveData)
		if shouldHide && txn.Category == "EGG" && txn.TransactionType == "SALE" {
			txn.Amount = 0 // Hide amount
		}

		shouldHide, _ = utils.IsDataSensitive(database.DB, tenantID, "FEED_PURCHASED", perms.CanViewSensitiveData)
		if shouldHide && txn.Category == "FEED" && txn.TransactionType == "PURCHASE" {
			txn.Amount = 0 // Hide amount
		}

		transactions = append(transactions, txn)
	}

	// Return transactions with message if empty
	message := ""
	if len(transactions) == 0 {
		if category == "OTHER" && transactionType == "EXPENSE" {
			message = "No expenses found for this tenant"
		} else {
			message = "No transactions found matching the criteria"
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    transactions,
		"message": message,
	})
}

// GetTransaction returns a single transaction
func GetTransaction(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	transactionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	var txn models.Transaction
	var itemName, unit, notes sql.NullString
	var quantity, rate sql.NullFloat64

	err = database.DB.QueryRow(`
		SELECT id, tenant_id, transaction_date, transaction_type, category,
		       item_name, quantity, unit, rate, amount, notes,
		       created_at, updated_at
		FROM transactions
		WHERE id = $1 AND tenant_id = $2
	`, transactionID, tenantID).Scan(
		&txn.ID, &txn.TenantID, &txn.TransactionDate, &txn.TransactionType,
		&txn.Category, &itemName, &quantity, &unit, &rate, &txn.Amount,
		&notes, &txn.CreatedAt, &txn.UpdatedAt,
	)

	// Set default values for columns that don't exist in the database
	txn.Status = "APPROVED"
	txn.SubmittedByUserID = nil
	txn.ApprovedByUserID = nil
	txn.ApprovedAt = nil

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Transaction not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if itemName.Valid {
		txn.ItemName = itemName
	}
	if quantity.Valid {
		txn.Quantity = quantity
	}
	if unit.Valid {
		txn.Unit = unit
	}
	if rate.Valid {
		txn.Rate = rate
	}
	if notes.Valid {
		txn.Notes = notes
	}

	// Check sensitive data permissions
	perms, _ := middleware.GetUserPermissions(user.UserID, user.TenantID)
	shouldHide, _ := utils.IsDataSensitive(database.DB, tenantID, "EGGS_SOLD", perms.CanViewSensitiveData)
	if shouldHide && txn.Category == "EGG" && txn.TransactionType == "SALE" {
		txn.Amount = 0
	}

	shouldHide, _ = utils.IsDataSensitive(database.DB, tenantID, "FEED_PURCHASED", perms.CanViewSensitiveData)
	if shouldHide && txn.Category == "FEED" && txn.TransactionType == "PURCHASE" {
		txn.Amount = 0
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    txn,
	})
}

type TransactionCreateRequest struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	TransactionDate time.Time `json:"transaction_date"`
	TransactionType string    `json:"transaction_type"`
	Category        string    `json:"category"`
	ItemName        *string   `json:"item_name,omitempty"`
	Quantity        *float64  `json:"quantity,omitempty"`
	Unit            *string   `json:"unit,omitempty"`
	Rate            *float64  `json:"rate,omitempty"`
	Amount          float64   `json:"amount"`
	Notes           *string   `json:"notes,omitempty"`
}

// CreateTransaction creates a new transaction
func CreateTransaction(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req TransactionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	// Use tenant from token if not provided
	if req.TenantID == uuid.Nil {
		req.TenantID = tenantID
	}

	// Verify user has access to this tenant
	var hasAccess bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM tenant_users 
			WHERE user_id = $1 AND tenant_id = $2
		)
	`, user.UserID, req.TenantID).Scan(&hasAccess)

	if err != nil || !hasAccess {
		respondWithError(w, http.StatusForbidden, "Access denied to this tenant")
		return
	}

	// Get permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanEditTransactions {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions to create transactions")
		return
	}

	// Determine status based on permissions
	status := "DRAFT"
	submittedByUserID := &user.UserID
	if !perms.CanApproveTransactions {
		// User must submit for approval
		status = "SUBMITTED"
	} else {
		// User can approve, so auto-approve
		status = "APPROVED"
		submittedByUserID = nil
		approvedByUserID := &user.UserID
		approvedAt := time.Now()

		// Insert with approval
		var txnID int
		err = database.DB.QueryRow(`
			INSERT INTO transactions (
				tenant_id, transaction_date, transaction_type, category,
				item_name, quantity, unit, rate, amount, notes, status,
				approved_by_user_id, approved_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING id
		`, req.TenantID, req.TransactionDate, req.TransactionType, req.Category,
			req.ItemName, req.Quantity, req.Unit, req.Rate, req.Amount, req.Notes,
			status, approvedByUserID, approvedAt).Scan(&txnID)

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create transaction")
			return
		}

		// Return created transaction
		vars := mux.Vars(r)
		vars["id"] = strconv.Itoa(txnID)
		GetTransaction(w, r)
		return
	}

	// Insert as DRAFT or SUBMITTED
	var txnID int
	err = database.DB.QueryRow(`
		INSERT INTO transactions (
			tenant_id, transaction_date, transaction_type, category,
			item_name, quantity, unit, rate, amount, notes, status,
			submitted_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, req.TenantID, req.TransactionDate, req.TransactionType, req.Category,
		req.ItemName, req.Quantity, req.Unit, req.Rate, req.Amount, req.Notes,
		status, submittedByUserID).Scan(&txnID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create transaction")
		return
	}

	// Return created transaction
	vars := mux.Vars(r)
	vars["id"] = strconv.Itoa(txnID)
	GetTransaction(w, r)
}

// UpdateTransaction updates an existing transaction
func UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	transactionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	// Check if transaction exists and user has access
	var currentStatus string
	err = database.DB.QueryRow(`
		SELECT status FROM transactions
		WHERE id = $1 AND tenant_id = $2
	`, transactionID, tenantID).Scan(&currentStatus)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Transaction not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Get permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanEditTransactions {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	// Can only update DRAFT or SUBMITTED transactions
	if currentStatus == "APPROVED" || currentStatus == "REJECTED" {
		respondWithError(w, http.StatusBadRequest, "Cannot update approved or rejected transactions")
		return
	}

	var req TransactionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update transaction
	_, err = database.DB.Exec(`
		UPDATE transactions
		SET transaction_date = $1, transaction_type = $2, category = $3,
		    item_name = $4, quantity = $5, unit = $6, rate = $7,
		    amount = $8, notes = $9, updated_at = CURRENT_TIMESTAMP
		WHERE id = $10 AND tenant_id = $11
	`, req.TransactionDate, req.TransactionType, req.Category,
		req.ItemName, req.Quantity, req.Unit, req.Rate, req.Amount, req.Notes,
		transactionID, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update transaction")
		return
	}

	GetTransaction(w, r)
}

// SubmitTransaction submits a DRAFT transaction for approval
func SubmitTransaction(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	transactionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	// Update status to SUBMITTED
	_, err = database.DB.Exec(`
		UPDATE transactions
		SET status = 'SUBMITTED', submitted_by_user_id = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND tenant_id = $3 AND status = 'DRAFT'
	`, user.UserID, transactionID, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to submit transaction")
		return
	}

	GetTransaction(w, r)
}

// ApproveTransaction approves a SUBMITTED transaction
func ApproveTransaction(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	transactionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanApproveTransactions {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions to approve")
		return
	}

	// Update status to APPROVED
	approvedAt := time.Now()
	_, err = database.DB.Exec(`
		UPDATE transactions
		SET status = 'APPROVED', approved_by_user_id = $1, approved_at = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND tenant_id = $4 AND status = 'SUBMITTED'
	`, user.UserID, approvedAt, transactionID, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to approve transaction")
		return
	}

	GetTransaction(w, r)
}

// RejectTransaction rejects a SUBMITTED transaction
func RejectTransaction(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	transactionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanApproveTransactions {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions to reject")
		return
	}

	// Update status to REJECTED
	_, err = database.DB.Exec(`
		UPDATE transactions
		SET status = 'REJECTED', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND tenant_id = $2 AND status = 'SUBMITTED'
	`, transactionID, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to reject transaction")
		return
	}

	GetTransaction(w, r)
}

// DeleteTransaction deletes a DRAFT transaction
func DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	transactionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanEditTransactions {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	// Only allow deletion of DRAFT transactions
	_, err = database.DB.Exec(`
		DELETE FROM transactions
		WHERE id = $1 AND tenant_id = $2 AND status = 'DRAFT'
	`, transactionID, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete transaction")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Transaction deleted",
	})
}
