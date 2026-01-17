package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"poultry-farm-api/database"
)

func main() {
	// Initialize database connection
	database.InitDB()
	db := database.DB

	// Get tenant_id from command line or use a test one
	tenantID := os.Args[1]
	if tenantID == "" {
		log.Fatal("Usage: go run debug_medicine.go <tenant_id>")
	}

	fmt.Printf("Checking medicine transactions for tenant: %s\n\n", tenantID)

	// Check total medicine transactions
	var totalCount int
	var totalAmount float64
	err = db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE tenant_id = $1
			AND category = 'MEDICINE'
			AND transaction_type IN ('PURCHASE', 'SALE')
	`, tenantID).Scan(&totalCount, &totalAmount)
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	fmt.Printf("Total medicine transactions: %d, Total amount: %.2f\n\n", totalCount, totalAmount)

	// Check by month for 2025
	fmt.Println("Medicine transactions by month (2025):")
	rows, err := db.Query(`
		SELECT 
			EXTRACT(YEAR FROM transaction_date)::int as year,
			EXTRACT(MONTH FROM transaction_date)::int as month,
			COUNT(*) as count,
			SUM(amount) as total
		FROM transactions
		WHERE tenant_id = $1
			AND category = 'MEDICINE'
			AND transaction_type IN ('PURCHASE', 'SALE')
			AND EXTRACT(YEAR FROM transaction_date) = 2025
		GROUP BY 
			EXTRACT(YEAR FROM transaction_date),
			EXTRACT(MONTH FROM transaction_date)
		ORDER BY year, month
	`, tenantID)
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var year, month, count int
		var total float64
		rows.Scan(&year, &month, &count, &total)
		fmt.Printf("  %d-%02d: %d transactions, ₹%.2f\n", year, month, count, total)
	}

	// Check what GetLast12MonthsSummary would return
	fmt.Println("\n\nWhat GetLast12MonthsSummary would return:")
	for i := 11; i >= 0; i-- {
		date := time.Now().AddDate(0, -i, 0)
		year := date.Year()
		month := int(date.Month())

		var medicineAmount float64
		err = db.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE tenant_id = $1 
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
				AND category = 'MEDICINE' AND transaction_type IN ('PURCHASE', 'SALE')
		`, tenantID, year, month).Scan(&medicineAmount)

		if err != nil {
			fmt.Printf("  %d-%02d: ERROR - %v\n", year, month, err)
		} else {
			fmt.Printf("  %d-%02d: ₹%.2f\n", year, month, medicineAmount)
		}
	}
}

