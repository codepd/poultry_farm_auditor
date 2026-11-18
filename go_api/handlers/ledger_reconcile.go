package handlers

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type ReconcileItem struct {
	ID       int     `json:"id,omitempty"`
	Date     string  `json:"date"`     // YYYY-MM-DD
	Category string  `json:"category"` // EGG, FEED, MEDICINE, OTHER
	ItemName string  `json:"item_name"`
	Quantity float64 `json:"quantity"`
	Amount   float64 `json:"amount"`
	Source   string  `json:"source"` // LEDGER or DAILY_LOG
}

type ReconciliationMatch struct {
	Status            string         `json:"status"` // EXACT_MATCH, DATE_SHIFT, MISMATCH, LEDGER_ONLY, DAILY_LOG_ONLY
	Color             string         `json:"color"`  // green, yellow, red
	DateDiffDays      int            `json:"date_diff_days,omitempty"`
	DefaultResolution string         `json:"default_resolution"` // KEEP_LEDGER, KEEP_DAILY_LOG, KEEP_BOTH
	LedgerItem        *ReconcileItem `json:"ledger_item,omitempty"`
	DailyLogItem      *ReconcileItem `json:"daily_log_item,omitempty"`
}

type ReconciliationSummary struct {
	ExactMatch   int `json:"exact_match"`
	DateShift    int `json:"date_shift"`
	Mismatch     int `json:"mismatch"`
	LedgerOnly   int `json:"ledger_only"`
	DailyLogOnly int `json:"daily_log_only"`
}

type ReconciliationReport struct {
	Summary ReconciliationSummary `json:"summary"`
	Matches []ReconciliationMatch `json:"matches"`
}

func reconcileValuesMatch(a, b ReconcileItem) bool {
	if a.Quantity > 0 && b.Quantity > 0 && math.Abs(a.Quantity-b.Quantity) <= 0.01 {
		return true
	}
	if a.Amount > 0 && b.Amount > 0 && math.Abs(a.Amount-b.Amount) <= 1.0 {
		return true
	}
	return false
}

func reconcileDayDiff(dateA, dateB string) (int, bool) {
	ta, errA := time.Parse("2006-01-02", dateA)
	tb, errB := time.Parse("2006-01-02", dateB)
	if errA != nil || errB != nil {
		return 0, false
	}
	return int(math.Round(ta.Sub(tb).Hours() / 24)), true
}

func reconcileCategoryMatches(catA, catB string) bool {
	return strings.EqualFold(strings.TrimSpace(catA), strings.TrimSpace(catB))
}

func reconcileItemsCompatible(a, b ReconcileItem) bool {
	na := strings.ToUpper(strings.TrimSpace(a.ItemName))
	nb := strings.ToUpper(strings.TrimSpace(b.ItemName))
	if na == "" || nb == "" {
		return true
	}
	return na == nb || strings.Contains(na, nb) || strings.Contains(nb, na)
}

