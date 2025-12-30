package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"poultry-farm-api/utils"

	"github.com/google/uuid"
)

// GetEnhancedMonthlySummary returns detailed monthly statistics
func GetEnhancedMonthlySummary(w http.ResponseWriter, r *http.Request) {
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

	if !perms.CanViewCharts {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions to view charts")
		return
	}

	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	if yearStr == "" || monthStr == "" {
		respondWithError(w, http.StatusBadRequest, "year and month parameters required")
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid year")
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		respondWithError(w, http.StatusBadRequest, "Invalid month")
		return
	}

	// Get ledger parse data
	var ledgerParse struct {
		TotalEggs      sql.NullFloat64
		TotalFeeds     sql.NullFloat64
		TotalMedicines sql.NullFloat64
		NetProfit      sql.NullFloat64
	}

	err = database.DB.QueryRow(`
		SELECT total_eggs, total_feeds, total_medicines, net_profit
		FROM ledger_parses
		WHERE tenant_id = $1 AND year = $2 AND month = $3
	`, tenantID, year, month).Scan(
		&ledgerParse.TotalEggs, &ledgerParse.TotalFeeds,
		&ledgerParse.TotalMedicines, &ledgerParse.NetProfit,
	)

	if err == sql.ErrNoRows {
		// Get data from transactions if ledger_parse doesn't exist
		var calcTotalEggPrice, calcTotalFeedPrice, calcTotalMedicines, calcOtherExpenses, calcNetProfit float64
		var calcTotalEggsSold, calcFeedPurchasedTonne float64
		// No ledger parse - calculate from transactions
		// Get egg sales
		database.DB.QueryRow(`
			SELECT 
				COALESCE(SUM(quantity), 0),
				COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'EGG' AND transaction_type = 'SALE'
		`, tenantID, year, month).Scan(&calcTotalEggsSold, &calcTotalEggPrice)

		// Get feed purchases
		database.DB.QueryRow(`
			SELECT 
				COALESCE(SUM(quantity), 0),
				COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'FEED' AND transaction_type = 'PURCHASE'
		`, tenantID, year, month).Scan(&calcFeedPurchasedTonne, &calcTotalFeedPrice)

		// Get medicine expenses
		// Note: Include SALE as some medicine items might be incorrectly stored as SALE
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'MEDICINE' AND transaction_type IN ('PURCHASE', 'SALE')
		`, tenantID, year, month).Scan(&calcTotalMedicines)

		// Get other expenses
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'OTHER' AND transaction_type = 'EXPENSE'
		`, tenantID, year, month).Scan(&calcOtherExpenses)

		// Calculate net profit: Sales - Total Expenses
		calcNetProfit = calcTotalEggPrice - (calcTotalFeedPrice + calcTotalMedicines + calcOtherExpenses)

		// Check sensitive data
		shouldHideEggs, _ := utils.IsDataSensitive(database.DB, tenantID, "EGGS_SOLD", perms.CanViewSensitiveData)
		shouldHideFeed, _ := utils.IsDataSensitive(database.DB, tenantID, "FEED_PURCHASED", perms.CanViewSensitiveData)
		shouldHideProfit, _ := utils.IsDataSensitive(database.DB, tenantID, "NET_PROFIT", perms.CanViewSensitiveData)

		if shouldHideEggs {
			calcTotalEggPrice = 0
			calcTotalEggsSold = 0
		}
		if shouldHideFeed {
			calcTotalFeedPrice = 0
			calcFeedPurchasedTonne = 0
		}
		if shouldHideProfit {
			calcNetProfit = 0
		}

		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": []map[string]interface{}{
				{
					"year":                year,
					"month":               month,
					"total_eggs_sold":     calcTotalEggsSold,
					"total_egg_price":      calcTotalEggPrice,
					"feed_purchased_tonne": calcFeedPurchasedTonne,
					"total_feed_price":     calcTotalFeedPrice,
					"total_medicines":      calcTotalMedicines,
					"other_expenses":       calcOtherExpenses,
					"net_profit":           calcNetProfit,
					"estimated_hens":       0,
					"egg_percentage":       0,
					"egg_breakdown":        []interface{}{},
					"feed_breakdown":       []interface{}{},
				},
			},
		})
		return
	}

	// Get egg breakdowns
	type EggBreakdown struct {
		Type     string  `json:"type"`
		Quantity float64 `json:"quantity"`
	}

	eggRows, _ := database.DB.Query(`
		SELECT breakdown_type, quantity
		FROM ledger_breakdowns lb
		JOIN ledger_parses lp ON lp.id = lb.ledger_parse_id
		WHERE lp.tenant_id = $1 AND lp.year = $2 AND lp.month = $3
		  AND lb.breakdown_type LIKE 'EGG_%'
	`, tenantID, year, month)

	var eggBreakdowns []EggBreakdown
	for eggRows.Next() {
		var breakdown EggBreakdown
		var breakdownType string
		eggRows.Scan(&breakdownType, &breakdown.Quantity)
		breakdown.Type = breakdownType
		eggBreakdowns = append(eggBreakdowns, breakdown)
	}
	eggRows.Close()

	// Get feed breakdowns
	type FeedBreakdown struct {
		Type     string  `json:"type"`
		Quantity float64 `json:"quantity"`
	}

	feedRows, _ := database.DB.Query(`
		SELECT breakdown_type, quantity
		FROM ledger_breakdowns lb
		JOIN ledger_parses lp ON lp.id = lb.ledger_parse_id
		WHERE lp.tenant_id = $1 AND lp.year = $2 AND lp.month = $3
		  AND lb.breakdown_type LIKE 'FEED_%'
	`, tenantID, year, month)

	var feedBreakdowns []FeedBreakdown
	for feedRows.Next() {
		var breakdown FeedBreakdown
		var breakdownType string
		feedRows.Scan(&breakdownType, &breakdown.Quantity)
		breakdown.Type = breakdownType
		feedBreakdowns = append(feedBreakdowns, breakdown)
	}
	feedRows.Close()

	// Calculate totals from transactions
	var totalEggPrice, totalFeedPrice float64
	var totalEggsSold float64
	var feedPurchasedKg float64

	// Total egg sales
	database.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0), COALESCE(SUM(quantity), 0)
		FROM transactions
		WHERE tenant_id = $1 AND category = 'EGG' AND transaction_type = 'SALE'
		  AND EXTRACT(YEAR FROM transaction_date) = $2
		  AND EXTRACT(MONTH FROM transaction_date) = $3
	`, tenantID, year, month).Scan(&totalEggPrice, &totalEggsSold)

	// Total feed purchases
	database.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0), COALESCE(SUM(quantity), 0)
		FROM transactions
		WHERE tenant_id = $1 AND category = 'FEED' AND transaction_type = 'PURCHASE'
		  AND EXTRACT(YEAR FROM transaction_date) = $2
		  AND EXTRACT(MONTH FROM transaction_date) = $3
	`, tenantID, year, month).Scan(&totalFeedPrice, &feedPurchasedKg)

	feedPurchasedTonne := feedPurchasedKg / 1000.0

	// Calculate days in month
	daysInMonth := 31 // Default, will calculate properly
	if month == 2 {
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			daysInMonth = 29
		} else {
			daysInMonth = 28
		}
	} else if month == 4 || month == 6 || month == 9 || month == 11 {
		daysInMonth = 30
	}

	// Estimate hens: 10,000 hens consume 1 tonne per day
	estimatedHens := 0.0
	if feedPurchasedTonne > 0 && daysInMonth > 0 {
		estimatedHens = (feedPurchasedTonne / float64(daysInMonth)) * 10000.0
	}

	// Egg percentage: (Total Eggs / Estimated Hens / daysInMonth) * 100
	eggPercentage := 0.0
	if estimatedHens > 0 && daysInMonth > 0 {
		dailyEggs := totalEggsSold / float64(daysInMonth)
		eggPercentage = (dailyEggs / estimatedHens) * 100.0
	}

	// Get net profit
	netProfit := 0.0
	if ledgerParse.NetProfit.Valid {
		netProfit = ledgerParse.NetProfit.Float64
	}

	// Check sensitive data
	shouldHideEggs, _ := utils.IsDataSensitive(database.DB, tenantID, "EGGS_SOLD", perms.CanViewSensitiveData)
	shouldHideFeed, _ := utils.IsDataSensitive(database.DB, tenantID, "FEED_PURCHASED", perms.CanViewSensitiveData)
	shouldHideProfit, _ := utils.IsDataSensitive(database.DB, tenantID, "NET_PROFIT", perms.CanViewSensitiveData)

	if shouldHideEggs {
		totalEggPrice = 0
	}
	if shouldHideFeed {
		totalFeedPrice = 0
	}
	if shouldHideProfit {
		netProfit = 0
	}

	// Get medicine and other expenses from transactions for this month
	var medicineExpenses, otherExpensesFromTxns float64
	database.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND category = 'MEDICINE' AND transaction_type IN ('PURCHASE', 'SALE')
	`, tenantID, year, month).Scan(&medicineExpenses)

	database.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND category = 'OTHER' AND transaction_type = 'EXPENSE'
	`, tenantID, year, month).Scan(&otherExpensesFromTxns)

	// Use ledger parse medicines if available, otherwise use transactions
	var finalMedicines float64
	if ledgerParse.TotalMedicines.Valid && ledgerParse.TotalMedicines.Float64 > 0 {
		finalMedicines = ledgerParse.TotalMedicines.Float64
	} else {
		finalMedicines = medicineExpenses
	}

	// Recalculate net profit: Sales - Total Expenses (more accurate)
	calculatedNetProfit := totalEggPrice - (totalFeedPrice + finalMedicines + otherExpensesFromTxns)

	// Use calculated profit if ledger profit is 0, invalid, or significantly different
	if !ledgerParse.NetProfit.Valid || ledgerParse.NetProfit.Float64 == 0 {
		netProfit = calculatedNetProfit
	} else {
		// Use ledger profit if it exists, but verify it's reasonable
		netProfit = ledgerParse.NetProfit.Float64
	}

	summary := map[string]interface{}{
		"year":                year,
		"month":               month,
		"total_eggs_sold":     totalEggsSold,
		"egg_breakdown":       eggBreakdowns,
		"total_egg_price":     totalEggPrice,
		"feed_purchased_tonne": feedPurchasedTonne,
		"feed_breakdown":      feedBreakdowns,
		"total_feed_price":    totalFeedPrice,
		"total_medicines":     finalMedicines,
		"other_expenses":      otherExpensesFromTxns,
		"net_profit":          netProfit,
		"estimated_hens":     estimatedHens,
		"egg_percentage":      eggPercentage,
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    []map[string]interface{}{summary},
	})
}

