package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type PriceHistory struct {
	ID        int     `json:"id"`
	TenantID  string  `json:"tenant_id"`
	PriceDate string  `json:"price_date"`
	PriceType string  `json:"price_type"` // 'EGG' or 'FEED'
	ItemName  string  `json:"item_name"`
	Price     float64 `json:"price"`
	CreatedAt string  `json:"created_at"`
}

// GetPriceHistory returns price history for a tenant
func GetPriceHistory(w http.ResponseWriter, r *http.Request) {
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

	// Get query parameters
	priceType := r.URL.Query().Get("price_type") // 'EGG' or 'FEED'
	itemName := r.URL.Query().Get("item_name")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	var prices []PriceHistory

	switch priceType {
	case "EGG":
		txEggPrices, err := fetchEggPricesFromTransactions(tenantID, itemName, startDate, endDate)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch monthly average egg prices")
			return
		}
		storedEggPrices, err := fetchStoredPrices(tenantID, "EGG", itemName, startDate, endDate)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch price history")
			return
		}
		prices = mergeEggPrices(txEggPrices, storedEggPrices)
	case "":
		txEggPrices, err := fetchEggPricesFromTransactions(tenantID, itemName, startDate, endDate)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch monthly average egg prices")
			return
		}
		storedPrices, err := fetchStoredPrices(tenantID, "", itemName, startDate, endDate)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch price history")
			return
		}
		var storedEggPrices, otherStoredPrices []PriceHistory
		for _, p := range storedPrices {
			if p.PriceType == "EGG" {
				storedEggPrices = append(storedEggPrices, p)
			} else {
				otherStoredPrices = append(otherStoredPrices, p)
			}
		}
		prices = append(mergeEggPrices(txEggPrices, storedEggPrices), otherStoredPrices...)
	default:
		storedPrices, err := fetchStoredPrices(tenantID, priceType, itemName, startDate, endDate)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch price history")
			return
		}
		prices = storedPrices
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    prices,
	})
}

// mergeEggPrices prefers transaction-derived monthly averages per (YYYY-MM, item_name)
// and falls back to stored price_history EGG rows for months without transaction data.
func mergeEggPrices(txPrices, storedPrices []PriceHistory) []PriceHistory {
	seen := make(map[string]bool, len(txPrices))
	merged := make([]PriceHistory, 0, len(txPrices)+len(storedPrices))

	eggKey := func(p PriceHistory) string {
		ym := p.PriceDate
		if len(ym) >= 7 {
			ym = ym[:7]
		}
		return ym + "|" + strings.ToUpper(strings.TrimSpace(p.ItemName))
	}

	for _, p := range txPrices {
		seen[eggKey(p)] = true
		merged = append(merged, p)
	}
	for _, p := range storedPrices {
		if !seen[eggKey(p)] {
			merged = append(merged, p)
		}
	}
	return merged
}