// ReconcileLedgerVsDailyLogs compares parsed ledger items against approved DAILY_LOG transactions.
func ReconcileLedgerVsDailyLogs(ledgerItems, dailyLogs []ReconcileItem) *ReconciliationReport {
	usedLedger := make([]bool, len(ledgerItems))
	usedDaily := make([]bool, len(dailyLogs))
	report := &ReconciliationReport{
		Matches: []ReconciliationMatch{},
	}

	// Pass 1: Exact match (same date, category, and matching quantity or amount)
	for li, lItem := range ledgerItems {
		for di, dItem := range dailyLogs {
			if usedDaily[di] {
				continue
			}
			if lItem.Date == dItem.Date && reconcileCategoryMatches(lItem.Category, dItem.Category) && reconcileValuesMatch(lItem, dItem) {
				usedLedger[li] = true
				usedDaily[di] = true
				lCopy, dCopy := lItem, dItem
				report.Matches = append(report.Matches, ReconciliationMatch{
					Status:            "EXACT_MATCH",
					Color:             "green",
					DateDiffDays:      0,
					DefaultResolution: "KEEP_LEDGER",
					LedgerItem:        &lCopy,
					DailyLogItem:      &dCopy,
				})
				report.Summary.ExactMatch++
				break
			}
		}
	}

	// Pass 2: Date-shift match (+/- 2 days, same category, matching quantity or amount)
	for li, lItem := range ledgerItems {
		if usedLedger[li] {
			continue
		}
		for di, dItem := range dailyLogs {
			if usedDaily[di] {
				continue
			}
			if !reconcileCategoryMatches(lItem.Category, dItem.Category) || !reconcileValuesMatch(lItem, dItem) {
				continue
			}
			diffDays, ok := reconcileDayDiff(lItem.Date, dItem.Date)
			if ok && diffDays != 0 && math.Abs(float64(diffDays)) <= 2 {
				usedLedger[li] = true
				usedDaily[di] = true
				lCopy, dCopy := lItem, dItem
				report.Matches = append(report.Matches, ReconciliationMatch{
					Status:            "DATE_SHIFT",
					Color:             "yellow",
					DateDiffDays:      diffDays,
					DefaultResolution: "KEEP_LEDGER",
					LedgerItem:        &lCopy,
					DailyLogItem:      &dCopy,
				})
				report.Summary.DateShift++
				break
			}
		}
	}

	// Pass 3: Same-date, same-category/item mismatch (different quantity & amount)
	for li, lItem := range ledgerItems {
		if usedLedger[li] {
			continue
		}
		for di, dItem := range dailyLogs {
			if usedDaily[di] {
				continue
			}
			if lItem.Date == dItem.Date && reconcileCategoryMatches(lItem.Category, dItem.Category) && reconcileItemsCompatible(lItem, dItem) {
				usedLedger[li] = true
				usedDaily[di] = true
				lCopy, dCopy := lItem, dItem
				report.Matches = append(report.Matches, ReconciliationMatch{
					Status:            "MISMATCH",
					Color:             "red",
					DateDiffDays:      0,
					DefaultResolution: "KEEP_LEDGER",
					LedgerItem:        &lCopy,
					DailyLogItem:      &dCopy,
				})
				report.Summary.Mismatch++
				break
			}
		}
	}

	// Remaining unmatched rows
	for li, lItem := range ledgerItems {
		if usedLedger[li] {
			continue
		}
		lCopy := lItem
		report.Matches = append(report.Matches, ReconciliationMatch{
			Status:            "LEDGER_ONLY",
			Color:             "red",
			DefaultResolution: "KEEP_LEDGER",
			LedgerItem:        &lCopy,
		})
		report.Summary.LedgerOnly++
	}
	for di, dItem := range dailyLogs {
		if usedDaily[di] {
			continue
		}
		dCopy := dItem
		report.Matches = append(report.Matches, ReconciliationMatch{
			Status:            "DAILY_LOG_ONLY",
			Color:             "red",
			DefaultResolution: "KEEP_DAILY_LOG",
			DailyLogItem:      &dCopy,
		})
		report.Summary.DailyLogOnly++
	}

	return report
}

func extractReconcileItemsFromParseResult(parseResult map[string]interface{}) (int, int, []ReconcileItem) {
	toInt := func(v interface{}) int {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
		return 0
	}
	toFloat := func(v interface{}) float64 {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		}
		return 0
	}
	toStr := func(v interface{}) string {
		if s, ok := v.(string); ok {
			return s
		}
		return ""
	}

	year := toInt(parseResult["year"])
	month := toInt(parseResult["month"])
	var items []ReconcileItem

	appendSlice := func(key, category string) {
		rawList, ok := parseResult[key].([]interface{})
		if !ok {
			return
		}
		for _, entry := range rawList {
			m, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			dateStr := toStr(m["date"])
			if dateStr == "" {
				dateStr = toStr(m["transaction_date"])
			}
			if len(dateStr) > 10 {
				dateStr = dateStr[:10]
			}
			itemName := toStr(m["item_name"])
			if itemName == "" {
				itemName = toStr(m["item"])
			}
			if itemName == "" {
				itemName = toStr(m["description"])
			}
			items = append(items, ReconcileItem{
				Date:     dateStr,
				Category: category,
				ItemName: itemName,
				Quantity: toFloat(m["quantity"]),
				Amount:   toFloat(m["amount"]),
				Source:   "LEDGER",
			})
		}
	}

	appendSlice("eggs", "EGG")
	appendSlice("feeds", "FEED")
	appendSlice("medicines", "MEDICINE")
	appendSlice("other_expenses", "OTHER")
	return year, month, items
}