// GetAllYearsSummary returns yearly summaries
func GetAllYearsSummary(w http.ResponseWriter, r *http.Request) {
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

	if !perms.CanViewCharts {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions to view charts")
		return
	}

	// Get all years with data
	rows, err := database.DB.Query(`
		SELECT DISTINCT year
		FROM ledger_parses
		WHERE tenant_id = $1
		ORDER BY year DESC
	`, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch years")
		return
	}
	defer rows.Close()

	type YearlySummary struct {
		Year            int     `json:"year"`
		TotalEggPrice   float64 `json:"total_egg_price"`
		TotalFeedPrice  float64 `json:"total_feed_price"`
		NetProfit       float64 `json:"net_profit"`
	}

		var summaries []YearlySummary
	currentYear := time.Now().Year()
	currentMonth := int(time.Now().Month())
	
	for rows.Next() {
		var year int
		rows.Scan(&year)

		var totalEggPrice, totalFeedPrice, netProfit float64
		var query string

		// For current year, only include data up to current month
		if year == currentYear {
			query = `
				SELECT 
					COALESCE(SUM(CASE WHEN category = 'EGG' AND transaction_type = 'SALE' THEN amount ELSE 0 END), 0),
					COALESCE(SUM(CASE WHEN category = 'FEED' AND transaction_type = 'PURCHASE' THEN amount ELSE 0 END), 0),
					COALESCE(SUM(CASE 
						WHEN transaction_type IN ('SALE', 'DISCOUNT', 'INCOME') THEN amount
						WHEN transaction_type IN ('PURCHASE', 'TDS', 'EXPENSE') THEN -amount
						ELSE 0
					END), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND EXTRACT(MONTH FROM transaction_date) <= $3
			`
			database.DB.QueryRow(query, tenantID, year, currentMonth).Scan(&totalEggPrice, &totalFeedPrice, &netProfit)
		} else {
			// For past years, include all months
			query = `
				SELECT 
					COALESCE(SUM(CASE WHEN category = 'EGG' AND transaction_type = 'SALE' THEN amount ELSE 0 END), 0),
					COALESCE(SUM(CASE WHEN category = 'FEED' AND transaction_type = 'PURCHASE' THEN amount ELSE 0 END), 0),
					COALESCE(SUM(CASE 
						WHEN transaction_type IN ('SALE', 'DISCOUNT', 'INCOME') THEN amount
						WHEN transaction_type IN ('PURCHASE', 'TDS', 'EXPENSE') THEN -amount
						ELSE 0
					END), 0)
				FROM transactions
				WHERE tenant_id = $1 AND EXTRACT(YEAR FROM transaction_date) = $2
			`
			database.DB.QueryRow(query, tenantID, year).Scan(&totalEggPrice, &totalFeedPrice, &netProfit)
		}

		// Check sensitive data
		shouldHideEggs, _ := utils.IsDataSensitive(database.DB, tenantID, "EGGS_SOLD", perms.CanViewSensitiveData)
		shouldHideFeed, _ := utils.IsDataSensitive(database.DB, tenantID, "FEED_PURCHASED", perms.CanViewSensitiveData)
		shouldHideProfit, _ := utils.IsDataSensitive(database.DB, tenantID, "NET_PROFIT", perms.CanViewSensitiveData)

		if shouldHideEggs {
			totalEggPrice = 0
		}
		if shouldHideFeed {
			totalFeedPrice = 0
		}
		if shouldHideProfit {
			netProfit = 0
		}

		summaries = append(summaries, YearlySummary{
			Year:           year,
			TotalEggPrice:  totalEggPrice,
			TotalFeedPrice: totalFeedPrice,
			NetProfit:      netProfit,
		})
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    summaries,
	})
}

