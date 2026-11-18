package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type DailyLogSubmission struct {
	ID                int     `json:"id"`
	TenantID          string  `json:"tenant_id"`
	LogType           string  `json:"log_type"` // EGG_SALE, FEED_PURCHASE, EXPENSE, MORTALITY
	LogDate           string  `json:"log_date"`
	Category          string  `json:"category"`
	ItemName          string  `json:"item_name"`
	Quantity          float64 `json:"quantity"`
	Unit              string  `json:"unit"`
	Rate              float64 `json:"rate"`
	Amount            float64 `json:"amount"`
	HenBatchID        int     `json:"hen_batch_id,omitempty"`
	Notes             string  `json:"notes,omitempty"`
	Status            string  `json:"status"` // PENDING_APPROVAL, APPROVED, REJECTED, RECONCILED
	SubmittedByUserID int     `json:"submitted_by_user_id"`
	SubmittedByName   string  `json:"submitted_by_name,omitempty"`
	ReviewedByUserID  *int    `json:"reviewed_by_user_id,omitempty"`
	ReviewedAt        string  `json:"reviewed_at,omitempty"`
	ReviewNotes       string  `json:"review_notes,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

func CanApproveDailyLogs(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "OWNER", "CO_OWNER", "ADMIN", "MANAGER":
		return true
	default:
		return false
	}
}

func ResolveInitialDailyLogStatus(role string) string {
	if CanApproveDailyLogs(role) {
		return "APPROVED"
	}
	return "PENDING_APPROVAL"
}

func SubmitDailyLog(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	var req struct {
		LogType    string  `json:"log_type"`
		LogDate    string  `json:"log_date"`
		Category   string  `json:"category"`
		ItemName   string  `json:"item_name"`
		Quantity   float64 `json:"quantity"`
		Unit       string  `json:"unit"`
		Rate       float64 `json:"rate"`
		Amount     float64 `json:"amount"`
		HenBatchID int     `json:"hen_batch_id"`
		Notes      string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	logType := strings.ToUpper(strings.TrimSpace(req.LogType))
	switch logType {
	case "EGG_SALE", "FEED_PURCHASE", "EXPENSE", "MORTALITY":
	default:
		respondWithError(w, http.StatusBadRequest, "Invalid log_type")
		return
	}

	parsedDate, err := time.Parse("2006-01-02", req.LogDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid log_date; expected YYYY-MM-DD")
		return
	}

	if logType == "MORTALITY" {
		if req.HenBatchID <= 0 || req.Quantity <= 0 {
			respondWithError(w, http.StatusBadRequest, "hen_batch_id and positive quantity are required for mortality")
			return
		}
	} else if req.Amount <= 0 && (req.Quantity <= 0 || req.Rate <= 0) {
		respondWithError(w, http.StatusBadRequest, "Positive amount (or quantity and rate) is required")
		return
	}

	amount := req.Amount
	if amount <= 0 && req.Quantity > 0 && req.Rate > 0 {
		amount = req.Quantity * req.Rate
	}

	initialStatus := ResolveInitialDailyLogStatus(user.Role)

	var id int
	err = database.DB.QueryRow(`
		INSERT INTO daily_log_submissions (
			tenant_id, log_type, log_date, category, item_name,
			quantity, unit, rate, amount, hen_batch_id, notes,
			status, submitted_by_user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, 0), $11, $12, $13)
		RETURNING id
	`, tenantID, logType, parsedDate, strings.ToUpper(strings.TrimSpace(req.Category)),
		strings.TrimSpace(req.ItemName), req.Quantity, strings.TrimSpace(req.Unit),
		req.Rate, amount, req.HenBatchID, strings.TrimSpace(req.Notes),
		initialStatus, user.UserID,
	).Scan(&id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save daily log submission")
		return
	}

	if initialStatus == "APPROVED" {
		if err := commitApprovedDailyLog(id, tenantID, user.UserID); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Saved submission but failed to apply to ledger")
			return
		}
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":     id,
			"status": initialStatus,
		},
	})
}

func ListDailyLogs(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))

	query := `
		SELECT d.id, d.tenant_id, d.log_type, d.log_date, COALESCE(d.category, ''),
		       COALESCE(d.item_name, ''), COALESCE(d.quantity, 0), COALESCE(d.unit, ''),
		       COALESCE(d.rate, 0), COALESCE(d.amount, 0), COALESCE(d.hen_batch_id, 0),
		       COALESCE(d.notes, ''), d.status, d.submitted_by_user_id,
		       COALESCE(NULLIF(u.full_name, ''), COALESCE(u.email, u.phone, 'User')),
		       d.reviewed_by_user_id, d.reviewed_at, COALESCE(d.review_notes, ''), d.created_at
		FROM daily_log_submissions d
		LEFT JOIN users u ON u.id = d.submitted_by_user_id
		WHERE d.tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if !CanApproveDailyLogs(user.Role) {
		query += " AND d.submitted_by_user_id = $" + strconv.Itoa(argIdx)
		args = append(args, user.UserID)
		argIdx++
	}
	if statusFilter != "" {
		query += " AND d.status = $" + strconv.Itoa(argIdx)
		args = append(args, statusFilter)
		argIdx++
	}
	query += " ORDER BY d.created_at DESC, d.id DESC LIMIT 100"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load daily logs")
		return
	}
	defer rows.Close()

	list := []DailyLogSubmission{}
	for rows.Next() {
		var item DailyLogSubmission
		var logDate, createdAt time.Time
		var reviewedBy sql.NullInt64
		var reviewedAt sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.LogType, &logDate, &item.Category,
			&item.ItemName, &item.Quantity, &item.Unit, &item.Rate, &item.Amount,
			&item.HenBatchID, &item.Notes, &item.Status, &item.SubmittedByUserID,
			&item.SubmittedByName, &reviewedBy, &reviewedAt, &item.ReviewNotes, &createdAt,
		); err != nil {
			continue
		}
		item.LogDate = logDate.Format("2006-01-02")
		item.CreatedAt = createdAt.Format(time.RFC3339)
		if reviewedBy.Valid {
			v := int(reviewedBy.Int64)
			item.ReviewedByUserID = &v
		}
		if reviewedAt.Valid {
			item.ReviewedAt = reviewedAt.Time.Format(time.RFC3339)
		}
		list = append(list, item)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    list,
	})
}