// BuildReconciliationForUpload loads approved DAILY_LOG transactions for the upload's month (±2 days) and reconciles them.
func BuildReconciliationForUpload(tenantID uuid.UUID, parseResult map[string]interface{}) (*ReconciliationReport, error) {
	year, month, ledgerItems := extractReconcileItemsFromParseResult(parseResult)
	if year <= 0 || month < 1 || month > 12 {
		return ReconcileLedgerVsDailyLogs(ledgerItems, nil), nil
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -2)
	endDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 2)

	rows, err := database.DB.Query(`
		SELECT id, transaction_date, COALESCE(category, ''), COALESCE(item_name, ''),
		       COALESCE(quantity, 0), COALESCE(amount, 0)
		FROM transactions
		WHERE tenant_id = $1
		  AND COALESCE(source, 'LEDGER') = 'DAILY_LOG'
		  AND transaction_date >= $2 AND transaction_date <= $3
		ORDER BY transaction_date ASC, id ASC
	`, tenantID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailyLogs []ReconcileItem
	for rows.Next() {
		var item ReconcileItem
		var txDate time.Time
		if err := rows.Scan(&item.ID, &txDate, &item.Category, &item.ItemName, &item.Quantity, &item.Amount); err != nil {
			continue
		}
		item.Date = txDate.Format("2006-01-02")
		item.Source = "DAILY_LOG"
		dailyLogs = append(dailyLogs, item)
	}

	return ReconcileLedgerVsDailyLogs(ledgerItems, dailyLogs), nil
}

// ReconcileDocumentUploadHandler returns the Ledger vs Daily Log reconciliation preview for a parsed upload.
func ReconcileDocumentUploadHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id")
		return
	}
	uploadID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid upload ID")
		return
	}

	var parseResultRaw sql.NullString
	err = database.DB.QueryRow(`
		SELECT parse_result::text
		FROM document_uploads
		WHERE id = $1 AND tenant_id = $2
	`, uploadID, tenantID).Scan(&parseResultRaw)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Upload record not found")
		return
	} else if err != nil || !parseResultRaw.Valid {
		respondWithError(w, http.StatusBadRequest, "Parsed ledger preview is not available for this upload")
		return
	}

	var parseResult map[string]interface{}
	if err := json.Unmarshal([]byte(parseResultRaw.String), &parseResult); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to decode parsed ledger preview")
		return
	}

	report, err := BuildReconciliationForUpload(tenantID, parseResult)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to run reconciliation")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    report,
	})
}

// ApplyReconciliationResolutions removes superseded DAILY_LOG transactions so charts never double-count.
func ApplyReconciliationResolutions(tenantID uuid.UUID, supersedeDailyLogIDs []int) error {
	for _, txID := range supersedeDailyLogIDs {
		if txID <= 0 {
			continue
		}
		_, _ = database.DB.Exec(`
			DELETE FROM transactions
			WHERE id = $1 AND tenant_id = $2 AND COALESCE(source, 'LEDGER') = 'DAILY_LOG'
		`, txID, tenantID)
	}
	if len(supersedeDailyLogIDs) > 0 {
		InvalidateAnalyticsCache(tenantID.String())
	}
	return nil
}
