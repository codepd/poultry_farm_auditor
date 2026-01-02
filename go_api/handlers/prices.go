package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

	// If price_type is EGG, fetch monthly averages from transactions
	if priceType == "EGG" {
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

		// Map to store prices by month and item
		type MonthKey struct {
			Year  int
			Month int
		}
		priceMap := make(map[MonthKey]map[string]float64) // month -> item_name -> price

		for rows.Next() {
			var year, month int
			var itemName string
			var totalAmount, totalQuantity, averagePrice sql.NullFloat64

			err := rows.Scan(&year, &month, &itemName, &totalAmount, &totalQuantity, &averagePrice)
			if err != nil {
				continue
			}

			if averagePrice.Valid {
				key := MonthKey{Year: year, Month: month}
				if priceMap[key] == nil {
					priceMap[key] = make(map[string]float64)
				}
				priceMap[key][itemName] = averagePrice.Float64
			}
		}

		// Helper function to check if item name matches egg type
		matchesEggType := func(itemName, eggType string) bool {
			normalized := strings.ToUpper(strings.TrimSpace(itemName))
			return strings.Contains(normalized, eggType) && strings.Contains(normalized, "EGG")
		}

		// Convert map to slice and fill missing Medium/Small prices
		for monthKey, itemPrices := range priceMap {
			// Find Large egg price
			var largePrice float64
			hasLarge := false
			for itemName, price := range itemPrices {
				if matchesEggType(itemName, "LARGE") {
					largePrice = price
					hasLarge = true
					break
				}
			}

			// Add all existing prices
			for itemName, price := range itemPrices {
				priceHistory := PriceHistory{
					ID:        0, // Calculated price
					TenantID:  tenantID.String(),
					PriceType: "EGG",
					ItemName:  itemName,
					PriceDate: fmt.Sprintf("%04d-%02d-01", monthKey.Year, monthKey.Month),
					Price:     price,
					CreatedAt: fmt.Sprintf("%04d-%02d-01T00:00:00Z", monthKey.Year, monthKey.Month),
				}
				prices = append(prices, priceHistory)
			}

			// Fill missing Medium and Small prices if Large exists
			if hasLarge {
				// Check if Medium and Small eggs exist
				hasMedium := false
				hasSmall := false
				for itemName := range itemPrices {
					if matchesEggType(itemName, "MEDIUM") {
						hasMedium = true
					}
					if matchesEggType(itemName, "SMALL") {
						hasSmall = true
					}
				}

				// Add Medium egg if missing (Large - 0.10)
				if !hasMedium {
					priceHistory := PriceHistory{
						ID:        0, // Calculated price
						TenantID:  tenantID.String(),
						PriceType: "EGG",
						ItemName:  "MEDIUM EGG",
						PriceDate: fmt.Sprintf("%04d-%02d-01", monthKey.Year, monthKey.Month),
						Price:     largePrice - 0.10,
						CreatedAt: fmt.Sprintf("%04d-%02d-01T00:00:00Z", monthKey.Year, monthKey.Month),
					}
					prices = append(prices, priceHistory)
				}

				// Add Small egg if missing (Large - 0.15)
				if !hasSmall {
					priceHistory := PriceHistory{
						ID:        0, // Calculated price
						TenantID:  tenantID.String(),
						PriceType: "EGG",
						ItemName:  "SMALL EGG",
						PriceDate: fmt.Sprintf("%04d-%02d-01", monthKey.Year, monthKey.Month),
						Price:     largePrice - 0.15,
						CreatedAt: fmt.Sprintf("%04d-%02d-01T00:00:00Z", monthKey.Year, monthKey.Month),
					}
					prices = append(prices, priceHistory)
				}
			}
		}
	} else {
		// For FEED or no filter, fetch from price_history table
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
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch price history")
			return
		}
		defer rows.Close()

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
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    prices,
	})
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