func fetchEggPricesFromTransactions(tenantID uuid.UUID, itemName, startDate, endDate string) ([]PriceHistory, error) {
	query := `
		SELECT 
			EXTRACT(YEAR FROM transaction_date)::INTEGER as year,
			EXTRACT(MONTH FROM transaction_date)::INTEGER as month,
			item_name,
			SUM(amount) as total_amount,
			SUM(quantity) as total_quantity,
			CASE 
				WHEN SUM(quantity) > 0 THEN SUM(amount) / SUM(quantity)
				ELSE 0
			END as average_price
		FROM transactions
		WHERE tenant_id = $1
			AND category = 'EGG'
			AND transaction_type = 'SALE'
			AND quantity IS NOT NULL
			AND quantity > 0
	`
	args := []interface{}{tenantID}
	argIndex := 2

	if itemName != "" {
		query += ` AND item_name = $` + strconv.Itoa(argIndex)
		args = append(args, itemName)
		argIndex++
	}

	if startDate != "" {
		query += ` AND transaction_date >= $` + strconv.Itoa(argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		query += ` AND transaction_date <= $` + strconv.Itoa(argIndex)
		args = append(args, endDate)
		argIndex++
	}

	query += `
		GROUP BY EXTRACT(YEAR FROM transaction_date), EXTRACT(MONTH FROM transaction_date), item_name
		ORDER BY year DESC, month DESC, item_name
	`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type MonthKey struct {
		Year  int
		Month int
	}
	priceMap := make(map[MonthKey]map[string]float64)

	for rows.Next() {
		var year, month int
		var item string
		var totalAmount, totalQuantity, averagePrice sql.NullFloat64

		if err := rows.Scan(&year, &month, &item, &totalAmount, &totalQuantity, &averagePrice); err != nil {
			continue
		}

		if averagePrice.Valid {
			key := MonthKey{Year: year, Month: month}
			if priceMap[key] == nil {
				priceMap[key] = make(map[string]float64)
			}
			priceMap[key][item] = averagePrice.Float64
		}
	}

	matchesEggType := func(item, eggType string) bool {
		normalized := strings.ToUpper(strings.TrimSpace(item))
		return strings.Contains(normalized, eggType) && strings.Contains(normalized, "EGG")
	}

	var prices []PriceHistory
	for monthKey, itemPrices := range priceMap {
		var largePrice float64
		hasLarge := false
		for item, price := range itemPrices {
			if matchesEggType(item, "LARGE") {
				largePrice = price
				hasLarge = true
				break
			}
		}

		for item, price := range itemPrices {
			prices = append(prices, PriceHistory{
				ID:        0,
				TenantID:  tenantID.String(),
				PriceType: "EGG",
				ItemName:  item,
				PriceDate: fmt.Sprintf("%04d-%02d-01", monthKey.Year, monthKey.Month),
				Price:     price,
				CreatedAt: fmt.Sprintf("%04d-%02d-01T00:00:00Z", monthKey.Year, monthKey.Month),
			})
		}

		if hasLarge {
			hasMedium := false
			hasSmall := false
			for item := range itemPrices {
				if matchesEggType(item, "MEDIUM") {
					hasMedium = true
				}
				if matchesEggType(item, "SMALL") {
					hasSmall = true
				}
			}

			if !hasMedium {
				prices = append(prices, PriceHistory{
					ID:        0,
					TenantID:  tenantID.String(),
					PriceType: "EGG",
					ItemName:  "MEDIUM EGG",
					PriceDate: fmt.Sprintf("%04d-%02d-01", monthKey.Year, monthKey.Month),
					Price:     largePrice - 0.10,
					CreatedAt: fmt.Sprintf("%04d-%02d-01T00:00:00Z", monthKey.Year, monthKey.Month),
				})
			}

			if !hasSmall {
				prices = append(prices, PriceHistory{
					ID:        0,
					TenantID:  tenantID.String(),
					PriceType: "EGG",
					ItemName:  "SMALL EGG",
					PriceDate: fmt.Sprintf("%04d-%02d-01", monthKey.Year, monthKey.Month),
					Price:     largePrice - 0.15,
					CreatedAt: fmt.Sprintf("%04d-%02d-01T00:00:00Z", monthKey.Year, monthKey.Month),
				})
			}
		}
	}
	return prices, nil
}

func fetchStoredPrices(tenantID uuid.UUID, priceType, itemName, startDate, endDate string) ([]PriceHistory, error) {
	query := `
		SELECT id, tenant_id, price_date, price_type, item_name, price, created_at
		FROM price_history
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIndex := 2

	if priceType != "" {
		query += ` AND price_type = $` + strconv.Itoa(argIndex)
		args = append(args, priceType)
		argIndex++
	}

	if itemName != "" {
		query += ` AND item_name = $` + strconv.Itoa(argIndex)
		args = append(args, itemName)
		argIndex++
	}

	if startDate != "" {
		query += ` AND price_date >= $` + strconv.Itoa(argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		query += ` AND price_date <= $` + strconv.Itoa(argIndex)
		args = append(args, endDate)
		argIndex++
	}

	query += ` ORDER BY price_date DESC, item_name`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []PriceHistory
	for rows.Next() {
		var price PriceHistory
		var createdAt sql.NullTime

		err := rows.Scan(
			&price.ID, &price.TenantID, &price.PriceDate,
			&price.PriceType, &price.ItemName, &price.Price, &createdAt,
		)
		if err != nil {
			continue
		}

		if createdAt.Valid {
			price.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z")
		}

		prices = append(prices, price)
	}
	return prices, nil
}

// CreatePriceHistory creates a new price history entry
func CreatePriceHistory(w http.ResponseWriter, r *http.Request) {
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

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanEditTransactions {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	var req struct {
		PriceDate string  `json:"price_date"`
		PriceType string  `json:"price_type"`
		ItemName  string  `json:"item_name"`
		Price     float64 `json:"price"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Insert price history
	var priceID int
	err = database.DB.QueryRow(`
		INSERT INTO price_history (tenant_id, price_date, price_type, item_name, price)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, price_date, price_type, item_name) 
		DO UPDATE SET price = EXCLUDED.price
		RETURNING id
	`, tenantID, req.PriceDate, req.PriceType, req.ItemName, req.Price).Scan(&priceID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create price history")
		return
	}

	// Return created price
	var price PriceHistory
	err = database.DB.QueryRow(`
		SELECT id, tenant_id, price_date, price_type, item_name, price, created_at
		FROM price_history
		WHERE id = $1
	`, priceID).Scan(
		&price.ID, &price.TenantID, &price.PriceDate,
		&price.PriceType, &price.ItemName, &price.Price, &price.CreatedAt,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch created price")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    price,
	})
}

