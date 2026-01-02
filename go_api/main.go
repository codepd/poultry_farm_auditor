package main

import (
	"log"
	"net/http"
	"poultry-farm-api/config"
	"poultry-farm-api/database"
	"poultry-farm-api/handlers"
	"poultry-farm-api/middleware"

	"github.com/gorilla/mux"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := database.Init(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create router
	r := mux.NewRouter()

	// Apply middleware first (runs on all requests including OPTIONS)
	r.Use(middleware.Logging)
	r.Use(middleware.CORS)

	// Global OPTIONS handler for CORS preflight - must match all paths
	// Register this before other routes so it catches OPTIONS requests
	r.Methods("OPTIONS").MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return true // Match all OPTIONS requests
	}).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS middleware already set headers and would return early,
		// but this ensures the route matches
		w.WriteHeader(http.StatusOK)
	})

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// API routes
	apiRouter := r.PathPrefix("/api").Subrouter()

	// Public routes
	apiRouter.HandleFunc("/auth/login", handlers.Login(cfg)).Methods("POST")

	// Protected routes (require authentication)
	protectedRouter := apiRouter.PathPrefix("").Subrouter()
	protectedRouter.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	// Tenant routes
	// IMPORTANT: More specific routes must come before parameterized routes
	protectedRouter.HandleFunc("/tenants/items", handlers.GetTenantItems).Methods("GET")
	protectedRouter.HandleFunc("/tenants", handlers.GetTenants).Methods("GET")
	protectedRouter.HandleFunc("/tenants", handlers.CreateTenant).Methods("POST")
	protectedRouter.HandleFunc("/tenants/{id}", handlers.GetTenant).Methods("GET")
	protectedRouter.HandleFunc("/tenants/{id}", handlers.UpdateTenant).Methods("PUT")

	// Transaction routes
	protectedRouter.HandleFunc("/transactions", handlers.GetTransactions).Methods("GET")
	protectedRouter.HandleFunc("/transactions", handlers.CreateTransaction).Methods("POST")
	protectedRouter.HandleFunc("/transactions/{id}", handlers.GetTransaction).Methods("GET")
	protectedRouter.HandleFunc("/transactions/{id}", handlers.UpdateTransaction).Methods("PUT")
	protectedRouter.HandleFunc("/transactions/{id}", handlers.DeleteTransaction).Methods("DELETE")
	protectedRouter.HandleFunc("/transactions/{id}/submit", handlers.SubmitTransaction).Methods("POST")
	protectedRouter.HandleFunc("/transactions/{id}/approve", handlers.ApproveTransaction).Methods("POST")
	protectedRouter.HandleFunc("/transactions/{id}/reject", handlers.RejectTransaction).Methods("POST")

	// Hen batch routes
	protectedRouter.HandleFunc("/hen-batches", handlers.GetHenBatches).Methods("GET")
	protectedRouter.HandleFunc("/hen-batches", handlers.CreateHenBatch).Methods("POST")
	protectedRouter.HandleFunc("/hen-batches/{id}", handlers.GetHenBatch).Methods("GET")
	protectedRouter.HandleFunc("/hen-batches/{id}", handlers.UpdateHenBatch).Methods("PUT")
	protectedRouter.HandleFunc("/hen-batches/mortality", handlers.CreateMortality).Methods("POST")

	// Employee routes
	protectedRouter.HandleFunc("/employees", handlers.GetEmployees).Methods("GET")
	protectedRouter.HandleFunc("/employees", handlers.CreateEmployee).Methods("POST")
	protectedRouter.HandleFunc("/employees/{id}", handlers.GetEmployee).Methods("GET")
	protectedRouter.HandleFunc("/employees/{id}", handlers.UpdateEmployee).Methods("PUT")

	// User management routes
	protectedRouter.HandleFunc("/users", handlers.GetUsers).Methods("GET")
	protectedRouter.HandleFunc("/users/invite", handlers.InviteUser).Methods("POST")
	protectedRouter.HandleFunc("/users/invitations", handlers.GetInvitations).Methods("GET")

	// Public invitation acceptance
	apiRouter.HandleFunc("/users/accept-invite", handlers.AcceptInvite).Methods("POST")

	// Admin endpoints (for local development)
	apiRouter.HandleFunc("/admin/create-tenant", handlers.CreateTenantDirectly).Methods("POST")
	apiRouter.HandleFunc("/admin/create-owner", handlers.CreateOwnerDirectly).Methods("POST")

	// Analytics routes
	protectedRouter.HandleFunc("/analytics/enhanced-monthly-summary", handlers.GetEnhancedMonthlySummary).Methods("GET")
	protectedRouter.HandleFunc("/analytics/all-years-summary", handlers.GetAllYearsSummary).Methods("GET")
	protectedRouter.HandleFunc("/analytics/last-12-months", handlers.GetLast12MonthsSummary).Methods("GET")
	protectedRouter.HandleFunc("/analytics/monthly-breakdown", handlers.GetMonthlyBreakdown).Methods("GET")

	// Price history routes
	protectedRouter.HandleFunc("/prices", handlers.GetPriceHistory).Methods("GET")
	protectedRouter.HandleFunc("/prices", handlers.CreatePriceHistory).Methods("POST")
	protectedRouter.HandleFunc("/prices/{id}", handlers.UpdatePriceHistory).Methods("PUT")
	protectedRouter.HandleFunc("/prices/{id}", handlers.DeletePriceHistory).Methods("DELETE")
	protectedRouter.HandleFunc("/prices/monthly-egg-averages", handlers.GetMonthlyAverageEggPrices).Methods("GET")

	// Sensitive data config routes
	protectedRouter.HandleFunc("/sensitive-data-config", handlers.GetSensitiveDataConfig).Methods("GET")
	protectedRouter.HandleFunc("/sensitive-data-config", handlers.UpdateSensitiveDataConfig).Methods("PUT")

	// Receipt routes
	protectedRouter.HandleFunc("/transactions/{transaction_id}/receipts", handlers.GetReceipts).Methods("GET")
	protectedRouter.HandleFunc("/transactions/{transaction_id}/receipts", handlers.UploadReceipt(cfg)).Methods("POST")

	log.Printf("Server starting on port %s", cfg.APIPort)
	if err := http.ListenAndServe(":"+cfg.APIPort, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