func ApproveDailyLog(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if !CanApproveDailyLogs(user.Role) {
		respondWithError(w, http.StatusForbidden, "Only Owner, Co-Owner, Admin, or Manager can approve submissions")
		return
	}
	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid submission ID")
		return
	}

	if err := commitApprovedDailyLog(id, tenantID, user.UserID); err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Pending submission not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to approve submission")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Daily log approved and applied to farm records",
	})
}

func RejectDailyLog(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if !CanApproveDailyLogs(user.Role) {
		respondWithError(w, http.StatusForbidden, "Only Owner, Co-Owner, Admin, or Manager can reject submissions")
		return
	}
	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid submission ID")
		return
	}

	var req struct {
		ReviewNotes string `json:"review_notes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	res, err := database.DB.Exec(`
		UPDATE daily_log_submissions
		SET status = 'REJECTED', reviewed_by_user_id = $1, reviewed_at = CURRENT_TIMESTAMP,
		    review_notes = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND tenant_id = $4 AND status = 'PENDING_APPROVAL'
	`, user.UserID, strings.TrimSpace(req.ReviewNotes), id, tenantID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to reject submission")
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		respondWithError(w, http.StatusNotFound, "Pending submission not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Daily log submission rejected",
	})
}

func commitApprovedDailyLog(submissionID int, tenantID uuid.UUID, reviewerUserID int) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var logType, category, itemName, unit, notes, status string
	var logDate time.Time
	var qty, rate, amount float64
	var batchID sql.NullInt64
	err = tx.QueryRow(`
		SELECT log_type, log_date, COALESCE(category, ''), COALESCE(item_name, ''),
		       COALESCE(quantity, 0), COALESCE(unit, ''), COALESCE(rate, 0),
		       COALESCE(amount, 0), hen_batch_id, COALESCE(notes, ''), status
		FROM daily_log_submissions
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE
	`, submissionID, tenantID).Scan(
		&logType, &logDate, &category, &itemName, &qty, &unit, &rate, &amount, &batchID, &notes, &status,
	)
	if err != nil {
		return err
	}

	if logType == "MORTALITY" {
		if !batchID.Valid {
			return sql.ErrNoRows
		}
		_, err = tx.Exec(`
			INSERT INTO hen_mortality (batch_id, mortality_date, count, reason, notes, recorded_by_user_id)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, batchID.Int64, logDate, int(qty), itemName, notes, reviewerUserID)
		if err != nil {
			return err
		}
	} else {
		txType := "EXPENSE"
		if logType == "EGG_SALE" {
			txType = "SALE"
		} else if logType == "FEED_PURCHASE" {
			txType = "PURCHASE"
		}
		_, err = tx.Exec(`
			INSERT INTO transactions (
				tenant_id, transaction_date, transaction_type, category,
				item_name, quantity, unit, rate, amount, notes, source
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'DAILY_LOG')
		`, tenantID, logDate, txType, category, itemName, qty, unit, rate, amount, notes)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`
		UPDATE daily_log_submissions
		SET status = 'APPROVED', reviewed_by_user_id = $1, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND tenant_id = $3
	`, reviewerUserID, submissionID, tenantID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	InvalidateAnalyticsCache(tenantID.String())
	return nil
}