// GetLast12MonthsSummary returns monthly summaries for the last 12 months
func GetLast12MonthsSummary(w http.ResponseWriter, r *http.Request) {
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

	if !perms.CanViewCharts {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions to view charts")
		return
	}

	type MonthlyData struct {
		Year            int     `json:"year"`
		Month           int     `json:"month"`
		MonthName       string  `json:"month_name"`
		Sales           float64 `json:"sales"`
		FeedExpense     float64 `json:"feed_expense"`
		MedicineExpense float64 `json:"medicine_expense"`
		LaborExpense    float64 `json:"labor_expense"`
		OtherExpense    float64 `json:"other_expense"`
		TotalExpense    float64 `json:"total_expense"`
		NetProfit       float64 `json:"net_profit"`
	}

	var monthlyData []MonthlyData

	// Get last 12 months
	for i := 11; i >= 0; i-- {
		date := time.Now().AddDate(0, -i, 0)
		year := date.Year()
		month := int(date.Month())

		var sales, feedExpense, medicineExpense, laborExpense, otherExpense, netProfit float64

		// Get sales (egg sales)
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'EGG' AND transaction_type = 'SALE'
		`, tenantID, year, month).Scan(&sales)

		// Get medicine expense first (from MEDICINE category)
		// Note: Medicine items might be stored as SALE (incorrectly) or PURCHASE/EXPENSE (correctly)
		var medicineFromMedicineCategory float64
		err = database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'MEDICINE' AND transaction_type IN ('PURCHASE', 'SALE')
		`, tenantID, year, month).Scan(&medicineFromMedicineCategory)
		if err != nil {
			log.Printf("Error getting medicine from MEDICINE category for %d-%02d: %v", year, month, err)
		} else {
			// Debug logging
			if medicineFromMedicineCategory > 0 {
				log.Printf("Found medicine expense for %d-%02d: %.2f", year, month, medicineFromMedicineCategory)
			}
		}

		// Get medicine expenses that might be incorrectly categorized as FEED
		// Check for common medicine item names (expanded list with more variations)
		// Note: Include SALE as some medicine items might be incorrectly stored as SALE
		var medicineFromFeedCategory float64
		err = database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'FEED' 
				AND transaction_type IN ('PURCHASE', 'SALE')
				AND (
					UPPER(item_name) LIKE '%D3%' OR
					UPPER(item_name) LIKE '%VETMULIN%' OR
					UPPER(item_name) LIKE '%OXYCYCLINE%' OR
					UPPER(item_name) LIKE '%TIAZIN%' OR
					UPPER(item_name) LIKE '%BPPS%' OR
					UPPER(item_name) LIKE '%CTC%' OR
					UPPER(item_name) LIKE '%SHELL GRIT%' OR
					UPPER(item_name) LIKE '%ROVIMIX%' OR
					UPPER(item_name) LIKE '%CHOLIMARIN%' OR
					UPPER(item_name) LIKE '%ZAGROMIN%' OR
					UPPER(item_name) LIKE '%G PRO NATURO%' OR
					UPPER(item_name) LIKE '%NECROVET%' OR
					UPPER(item_name) LIKE '%TOXOL%' OR
					UPPER(item_name) LIKE '%FRA C12%' OR
					UPPER(item_name) LIKE '%FRA C 12%' OR
					UPPER(item_name) LIKE '%CALCI%' OR
					UPPER(item_name) LIKE '%CALDLIV%' OR
					UPPER(item_name) LIKE '%RESPAFEED%' OR
					UPPER(item_name) LIKE '%VENTRIM%' OR
					UPPER(item_name) LIKE '%VITAL%' OR
					UPPER(item_name) LIKE '%MEDICINE%' OR
					UPPER(item_name) LIKE '%MEDIC%' OR
					UPPER(item_name) LIKE '%VITAMIN%' OR
					UPPER(item_name) LIKE '%SUPPLEMENT%' OR
					UPPER(item_name) LIKE '%GRIT%' OR
					UPPER(item_name) LIKE '%VET%' OR
					UPPER(item_name) LIKE '%NECRO%' OR
					UPPER(item_name) LIKE '%TOX%'
				)
		`, tenantID, year, month).Scan(&medicineFromFeedCategory)
		if err != nil {
			log.Printf("Error getting medicine from FEED category for %d-%02d: %v", year, month, err)
		}

		// Total medicine expense = medicine from MEDICINE category + medicine from FEED category
		medicineExpense = medicineFromMedicineCategory + medicineFromFeedCategory
		
		// Debug logging
		if medicineExpense > 0 {
			log.Printf("Total medicine expense for %d-%02d: %.2f (from MEDICINE: %.2f, from FEED: %.2f)", 
				year, month, medicineExpense, medicineFromMedicineCategory, medicineFromFeedCategory)
		}

		// Get feed expense (excluding medicine items)
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'FEED' 
				AND transaction_type = 'PURCHASE'
				AND NOT (
					UPPER(item_name) LIKE '%D3%' OR
					UPPER(item_name) LIKE '%VETMULIN%' OR
					UPPER(item_name) LIKE '%OXYCYCLINE%' OR
					UPPER(item_name) LIKE '%TIAZIN%' OR
					UPPER(item_name) LIKE '%BPPS%' OR
					UPPER(item_name) LIKE '%CTC%' OR
					UPPER(item_name) LIKE '%SHELL GRIT%' OR
					UPPER(item_name) LIKE '%ROVIMIX%' OR
					UPPER(item_name) LIKE '%CHOLIMARIN%' OR
					UPPER(item_name) LIKE '%ZAGROMIN%' OR
					UPPER(item_name) LIKE '%G PRO NATURO%' OR
					UPPER(item_name) LIKE '%NECROVET%' OR
					UPPER(item_name) LIKE '%TOXOL%' OR
					UPPER(item_name) LIKE '%FRA C12%' OR
					UPPER(item_name) LIKE '%FRA C 12%' OR
					UPPER(item_name) LIKE '%CALCI%' OR
					UPPER(item_name) LIKE '%CALDLIV%' OR
					UPPER(item_name) LIKE '%RESPAFEED%' OR
					UPPER(item_name) LIKE '%VENTRIM%' OR
					UPPER(item_name) LIKE '%VITAL%' OR
					UPPER(item_name) LIKE '%MEDICINE%' OR
					UPPER(item_name) LIKE '%MEDIC%' OR
					UPPER(item_name) LIKE '%VITAMIN%' OR
					UPPER(item_name) LIKE '%SUPPLEMENT%' OR
					UPPER(item_name) LIKE '%GRIT%' OR
					UPPER(item_name) LIKE '%VET%' OR
					UPPER(item_name) LIKE '%NECRO%' OR
					UPPER(item_name) LIKE '%TOX%'
				)
		`, tenantID, year, month).Scan(&feedExpense)

		// Get labor expense (EMPLOYEE category)
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'EMPLOYEE' AND transaction_type = 'EXPENSE'
		`, tenantID, year, month).Scan(&laborExpense)

		// Get other expenses (OTHER category, EXPENSE type)
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'OTHER' AND transaction_type = 'EXPENSE'
		`, tenantID, year, month).Scan(&otherExpense)

		// Calculate net profit: Sales - Total Expenses
		netProfit = sales - (feedExpense + medicineExpense + laborExpense + otherExpense)

		// Check sensitive data
		shouldHideEggs, _ := utils.IsDataSensitive(database.DB, tenantID, "EGGS_SOLD", perms.CanViewSensitiveData)
		shouldHideFeed, _ := utils.IsDataSensitive(database.DB, tenantID, "FEED_PURCHASED", perms.CanViewSensitiveData)
		shouldHideProfit, _ := utils.IsDataSensitive(database.DB, tenantID, "NET_PROFIT", perms.CanViewSensitiveData)

		if shouldHideEggs {
			sales = 0
		}
		if shouldHideFeed {
			feedExpense = 0
		}
		if shouldHideProfit {
			netProfit = 0
		}

		totalExpense := feedExpense + medicineExpense + laborExpense + otherExpense

		monthlyData = append(monthlyData, MonthlyData{
			Year:            year,
			Month:           month,
			MonthName:       date.Format("Jan"),
			Sales:           sales,
			FeedExpense:     feedExpense,
			MedicineExpense: medicineExpense,
			LaborExpense:    laborExpense,
			OtherExpense:    otherExpense,
			TotalExpense:    totalExpense,
			NetProfit:       netProfit,
		})
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    monthlyData,
	})
}

