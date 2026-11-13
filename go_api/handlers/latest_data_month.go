package handlers

import (
	"database/sql"
	"net/http"

	"poultry-farm-api/database"
	"poultry-farm-api/middleware"

	"github.com/google/uuid"
)

// GetLatestDataMonth returns the most recent year/month that actually has data for the tenant,
// so the Home dashboard opens on a month with figures instead of an empty "last calendar month".
//
// Priority: latest month with EGG sales (a loaded ledger always has egg sales),
// falling back to the latest month with any transaction.
// No flag is needed at ledger-load time: committing a ledger (or any manual entry) moves this automatically.
func GetLatestDataMonth(w http.ResponseWriter, r *http.Request) {
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

	queries := []struct {
		source string
		sql    string
	}{
		{"EGG_SALES", `
			SELECT EXTRACT(YEAR FROM MAX(transaction_date))::int, EXTRACT(MONTH FROM MAX(transaction_date))::int
			FROM transactions WHERE tenant_id = $1 AND category = 'EGG'`},
		{"ANY_TRANSACTION", `
			SELECT EXTRACT(YEAR FROM MAX(transaction_date))::int, EXTRACT(MONTH FROM MAX(transaction_date))::int
			FROM transactions WHERE tenant_id = $1`},
	}

	for _, q := range queries {
		var year, month sql.NullInt64
		if err := database.DB.QueryRow(q.sql, tenantID).Scan(&year, &month); err != nil {
			continue
		}
		if year.Valid && month.Valid {
			respondWithJSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"year":     year.Int64,
					"month":    month.Int64,
					"has_data": true,
					"source":   q.source,
				},
			})
			return
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    map[string]interface{}{"has_data": false},
	})
}
