package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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
		       payment_date, period_month, period_week, period_days,
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
		var paymentDate, periodMonth sql.NullTime
		var periodWeek, periodDays sql.NullInt64

		err := rows.Scan(
			&txn.ID, &txn.TenantID, &txn.TransactionDate, &txn.TransactionType,
			&txn.Category, &itemName, &quantity, &unit, &rate, &txn.Amount,
			&notes, &paymentDate, &periodMonth, &periodWeek, &periodDays,
			&txn.CreatedAt, &txn.UpdatedAt,
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
		if paymentDate.Valid {
			txn.PaymentDate = &paymentDate.Time
		}
		if periodMonth.Valid {
			txn.PeriodMonth = &periodMonth.Time
		}
		if periodWeek.Valid {
			week := int(periodWeek.Int64)
			txn.PeriodWeek = &week
		}
		if periodDays.Valid {
			days := int(periodDays.Int64)
			txn.PeriodDays = &days
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
	var paymentDate, periodMonth sql.NullTime
	var periodWeek, periodDays sql.NullInt64

	err = database.DB.QueryRow(`
		SELECT id, tenant_id, transaction_date, transaction_type, category,
		       item_name, quantity, unit, rate, amount, notes,
		       payment_date, period_month, period_week, period_days,
		       created_at, updated_at
		FROM transactions
		WHERE id = $1 AND tenant_id = $2
	`, transactionID, tenantID).Scan(
		&txn.ID, &txn.TenantID, &txn.TransactionDate, &txn.TransactionType,
		&txn.Category, &itemName, &quantity, &unit, &rate, &txn.Amount,
		&notes, &paymentDate, &periodMonth, &periodWeek, &periodDays,
		&txn.CreatedAt, &txn.UpdatedAt,
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
	if paymentDate.Valid {
		txn.PaymentDate = &paymentDate.Time
	}
	if periodMonth.Valid {
		txn.PeriodMonth = &periodMonth.Time
	}
	if periodWeek.Valid {
		week := int(periodWeek.Int64)
		txn.PeriodWeek = &week
	}
	if periodDays.Valid {
		days := int(periodDays.Int64)
		txn.PeriodDays = &days
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

// Date is a custom type for JSON dates that can parse "YYYY-MM-DD" format
type Date time.Time

func (d *Date) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) > 0 && s[0] == '"' {
		s = s[1 : len(s)-1] // Remove quotes
	}
	if s == "" || s == "null" {
		return nil
	}
	// Try RFC3339 first (includes date-only format "2006-01-02")
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		// Try RFC3339 full format
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return err
		}
	}
	*d = Date(t)
	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).Format("2006-01-02"))
}

func (d Date) Time() time.Time {
	return time.Time(d)
}

// convertDateToTimezone converts a date to a specific timezone
// For DATE fields, this interprets the date in the given timezone
func convertDateToTimezone(t time.Time, location *time.Location) time.Time {
	if t.IsZero() {
		return t
	}
	// Convert to the target timezone and extract just the date part (midnight)
	t = t.In(location)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, location)
}

type TransactionCreateRequest struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	TransactionDate Date      `json:"transaction_date"`
	TransactionType string    `json:"transaction_type"`
	Category        string    `json:"category"`
	ItemName        *string   `json:"item_name,omitempty"`
	Quantity        *float64  `json:"quantity,omitempty"`
	Unit            *string   `json:"unit,omitempty"`
	Rate            *float64  `json:"rate,omitempty"`
	Amount          float64   `json:"amount"`
	Notes           *string   `json:"notes,omitempty"`
	PaymentDate     *Date     `json:"payment_date,omitempty"`     // Date when payment was made (defaults to transaction_date)
	PeriodMonth     *Date     `json:"period_month,omitempty"`     // Month the payment is for (stored as first day of month, defaults to payment_date month)
	PeriodWeek      *int      `json:"period_week,omitempty"`      // Week number within the payment period (optional)
	PeriodDays      *int      `json:"period_days,omitempty"`      // Number of days the payment covers (optional)
}