// UpdatePriceHistory updates an existing price history entry
func UpdatePriceHistory(w http.ResponseWriter, r *http.Request) {
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

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanEditTransactions {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	// Get price ID from URL
	priceIDStr := strings.TrimPrefix(r.URL.Path, "/api/prices/")
	priceID, err := strconv.Atoi(priceIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid price ID")
		return
	}

	var req struct {
		PriceDate string  `json:"price_date"`
		PriceType string  `json:"price_type"`
		ItemName  string  `json:"item_name"`
		Price     float64 `json:"price"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Verify the price belongs to the tenant
	var existingTenantID uuid.UUID
	err = database.DB.QueryRow(`
		SELECT tenant_id FROM price_history WHERE id = $1
	`, priceID).Scan(&existingTenantID)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Price entry not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if existingTenantID != tenantID {
		respondWithError(w, http.StatusForbidden, "Price entry does not belong to your tenant")
		return
	}

	// Update price history
	_, err = database.DB.Exec(`
		UPDATE price_history
		SET price_date = $1, price_type = $2, item_name = $3, price = $4
		WHERE id = $5 AND tenant_id = $6
	`, req.PriceDate, req.PriceType, req.ItemName, req.Price, priceID, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update price history")
		return
	}

	// Return updated price
	var price PriceHistory
	var createdAt sql.NullTime
	err = database.DB.QueryRow(`
		SELECT id, tenant_id, price_date, price_type, item_name, price, created_at
		FROM price_history
		WHERE id = $1
	`, priceID).Scan(
		&price.ID, &price.TenantID, &price.PriceDate,
		&price.PriceType, &price.ItemName, &price.Price, &createdAt,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch updated price")
		return
	}

	if createdAt.Valid {
		price.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z")
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    price,
	})
}

// DeletePriceHistory deletes a price history entry
func DeletePriceHistory(w http.ResponseWriter, r *http.Request) {
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

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanEditTransactions {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	// Get price ID from URL
	priceIDStr := strings.TrimPrefix(r.URL.Path, "/api/prices/")
	priceID, err := strconv.Atoi(priceIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid price ID")
		return
	}

	// Verify the price belongs to the tenant
	var existingTenantID uuid.UUID
	err = database.DB.QueryRow(`
		SELECT tenant_id FROM price_history WHERE id = $1
	`, priceID).Scan(&existingTenantID)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Price entry not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if existingTenantID != tenantID {
		respondWithError(w, http.StatusForbidden, "Price entry does not belong to your tenant")
		return
	}

	// Delete price history
	_, err = database.DB.Exec(`
		DELETE FROM price_history WHERE id = $1 AND tenant_id = $2
	`, priceID, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete price history")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Price entry deleted successfully",
	})
}

// GetMonthlyAverageEggPrices returns monthly average egg prices calculated from transactions
func GetMonthlyAverageEggPrices(w http.ResponseWriter, r *http.Request) {
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

	// Get query parameters
	itemName := r.URL.Query().Get("item_name")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Build query to calculate monthly average prices from transactions
	query := `
		SELECT 
			EXTRACT(YEAR FROM transaction_date)::INTEGER as year,
			EXTRACT(MONTH FROM transaction_date)::INTEGER as month,
			item_name,
			SUM(amount) as total_amount,
			SUM(quantity) as total_quantity,
			CASE 
				WHEN SUM(quantity) > 0 THEN SUM(amount) / SUM(quantity)
				ELSE 0
			END as average_price
		FROM transactions
		WHERE tenant_id = $1
			AND category = 'EGG'
			AND transaction_type = 'SALE'
			AND quantity IS NOT NULL
			AND quantity > 0
	`
	args := []interface{}{tenantID}
	argIndex := 2

	if itemName != "" {
		query += ` AND item_name = $` + strconv.Itoa(argIndex)
		args = append(args, itemName)
		argIndex++
	}

	if startDate != "" {
		query += ` AND transaction_date >= $` + strconv.Itoa(argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		query += ` AND transaction_date <= $` + strconv.Itoa(argIndex)
		args = append(args, endDate)
		argIndex++
	}

	query += `
		GROUP BY EXTRACT(YEAR FROM transaction_date), EXTRACT(MONTH FROM transaction_date), item_name
		ORDER BY year DESC, month DESC, item_name
	`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch monthly average egg prices")
		return
	}
	defer rows.Close()

	type MonthlyAveragePrice struct {
		Year          int     `json:"year"`
		Month         int     `json:"month"`
		ItemName      string  `json:"item_name"`
		AveragePrice  float64 `json:"average_price"`
		TotalAmount   float64 `json:"total_amount"`
		TotalQuantity float64 `json:"total_quantity"`
		PriceDate     string  `json:"price_date"` // First day of the month for display
		PriceType     string  `json:"price_type"` // Always 'EGG'
	}

	var monthlyAverages []MonthlyAveragePrice
	for rows.Next() {
		var avg MonthlyAveragePrice
		var year, month int
		var itemName string
		var totalAmount, totalQuantity, averagePrice sql.NullFloat64

		err := rows.Scan(&year, &month, &itemName, &totalAmount, &totalQuantity, &averagePrice)
		if err != nil {
			continue
		}

		avg.Year = year
		avg.Month = month
		avg.ItemName = itemName
		avg.PriceType = "EGG"
		// Format date as first day of the month (YYYY-MM-01)
		avg.PriceDate = fmt.Sprintf("%04d-%02d-01", year, month)

		if totalAmount.Valid {
			avg.TotalAmount = totalAmount.Float64
		}
		if totalQuantity.Valid {
			avg.TotalQuantity = totalQuantity.Float64
		}
		if averagePrice.Valid {
			avg.AveragePrice = averagePrice.Float64
		}

		monthlyAverages = append(monthlyAverages, avg)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    monthlyAverages,
	})
}

type FeedPriceValidationRow struct {
	TransactionID   int     `json:"transaction_id"`
	TransactionDate string  `json:"transaction_date"`
	ItemName        string  `json:"item_name"`
	Quantity        float64 `json:"quantity"`
	Amount          float64 `json:"amount"`
	LedgerRatePerKg float64 `json:"ledger_rate_per_kg"`
	SheetDate       string  `json:"sheet_date,omitempty"`
	SheetBagRate    float64 `json:"sheet_bag_rate,omitempty"`
	SheetRatePerKg  float64 `json:"sheet_rate_per_kg,omitempty"`
	DiffPerKg       float64 `json:"diff_per_kg,omitempty"`
	DiffPct         float64 `json:"diff_pct,omitempty"`
	Status          string  `json:"status"` // ALIGNED, MISMATCH, NO_SHEET_RATE
}

type FeedPriceValidationSummary struct {
	Total       int `json:"total"`
	Aligned     int `json:"aligned"`
	Mismatch    int `json:"mismatch"`
	NoSheetRate int `json:"no_sheet_rate"`
}

type FeedPriceValidationResult struct {
	Summary FeedPriceValidationSummary `json:"summary"`
	Rows    []FeedPriceValidationRow   `json:"rows"`
}

func normalizeFeedItemName(raw string) string {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	switch {
	case upper == "PLM" || strings.Contains(upper, "PRE-LAYER") || strings.Contains(upper, "PRE LAYER"):
		return "PRE-LAYER MASH"
	case upper == "LM" || strings.Contains(upper, "LAYER"):
		return "LAYER MASH"
	case upper == "GM" || strings.Contains(upper, "GROWER"):
		return "GROWER MASH"
	case upper == "CM" || strings.Contains(upper, "CHICK"):
		return "CHICK MASH"
	default:
		return upper
	}
}

// EvaluateFeedPriceAlignment compares a ledger feed rate (₹/kg) against a sheet bag rate (₹/75kg bag).
func EvaluateFeedPriceAlignment(ledgerPerKg, sheetBagRate float64) (sheetPerKg, diffPerKg, diffPct float64, status string) {
	if sheetBagRate <= 0 || ledgerPerKg <= 0 {
		return 0, 0, 0, "NO_SHEET_RATE"
	}
	sheetPerKg = math.Round((sheetBagRate/75.0)*100) / 100
	diffPerKg = math.Round((ledgerPerKg-sheetPerKg)*100) / 100
	diffPct = math.Round(((diffPerKg/sheetPerKg)*100)*100) / 100
	if math.Abs(diffPerKg) <= 0.50 || math.Abs(diffPct) <= 2.0 {
		status = "ALIGNED"
	} else {
		status = "MISMATCH"
	}
	return sheetPerKg, diffPerKg, diffPct, status
}

// ValidateFeedPricesForTenant compares ledger FEED transactions against stored price_history FEED bag rates (÷ 75).
func ValidateFeedPricesForTenant(tenantID uuid.UUID, year, month int) (*FeedPriceValidationResult, error) {
	query := `
		SELECT id, transaction_date, COALESCE(item_name, ''), COALESCE(quantity, 0), COALESCE(rate, 0), COALESCE(amount, 0)
		FROM transactions
		WHERE tenant_id = $1 AND category = 'FEED'
	`
	args := []interface{}{tenantID}
	argIdx := 2
	if year > 0 {
		query += fmt.Sprintf(" AND EXTRACT(YEAR FROM transaction_date) = $%d", argIdx)
		args = append(args, year)
		argIdx++
	}
	if month >= 1 && month <= 12 {
		query += fmt.Sprintf(" AND EXTRACT(MONTH FROM transaction_date) = $%d", argIdx)
		args = append(args, month)
		argIdx++
	}
	query += " ORDER BY transaction_date DESC, id DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &FeedPriceValidationResult{
		Rows: []FeedPriceValidationRow{},
	}

	for rows.Next() {
		var txID int
		var txDate, rawItem string
		var qty, rate, amount float64
		if err := rows.Scan(&txID, &txDate, &rawItem, &qty, &rate, &amount); err != nil {
			continue
		}

		// Ledger feed rates are typically ₹/kg (< 500); if >= 500 it's a bag rate.
		ledgerPerKg := 0.0
		if rate > 0 && rate < 500 {
			ledgerPerKg = rate
		} else if rate >= 500 {
			ledgerPerKg = rate / 75.0
		} else if qty > 0 && amount > 0 {
			computed := amount / qty
			if computed >= 500 {
				ledgerPerKg = computed / 75.0
			} else {
				ledgerPerKg = computed
			}
		}
		if ledgerPerKg <= 0 {
			continue
		}
		ledgerPerKg = math.Round(ledgerPerKg*100) / 100

		dateOnly := txDate
		if len(dateOnly) > 10 {
			dateOnly = dateOnly[:10]
		}
		normItem := normalizeFeedItemName(rawItem)

		valRow := FeedPriceValidationRow{
			TransactionID:   txID,
			TransactionDate: dateOnly,
			ItemName:        normItem,
			Quantity:        qty,
			Amount:          amount,
			LedgerRatePerKg: ledgerPerKg,
			Status:          "NO_SHEET_RATE",
		}

		var sheetDate string
		var sheetBagRate float64
		err := database.DB.QueryRow(`
			SELECT price_date, price
			FROM price_history
			WHERE tenant_id = $1
			  AND price_type = 'FEED'
			  AND UPPER(item_name) = $2
			  AND price_date <= $3
			ORDER BY price_date DESC
			LIMIT 1
		`, tenantID, normItem, dateOnly).Scan(&sheetDate, &sheetBagRate)
		if err == nil && sheetBagRate > 0 {
			if len(sheetDate) > 10 {
				sheetDate = sheetDate[:10]
			}
			sheetPerKg, diffPerKg, diffPct, status := EvaluateFeedPriceAlignment(ledgerPerKg, sheetBagRate)
			valRow.SheetDate = sheetDate
			valRow.SheetBagRate = sheetBagRate
			valRow.SheetRatePerKg = sheetPerKg
			valRow.DiffPerKg = diffPerKg
			valRow.DiffPct = diffPct
			valRow.Status = status
		}

		result.Summary.Total++
		switch valRow.Status {
		case "ALIGNED":
			result.Summary.Aligned++
		case "MISMATCH":
			result.Summary.Mismatch++
		default:
			result.Summary.NoSheetRate++
		}
		result.Rows = append(result.Rows, valRow)
	}

	return result, nil
}

// GetFeedPriceValidation validates ledger FEED transaction rates against stored price_history sheet rates.
func GetFeedPriceValidation(w http.ResponseWriter, r *http.Request) {
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

	year := 0
	month := 0
	if yStr := r.URL.Query().Get("year"); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil && y > 0 {
			year = y
		}
	}
	if mStr := r.URL.Query().Get("month"); mStr != "" {
		if m, err := strconv.Atoi(mStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	result, err := ValidateFeedPricesForTenant(tenantID, year, month)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to validate feed prices against ledger")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}
