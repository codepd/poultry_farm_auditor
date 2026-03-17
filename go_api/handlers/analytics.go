package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"net/http"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"poultry-farm-api/utils"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func sanitizeFloat(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

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
		var calcTotalEggPrice, calcTotalFeedPrice, calcTotalMedicines, calcOtherExpenses float64
		var calcTotalDiscounts, calcTotalTDS, calcNetProfit float64
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

		// Get other expenses (excluding labor expenses)
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'OTHER' 
				AND transaction_type = 'EXPENSE'
				AND NOT (UPPER(item_name) LIKE '%LABOR%')
		`, tenantID, year, month).Scan(&calcOtherExpenses)

		// Get discounts (income - add to sales)
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND transaction_type = 'DISCOUNT'
		`, tenantID, year, month).Scan(&calcTotalDiscounts)

		// Get TDS (expense - subtract from sales)
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND transaction_type = 'TDS'
		`, tenantID, year, month).Scan(&calcTotalTDS)

		// Calculate net profit: Egg sales + discounts - (feed + medicine + other + TDS)
		calcNetProfit = (calcTotalEggPrice + calcTotalDiscounts) - (calcTotalFeedPrice + calcTotalMedicines + calcOtherExpenses + calcTotalTDS)

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
					"year":                 year,
					"month":                month,
					"total_eggs_sold":      calcTotalEggsSold,
					"total_egg_price":      calcTotalEggPrice,
					"feed_purchased_tonne": calcFeedPurchasedTonne,
					"total_feed_price":     calcTotalFeedPrice,
					"total_medicines":      calcTotalMedicines,
					"other_expenses":       calcOtherExpenses,
					"total_discounts":      calcTotalDiscounts,
					"total_tds":            calcTotalTDS,
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
	totalEggPrice = sanitizeFloat(totalEggPrice)
	totalEggsSold = sanitizeFloat(totalEggsSold)

	// Total feed purchases
	database.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0), COALESCE(SUM(quantity), 0)
		FROM transactions
		WHERE tenant_id = $1 AND category = 'FEED' AND transaction_type = 'PURCHASE'
		  AND EXTRACT(YEAR FROM transaction_date) = $2
		  AND EXTRACT(MONTH FROM transaction_date) = $3
	`, tenantID, year, month).Scan(&totalFeedPrice, &feedPurchasedKg)
	totalFeedPrice = sanitizeFloat(totalFeedPrice)
	feedPurchasedKg = sanitizeFloat(feedPurchasedKg)

	feedPurchasedTonne := sanitizeFloat(feedPurchasedKg / 1000.0)

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

	// Calculate actual head count based on daily counts for the month
	// Strategy: For each day in the month, calculate head count by:
	// 1. Calculate when each batch started laying based on their age at date_added
	// 2. If hens were already at laying age when added in this month, count them for entire month
	// 3. For each day, count hens that were at laying age on that day
	// 4. Start with initial_count and subtract all mortalities up to that day
	// 5. Sum all daily head counts and divide by days in month to get average
	var actualHeadCount float64 // Average daily head count (for display and calculation)
	usingActualCount := false

	// Laying typically starts around 18-20 weeks (126-140 days)
	const layingAgeDays = 126

	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0) // Start of next month

	// Get all batches that existed at any point during this month or could have been laying
	// Include batches added before or during this month
	batchRows, err := database.DB.Query(`
		SELECT id, date_added, initial_count, age_weeks, age_days
		FROM hen_batches
		WHERE tenant_id = $1
		  AND date_added <= $2
		ORDER BY date_added
	`, tenantID, monthEnd)

	if err == nil {
		defer batchRows.Close()

		// Store batch information including age
		type BatchInfo struct {
			ID           int
			DateAdded    time.Time
			InitialCount int
			AgeWeeks     int
			AgeDays      int
			LayingStart  time.Time // Calculated date when they started laying
		}
		var batches []BatchInfo

		for batchRows.Next() {
			var batch BatchInfo
			var ageWeeks, ageDays sql.NullInt64
			if err := batchRows.Scan(&batch.ID, &batch.DateAdded, &batch.InitialCount, &ageWeeks, &ageDays); err != nil {
				continue
			}

			// Get age in total days
			totalAgeDays := 0
			if ageWeeks.Valid {
				batch.AgeWeeks = int(ageWeeks.Int64)
				totalAgeDays = int(ageWeeks.Int64) * 7
			}
			if ageDays.Valid {
				batch.AgeDays = int(ageDays.Int64)
				totalAgeDays += int(ageDays.Int64)
			}

			// Calculate when they actually started laying based on their age at date_added
			// If they're already at laying age when added, calculate backwards when they started laying
			// Otherwise, calculate when they will reach laying age in the future
			if totalAgeDays >= layingAgeDays {
				// Already at laying age when added, so calculate when they started laying
				// laying_start = date_added - (current_age - laying_age)
				daysAgoStartedLaying := totalAgeDays - layingAgeDays
				batch.LayingStart = batch.DateAdded.AddDate(0, 0, -daysAgoStartedLaying)
			} else if totalAgeDays > 0 {
				// Not yet at laying age, calculate when they will reach laying age
				daysUntilLaying := layingAgeDays - totalAgeDays
				batch.LayingStart = batch.DateAdded.AddDate(0, 0, daysUntilLaying)
			} else {
				// Age is 0 or NULL - assume they will reach laying age in the future
				// Set laying_start far in the future so they won't be counted until age is updated
				batch.LayingStart = batch.DateAdded.AddDate(0, 0, layingAgeDays)
			}

			batches = append(batches, batch)
		}

		if len(batches) > 0 {
			// Get all mortality events for these batches up to month end
			batchIDs := make([]interface{}, len(batches))
			for i, b := range batches {
				batchIDs[i] = b.ID
			}

			// Build query for mortality events
			placeholders := ""
			for i := range batchIDs {
				if i > 0 {
					placeholders += ","
				}
				placeholders += fmt.Sprintf("$%d", i+1)
			}

			mortalityQuery := fmt.Sprintf(`
				SELECT batch_id, mortality_date, count
				FROM hen_mortality
				WHERE batch_id IN (%s)
				  AND mortality_date < $%d
				ORDER BY batch_id, mortality_date
			`, placeholders, len(batchIDs)+1)

			mortalityArgs := make([]interface{}, len(batchIDs)+1)
			copy(mortalityArgs, batchIDs)
			mortalityArgs[len(batchIDs)] = monthEnd

			mortalityRows, err := database.DB.Query(mortalityQuery, mortalityArgs...)
			if err == nil {
				defer mortalityRows.Close()

				// Store mortality by batch_id and date
				// Key: batch_id, Value: map of date -> count
				mortalityByBatch := make(map[int]map[time.Time]int)

				for mortalityRows.Next() {
					var batchID int
					var mortalityDate time.Time
					var count int
					if err := mortalityRows.Scan(&batchID, &mortalityDate, &count); err != nil {
						continue
					}

					if mortalityByBatch[batchID] == nil {
						mortalityByBatch[batchID] = make(map[time.Time]int)
					}
					mortalityByBatch[batchID][mortalityDate] += count
				}

				// Calculate total head count days (sum of head count for each day)
				var totalHeadCountDays float64

				// For each day in the month
				for day := 0; day < daysInMonth; day++ {
					currentDay := monthStart.AddDate(0, 0, day)
					dailyHeadCount := 0

					// For each batch, calculate head count on this day
					for _, batch := range batches {
						// Check if batch was added in this month and was already at laying age
						addedInThisMonth := !batch.DateAdded.Before(monthStart) && batch.DateAdded.Before(monthEnd)
						wasAlreadyLayingWhenAdded := !batch.LayingStart.After(batch.DateAdded)

						var shouldCount bool
						var countFromDate time.Time

						if addedInThisMonth && wasAlreadyLayingWhenAdded {
							// Batch was added in this month and was already at laying age
							// Count for ENTIRE month (all 31 days), not just from date_added
							countFromDate = monthStart
							shouldCount = !currentDay.Before(countFromDate)
						} else {
							// Batch was added before this month or will reach laying age in the future
							// Only count if hens were at laying age on this day
							if batch.LayingStart.After(currentDay) {
								continue
							}

							countFromDate = batch.LayingStart
							if batch.LayingStart.Before(monthStart) {
								countFromDate = monthStart
							}

							shouldCount = !currentDay.Before(countFromDate)
						}

						if !shouldCount {
							continue
						}

						// Start with initial count
						batchCountOnDay := batch.InitialCount

						// Subtract all mortalities up to and including this day
						if mortalities, exists := mortalityByBatch[batch.ID]; exists {
							for mortDate, mortCount := range mortalities {
								if !mortDate.After(currentDay) {
									batchCountOnDay -= mortCount
								}
							}
						}

						// Only add if count is positive (avoid negative counts from data issues)
						if batchCountOnDay > 0 {
							dailyHeadCount += batchCountOnDay
						}
					}

					totalHeadCountDays += float64(dailyHeadCount)
				}

				// Calculate average daily head count
				if daysInMonth > 0 {
					actualHeadCount = totalHeadCountDays / float64(daysInMonth)
					usingActualCount = actualHeadCount > 0
				}
			}
		}
	}

	// Use average daily head count for egg percentage calculation (more accurate)
	estimatedHens := 0.0

	if usingActualCount && actualHeadCount > 0 {
		// Use average daily head count for egg percentage calculation
		estimatedHens = actualHeadCount
	} else if feedPurchasedTonne > 0 && daysInMonth > 0 {
		// Fallback: Estimate hens: 10,000 hens consume 1 tonne per day
		estimatedHens = (feedPurchasedTonne / float64(daysInMonth)) * 10000.0
	}

	// Egg percentage: (Total Eggs / Head Count / daysInMonth) * 100
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
	var totalDiscounts, totalTDS float64
	database.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND category = 'MEDICINE' AND transaction_type IN ('PURCHASE', 'SALE')
	`, tenantID, year, month).Scan(&medicineExpenses)
	medicineExpenses = sanitizeFloat(medicineExpenses)

	database.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND category = 'OTHER' AND transaction_type = 'EXPENSE'
	`, tenantID, year, month).Scan(&otherExpensesFromTxns)
	otherExpensesFromTxns = sanitizeFloat(otherExpensesFromTxns)

	database.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND transaction_type = 'DISCOUNT'
	`, tenantID, year, month).Scan(&totalDiscounts)
	totalDiscounts = sanitizeFloat(totalDiscounts)

	database.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND transaction_type = 'TDS'
	`, tenantID, year, month).Scan(&totalTDS)
	totalTDS = sanitizeFloat(totalTDS)

	// Get payments received (PAYMENT transaction type)
	var totalPayments float64
	database.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND transaction_type = 'PAYMENT'
	`, tenantID, year, month).Scan(&totalPayments)
	totalPayments = sanitizeFloat(totalPayments)

	// Get payment breakdown from transactions
	type PaymentBreakdownFromTxns struct {
		ItemName string
		Amount   float64
	}
	paymentBreakdownRows, _ := database.DB.Query(`
		SELECT COALESCE(item_name, 'Cash Payment') as item_name, 
		       COALESCE(SUM(amount), 0) as total_amt
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND transaction_type = 'PAYMENT'
		GROUP BY COALESCE(item_name, 'Cash Payment')
		ORDER BY COALESCE(item_name, 'Cash Payment')
	`, tenantID, year, month)

	var paymentBreakdownsFromTxns []PaymentBreakdownFromTxns
	for paymentBreakdownRows.Next() {
		var breakdown PaymentBreakdownFromTxns
		paymentBreakdownRows.Scan(&breakdown.ItemName, &breakdown.Amount)
		paymentBreakdownsFromTxns = append(paymentBreakdownsFromTxns, breakdown)
	}
	paymentBreakdownRows.Close()

	// Get egg breakdown from transactions (by item_name)
	type EggBreakdownFromTxns struct {
		ItemName string
		Quantity float64
		Amount   float64
	}
	eggBreakdownRows, _ := database.DB.Query(`
		SELECT item_name, COALESCE(SUM(quantity), 0) as total_qty, COALESCE(SUM(amount), 0) as total_amt
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND category = 'EGG' AND transaction_type = 'SALE'
		GROUP BY item_name
		ORDER BY item_name
	`, tenantID, year, month)

	var eggBreakdownsFromTxns []EggBreakdownFromTxns
	for eggBreakdownRows.Next() {
		var breakdown EggBreakdownFromTxns
		eggBreakdownRows.Scan(&breakdown.ItemName, &breakdown.Quantity, &breakdown.Amount)
		breakdown.Quantity = sanitizeFloat(breakdown.Quantity)
		breakdown.Amount = sanitizeFloat(breakdown.Amount)
		eggBreakdownsFromTxns = append(eggBreakdownsFromTxns, breakdown)
	}
	eggBreakdownRows.Close()

	// Get feed breakdown from transactions (by item_name)
	type FeedBreakdownFromTxns struct {
		ItemName string
		Quantity float64
		Amount   float64
	}
	feedBreakdownRows, _ := database.DB.Query(`
		SELECT item_name, COALESCE(SUM(quantity), 0) as total_qty, COALESCE(SUM(amount), 0) as total_amt
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND category = 'FEED' AND transaction_type = 'PURCHASE'
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
		GROUP BY item_name
		ORDER BY item_name
	`, tenantID, year, month)

	var feedBreakdownsFromTxns []FeedBreakdownFromTxns
	for feedBreakdownRows.Next() {
		var breakdown FeedBreakdownFromTxns
		feedBreakdownRows.Scan(&breakdown.ItemName, &breakdown.Quantity, &breakdown.Amount)
		breakdown.Quantity = sanitizeFloat(breakdown.Quantity)
		breakdown.Amount = sanitizeFloat(breakdown.Amount)
		feedBreakdownsFromTxns = append(feedBreakdownsFromTxns, breakdown)
	}
	feedBreakdownRows.Close()

	// Get medicine breakdown from transactions (by item_name)
	type MedicineBreakdownFromTxns struct {
		ItemName string
		Quantity float64
		Amount   float64
	}
	medicineBreakdownRows, _ := database.DB.Query(`
		SELECT item_name, COALESCE(SUM(quantity), 0) as total_qty, COALESCE(SUM(amount), 0) as total_amt
		FROM transactions
		WHERE tenant_id = $1 
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3
			AND category = 'MEDICINE' AND transaction_type IN ('PURCHASE', 'SALE')
		GROUP BY item_name
		ORDER BY item_name
	`, tenantID, year, month)

	var medicineBreakdownsFromTxns []MedicineBreakdownFromTxns
	for medicineBreakdownRows.Next() {
		var breakdown MedicineBreakdownFromTxns
		medicineBreakdownRows.Scan(&breakdown.ItemName, &breakdown.Quantity, &breakdown.Amount)
		breakdown.Quantity = sanitizeFloat(breakdown.Quantity)
		breakdown.Amount = sanitizeFloat(breakdown.Amount)
		medicineBreakdownsFromTxns = append(medicineBreakdownsFromTxns, breakdown)
	}
	medicineBreakdownRows.Close()

	// Use ledger parse medicines if available, otherwise use transactions
	var finalMedicines float64
	if ledgerParse.TotalMedicines.Valid && ledgerParse.TotalMedicines.Float64 > 0 {
		finalMedicines = sanitizeFloat(ledgerParse.TotalMedicines.Float64)
	} else {
		finalMedicines = sanitizeFloat(medicineExpenses)
	}

	// Recalculate net profit: Egg sales + discounts - (feed + medicine + other + TDS)
	calculatedNetProfit := sanitizeFloat((totalEggPrice + totalDiscounts) - (totalFeedPrice + finalMedicines + otherExpensesFromTxns + totalTDS))

	// Use calculated profit if ledger profit is 0, invalid, or significantly different
	if !ledgerParse.NetProfit.Valid || ledgerParse.NetProfit.Float64 == 0 {
		netProfit = calculatedNetProfit
	} else {
		// Use ledger profit if it exists, but verify it's reasonable
		netProfit = sanitizeFloat(ledgerParse.NetProfit.Float64)
	}

	// Convert transaction breakdowns to the format expected by frontend
	var eggBreakdownFinal []map[string]interface{}
	// Use transaction breakdowns if available, otherwise use ledger breakdowns
	if len(eggBreakdownsFromTxns) > 0 {
		for _, item := range eggBreakdownsFromTxns {
			eggBreakdownFinal = append(eggBreakdownFinal, map[string]interface{}{
				"type":     item.ItemName,
				"quantity": item.Quantity,
				"amount":   item.Amount,
			})
		}
	} else {
		// Fallback to ledger breakdowns
		for _, item := range eggBreakdowns {
			eggBreakdownFinal = append(eggBreakdownFinal, map[string]interface{}{
				"type":     item.Type,
				"quantity": item.Quantity,
			})
		}
	}

	var feedBreakdownFinal []map[string]interface{}
	if len(feedBreakdownsFromTxns) > 0 {
		for _, item := range feedBreakdownsFromTxns {
			feedBreakdownFinal = append(feedBreakdownFinal, map[string]interface{}{
				"type":     item.ItemName,
				"quantity": item.Quantity / 1000.0, // Convert to tonnes
				"amount":   item.Amount,
			})
		}
	} else {
		// Fallback to ledger breakdowns
		for _, item := range feedBreakdowns {
			feedBreakdownFinal = append(feedBreakdownFinal, map[string]interface{}{
				"type":     item.Type,
				"quantity": item.Quantity,
			})
		}
	}

	var medicineBreakdownFinal []map[string]interface{}
	for _, item := range medicineBreakdownsFromTxns {
		medicineBreakdownFinal = append(medicineBreakdownFinal, map[string]interface{}{
			"type":     item.ItemName,
			"quantity": item.Quantity,
			"amount":   item.Amount,
		})
	}

	var paymentBreakdownFinal []map[string]interface{}
	for _, item := range paymentBreakdownsFromTxns {
		paymentBreakdownFinal = append(paymentBreakdownFinal, map[string]interface{}{
			"type":   item.ItemName,
			"amount": item.Amount,
		})
	}

	summary := map[string]interface{}{
		"year":                 year,
		"month":                month,
		"total_eggs_sold":      totalEggsSold,
		"egg_breakdown":        eggBreakdownFinal,
		"total_egg_price":      totalEggPrice,
		"feed_purchased_tonne": feedPurchasedTonne,
		"feed_breakdown":       feedBreakdownFinal,
		"total_feed_price":     totalFeedPrice,
		"total_medicines":      finalMedicines,
		"medicine_breakdown":   medicineBreakdownFinal,
		"other_expenses":       otherExpensesFromTxns,
		"total_discounts":      totalDiscounts,
		"total_tds":            totalTDS,
		"total_payments":       totalPayments,
		"payment_breakdown":    paymentBreakdownFinal,
		"net_profit":           netProfit,
		"estimated_hens":       estimatedHens,
		"actual_head_count":    actualHeadCount,
		"using_actual_count":   usingActualCount,
		"egg_percentage":       eggPercentage,
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

	// Get all years with data from transactions (more reliable than ledger_parses)
	rows, err := database.DB.Query(`
		SELECT DISTINCT EXTRACT(YEAR FROM transaction_date)::int as year
		FROM transactions
		WHERE tenant_id = $1
		ORDER BY year DESC
	`, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch years")
		return
	}
	defer rows.Close()

	type YearlySummary struct {
		Year         int     `json:"year"`
		TotalSales   float64 `json:"total_sales"`
		TotalExpense float64 `json:"total_expense"`
		NetProfit    float64 `json:"net_profit"`
	}

	var summaries []YearlySummary
	currentYear := time.Now().Year()
	currentMonth := int(time.Now().Month())

	for rows.Next() {
		var year int
		rows.Scan(&year)

		var totalSales, totalExpense, netProfit float64

		// Calculate totals from transactions
		var totalFeed, totalMedicine, totalLabor, totalOther, totalDiscounts, totalTDS float64

		// For current year, only include data up to current month
		if year == currentYear {
			// Get egg sales (total sales)
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND EXTRACT(MONTH FROM transaction_date) <= $3
					AND category = 'EGG' AND transaction_type = 'SALE'
			`, tenantID, year, currentMonth).Scan(&totalSales)

			// Get feed purchases (excluding medicine items)
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND EXTRACT(MONTH FROM transaction_date) <= $3
					AND category = 'FEED' AND transaction_type = 'PURCHASE'
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
			`, tenantID, year, currentMonth).Scan(&totalFeed)

			// Get medicine expenses (from MEDICINE category)
			var medicineFromMedicineCategory float64
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND EXTRACT(MONTH FROM transaction_date) <= $3
					AND category = 'MEDICINE' AND transaction_type IN ('PURCHASE', 'SALE')
			`, tenantID, year, currentMonth).Scan(&medicineFromMedicineCategory)

			// Get medicine expenses that might be incorrectly categorized as FEED
			var medicineFromFeedCategory float64
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND EXTRACT(MONTH FROM transaction_date) <= $3
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
			`, tenantID, year, currentMonth).Scan(&medicineFromFeedCategory)

			// Total medicine expense = medicine from MEDICINE category + medicine from FEED category
			totalMedicine = medicineFromMedicineCategory + medicineFromFeedCategory

			// Get labor expenses (OTHER category with labor-related item_name)
			var laborFromExpense float64
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND EXTRACT(MONTH FROM transaction_date) <= $3
					AND transaction_type = 'EXPENSE'
					AND category = 'OTHER' 
					AND UPPER(item_name) LIKE '%LABOR%'
			`, tenantID, year, currentMonth).Scan(&laborFromExpense)
			totalLabor = laborFromExpense

			// Get other expenses (includes electricity and other miscellaneous expenses, excluding labor)
			var otherFromExpense float64
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND EXTRACT(MONTH FROM transaction_date) <= $3
					AND category = 'OTHER' 
					AND transaction_type = 'EXPENSE'
					AND NOT (UPPER(item_name) LIKE '%LABOR%')
			`, tenantID, year, currentMonth).Scan(&otherFromExpense)
			totalOther = otherFromExpense

			// Get discounts (income - add to sales)
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND EXTRACT(MONTH FROM transaction_date) <= $3
					AND transaction_type = 'DISCOUNT'
			`, tenantID, year, currentMonth).Scan(&totalDiscounts)

			// Get TDS (expense - deduct from profit)
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND EXTRACT(MONTH FROM transaction_date) <= $3
					AND transaction_type = 'TDS'
			`, tenantID, year, currentMonth).Scan(&totalTDS)

			// Calculate total sales: egg sales + discounts
			totalSales = totalSales + totalDiscounts

			// Calculate total expense: feed + medicine + labor + other + TDS
			totalExpense = totalFeed + totalMedicine + totalLabor + totalOther + totalTDS
		} else {
			// For past years, include all months
			// Get egg sales (total sales)
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND category = 'EGG' AND transaction_type = 'SALE'
			`, tenantID, year).Scan(&totalSales)

			// Get feed purchases (excluding medicine items)
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND category = 'FEED' AND transaction_type = 'PURCHASE'
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
			`, tenantID, year).Scan(&totalFeed)

			// Get medicine expenses (from MEDICINE category)
			var medicineFromMedicineCategory float64
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND category = 'MEDICINE' AND transaction_type IN ('PURCHASE', 'SALE')
			`, tenantID, year).Scan(&medicineFromMedicineCategory)

			// Get medicine expenses that might be incorrectly categorized as FEED
			var medicineFromFeedCategory float64
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
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
			`, tenantID, year).Scan(&medicineFromFeedCategory)

			// Total medicine expense = medicine from MEDICINE category + medicine from FEED category
			totalMedicine = medicineFromMedicineCategory + medicineFromFeedCategory

			// Get labor expenses (OTHER category with labor-related item_name)
			var laborFromExpense float64
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND transaction_type = 'EXPENSE'
					AND category = 'OTHER' 
					AND UPPER(item_name) LIKE '%LABOR%'
			`, tenantID, year).Scan(&laborFromExpense)
			totalLabor = laborFromExpense

			// Get other expenses (includes electricity and other miscellaneous expenses, excluding labor)
			var otherFromExpense float64
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND category = 'OTHER' 
					AND transaction_type = 'EXPENSE'
					AND NOT (UPPER(item_name) LIKE '%LABOR%')
			`, tenantID, year).Scan(&otherFromExpense)
			totalOther = otherFromExpense

			// Get discounts (income - add to sales)
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND transaction_type = 'DISCOUNT'
			`, tenantID, year).Scan(&totalDiscounts)

			// Get TDS (expense - deduct from profit)
			database.DB.QueryRow(`
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions
				WHERE tenant_id = $1 
					AND EXTRACT(YEAR FROM transaction_date) = $2
					AND transaction_type = 'TDS'
			`, tenantID, year).Scan(&totalTDS)

			// Calculate total sales: egg sales + discounts
			totalSales = totalSales + totalDiscounts

			// Calculate total expense: feed + medicine + labor + other + TDS
			totalExpense = totalFeed + totalMedicine + totalLabor + totalOther + totalTDS
		}

		// Calculate net profit: (Sales + Discounts) - (Feed + Medicine + Labor + Other + TDS)
		// This should be close to the 12-month monthly net profit
		netProfit = totalSales - totalExpense

		// Check sensitive data
		shouldHideEggs, _ := utils.IsDataSensitive(database.DB, tenantID, "EGGS_SOLD", perms.CanViewSensitiveData)
		shouldHideFeed, _ := utils.IsDataSensitive(database.DB, tenantID, "FEED_PURCHASED", perms.CanViewSensitiveData)
		shouldHideProfit, _ := utils.IsDataSensitive(database.DB, tenantID, "NET_PROFIT", perms.CanViewSensitiveData)

		if shouldHideEggs {
			totalSales = 0
		}
		if shouldHideFeed {
			totalExpense = 0
		}
		if shouldHideProfit {
			netProfit = 0
		}

		summaries = append(summaries, YearlySummary{
			Year:         year,
			TotalSales:   totalSales,
			TotalExpense: totalExpense,
			NetProfit:    netProfit,
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

		// Get labor expense (OTHER category with labor-related item_name)
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND transaction_type = 'EXPENSE'
				AND category = 'OTHER' 
				AND UPPER(item_name) LIKE '%LABOR%'
		`, tenantID, year, month).Scan(&laborExpense)

		// Get other expenses (OTHER category, EXPENSE type, excluding labor expenses)
		database.DB.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'OTHER' 
				AND transaction_type = 'EXPENSE'
				AND NOT (UPPER(item_name) LIKE '%LABOR%')
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

// GetMonthlyBreakdown returns detailed transaction breakdown for a specific month and category
func GetMonthlyBreakdown(w http.ResponseWriter, r *http.Request) {
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
	category := r.URL.Query().Get("category")

	if yearStr == "" || monthStr == "" || category == "" {
		respondWithError(w, http.StatusBadRequest, "year, month, and category parameters required")
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid year parameter")
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		respondWithError(w, http.StatusBadRequest, "Invalid month parameter")
		return
	}

	// Build query based on category
	// Note: status column doesn't exist in the transactions table, so we exclude it
	// Special handling for PAYMENT - it's a transaction_type, not a category
	var query string
	var queryArgs []interface{}

	if category == "PAYMENT" {
		// PAYMENT is a transaction_type, not a category, so we filter differently
		query = `
			SELECT id, transaction_date, transaction_type, category,
			       item_name, quantity, unit, rate, amount, notes,
			       created_at, updated_at
			FROM transactions
			WHERE tenant_id = $1
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND transaction_type = 'PAYMENT'
		`
		queryArgs = []interface{}{tenantID, year, month}
	} else {
		query = `
			SELECT id, transaction_date, transaction_type, category,
			       item_name, quantity, unit, rate, amount, notes,
			       created_at, updated_at
			FROM transactions
			WHERE tenant_id = $1
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = $4
		`
		queryArgs = []interface{}{tenantID, year, month, category}

		// Add transaction type filter based on category
		switch category {
		case "EGG":
			query += " AND transaction_type = 'SALE'"
		case "FEED":
			// Exclude medicine items from feed
			query += ` AND transaction_type = 'PURCHASE'
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
				)`
		case "MEDICINE":
			query += " AND transaction_type IN ('PURCHASE', 'SALE')"
		case "OTHER":
			// Exclude PAYMENT transactions from OTHER expenses - payments are shown separately
			query += " AND transaction_type = 'PURCHASE'"
		}
	}

	query += " ORDER BY transaction_date ASC, created_at ASC"

	rows, err := database.DB.Query(query, queryArgs...)
	if err != nil {
		log.Printf("Error querying transactions: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch transactions")
		return
	}
	defer rows.Close()

	type TransactionDetail struct {
		ID              int      `json:"id"`
		TransactionDate string   `json:"transaction_date"`
		TransactionType string   `json:"transaction_type"`
		Category        string   `json:"category"`
		ItemName        *string  `json:"item_name,omitempty"`
		Quantity        *float64 `json:"quantity,omitempty"`
		Unit            *string  `json:"unit,omitempty"`
		Rate            *float64 `json:"rate,omitempty"`
		Amount          float64  `json:"amount"`
		Notes           *string  `json:"notes,omitempty"`
		Status          string   `json:"status"`
		CreatedAt       string   `json:"created_at"`
		UpdatedAt       string   `json:"updated_at"`
	}

	var transactions []TransactionDetail
	for rows.Next() {
		var txn TransactionDetail
		var itemName, unit, notes sql.NullString
		var quantity, rate sql.NullFloat64
		var transactionDate, createdAt, updatedAt time.Time

		// Set default status since column doesn't exist
		txn.Status = "APPROVED"

		err := rows.Scan(
			&txn.ID, &transactionDate, &txn.TransactionType, &txn.Category,
			&itemName, &quantity, &unit, &rate, &txn.Amount,
			&notes, &createdAt, &updatedAt,
		)
		if err != nil {
			log.Printf("Error scanning transaction: %v", err)
			continue
		}

		// Format dates
		txn.TransactionDate = transactionDate.Format("2006-01-02")
		txn.CreatedAt = createdAt.Format(time.RFC3339)
		txn.UpdatedAt = updatedAt.Format(time.RFC3339)

		// Handle nullable fields
		if itemName.Valid {
			txn.ItemName = &itemName.String
		}
		if quantity.Valid {
			txn.Quantity = &quantity.Float64
		}
		if unit.Valid {
			txn.Unit = &unit.String
		}
		if rate.Valid {
			txn.Rate = &rate.Float64
		}
		if notes.Valid {
			txn.Notes = &notes.String
		}

		// Check sensitive data
		shouldHide, _ := utils.IsDataSensitive(database.DB, tenantID, "EGGS_SOLD", perms.CanViewSensitiveData)
		if shouldHide && txn.Category == "EGG" && txn.TransactionType == "SALE" {
			txn.Amount = 0
			if txn.Quantity != nil {
				*txn.Quantity = 0
			}
		}

		shouldHide, _ = utils.IsDataSensitive(database.DB, tenantID, "FEED_PURCHASED", perms.CanViewSensitiveData)
		if shouldHide && txn.Category == "FEED" && txn.TransactionType == "PURCHASE" {
			txn.Amount = 0
			if txn.Quantity != nil {
				*txn.Quantity = 0
			}
		}

		transactions = append(transactions, txn)
	}

	// Calculate monthly average price (for EGG category)
	var averagePrice float64
	if category == "EGG" && len(transactions) > 0 {
		var totalAmount, totalQuantity float64
		for _, txn := range transactions {
			totalAmount += txn.Amount
			if txn.Quantity != nil {
				totalQuantity += *txn.Quantity
			}
		}
		if totalQuantity > 0 {
			averagePrice = totalAmount / totalQuantity
		}
	}

	// Group transactions by date
	groupedByDate := make(map[string][]TransactionDetail)
	for _, txn := range transactions {
		date := txn.TransactionDate
		groupedByDate[date] = append(groupedByDate[date], txn)
	}

	// Convert to array format for frontend
	type DateGroup struct {
		Date         string              `json:"date"`
		Transactions []TransactionDetail `json:"transactions"`
		TotalAmount  float64             `json:"total_amount"`
	}

	var dateGroups []DateGroup
	for date, txns := range groupedByDate {
		var totalAmount float64
		for _, txn := range txns {
			totalAmount += txn.Amount
		}
		dateGroups = append(dateGroups, DateGroup{
			Date:         date,
			Transactions: txns,
			TotalAmount:  totalAmount,
		})
	}

	// Sort by date
	for i := 0; i < len(dateGroups)-1; i++ {
		for j := i + 1; j < len(dateGroups); j++ {
			if dateGroups[i].Date > dateGroups[j].Date {
				dateGroups[i], dateGroups[j] = dateGroups[j], dateGroups[i]
			}
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"category":        category,
			"year":            year,
			"month":           month,
			"transactions":    transactions,
			"grouped_by_date": dateGroups,
			"average_price":   averagePrice,
			"total_count":     len(transactions),
		},
	})
}