// Helper function to set default payment_date and period_month
func setPaymentPeriodDefaults(req *TransactionCreateRequest) {
	// Set default payment_date to transaction_date if not provided
	if req.PaymentDate == nil {
		paymentDate := req.TransactionDate
		req.PaymentDate = &paymentDate
	}

	// Set default period_month to payment_date's month if not provided
	if req.PeriodMonth == nil {
		// Get first day of the payment_date's month
		paymentDate := req.PaymentDate.Time()
		periodMonth := Date(time.Date(paymentDate.Year(), paymentDate.Month(), 1, 0, 0, 0, 0, paymentDate.Location()))
		req.PeriodMonth = &periodMonth
	}
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
		log.Printf("Error decoding transaction request: %v", err)
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
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

	// Validate required fields
	if req.ItemName == nil || *req.ItemName == "" {
		respondWithError(w, http.StatusBadRequest, "item_name is required")
		return
	}

	// Get tenant timezone and convert dates to tenant timezone
	tenantLocation, err := utils.GetTenantTimezoneLocation(req.TenantID)
	if err != nil {
		log.Printf("Error getting tenant timezone, using UTC: %v", err)
		tenantLocation = time.UTC
	}

	// Convert dates to tenant timezone context
	transactionDateConverted := convertDateToTimezone(req.TransactionDate.Time(), tenantLocation)
	req.TransactionDate = Date(transactionDateConverted)
	if req.PaymentDate != nil {
		paymentDateConverted := convertDateToTimezone(req.PaymentDate.Time(), tenantLocation)
		paymentDate := Date(paymentDateConverted)
		req.PaymentDate = &paymentDate
	}
	if req.PeriodMonth != nil {
		periodMonthConverted := convertDateToTimezone(req.PeriodMonth.Time(), tenantLocation)
		periodMonth := Date(periodMonthConverted)
		req.PeriodMonth = &periodMonth
	}

	// Set default payment_date and period_month if not provided
	setPaymentPeriodDefaults(&req)

	// Insert transaction (status, submitted_by_user_id, approved_by_user_id columns don't exist in DB)
	var txnID int
	
	// Convert Date types to time.Time for database
	paymentDate := (*time.Time)(nil)
	if req.PaymentDate != nil {
		t := req.PaymentDate.Time()
		paymentDate = &t
	}
	periodMonth := (*time.Time)(nil)
	if req.PeriodMonth != nil {
		t := req.PeriodMonth.Time()
		periodMonth = &t
	}
	
	// Handle optional integer fields - use sql.NullInt64 for proper NULL handling
	var periodWeek sql.NullInt64
	if req.PeriodWeek != nil {
		periodWeek = sql.NullInt64{Int64: int64(*req.PeriodWeek), Valid: true}
	}
	var periodDays sql.NullInt64
	if req.PeriodDays != nil {
		periodDays = sql.NullInt64{Int64: int64(*req.PeriodDays), Valid: true}
	}
	
	err = database.DB.QueryRow(`
		INSERT INTO transactions (
			tenant_id, transaction_date, transaction_type, category,
			item_name, quantity, unit, rate, amount, notes,
			payment_date, period_month, period_week, period_days
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`, req.TenantID, req.TransactionDate.Time(), req.TransactionType, req.Category,
		req.ItemName, req.Quantity, req.Unit, req.Rate, req.Amount, req.Notes,
		paymentDate, periodMonth, periodWeek, periodDays).Scan(&txnID)

	if err != nil {
		log.Printf("Error creating transaction: %v", err)
		log.Printf("Request data: TenantID=%v, TransactionDate=%v, TransactionType=%v, Category=%v, Amount=%v", 
			req.TenantID, req.TransactionDate.Time(), req.TransactionType, req.Category, req.Amount)
		log.Printf("PaymentDate=%v, PeriodMonth=%v, PeriodWeek=%v, PeriodDays=%v",
			paymentDate, periodMonth, req.PeriodWeek, req.PeriodDays)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create transaction: %v", err))
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
		log.Printf("Error decoding transaction update request: %v", err)
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Get tenant timezone and convert dates to tenant timezone
	tenantLocation, err := utils.GetTenantTimezoneLocation(tenantID)
	if err != nil {
		log.Printf("Error getting tenant timezone, using UTC: %v", err)
		tenantLocation = time.UTC
	}

	// Convert dates to tenant timezone context
	transactionDateConverted := convertDateToTimezone(req.TransactionDate.Time(), tenantLocation)
	req.TransactionDate = Date(transactionDateConverted)
	if req.PaymentDate != nil {
		paymentDateConverted := convertDateToTimezone(req.PaymentDate.Time(), tenantLocation)
		paymentDate := Date(paymentDateConverted)
		req.PaymentDate = &paymentDate
	}
	if req.PeriodMonth != nil {
		periodMonthConverted := convertDateToTimezone(req.PeriodMonth.Time(), tenantLocation)
		periodMonth := Date(periodMonthConverted)
		req.PeriodMonth = &periodMonth
	}

	// Set default payment_date and period_month if not provided
	setPaymentPeriodDefaults(&req)

	// Convert Date types to time.Time for database
	paymentDate := (*time.Time)(nil)
	if req.PaymentDate != nil {
		t := req.PaymentDate.Time()
		paymentDate = &t
	}
	periodMonth := (*time.Time)(nil)
	if req.PeriodMonth != nil {
		t := req.PeriodMonth.Time()
		periodMonth = &t
	}

	// Handle optional integer fields - use sql.NullInt64 for proper NULL handling
	var periodWeek sql.NullInt64
	if req.PeriodWeek != nil {
		periodWeek = sql.NullInt64{Int64: int64(*req.PeriodWeek), Valid: true}
	}
	var periodDays sql.NullInt64
	if req.PeriodDays != nil {
		periodDays = sql.NullInt64{Int64: int64(*req.PeriodDays), Valid: true}
	}

	// Update transaction
	_, err = database.DB.Exec(`
		UPDATE transactions
		SET transaction_date = $1, transaction_type = $2, category = $3,
		    item_name = $4, quantity = $5, unit = $6, rate = $7,
		    amount = $8, notes = $9,
		    payment_date = $10, period_month = $11, period_week = $12, period_days = $13,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $14 AND tenant_id = $15
	`, req.TransactionDate.Time(), req.TransactionType, req.Category,
		req.ItemName, req.Quantity, req.Unit, req.Rate, req.Amount, req.Notes,
		paymentDate, periodMonth, periodWeek, periodDays,
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
