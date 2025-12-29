package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"poultry-farm-api/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GetTenants returns all tenants (with hierarchy support)
func GetTenants(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get tenant ID from query or use user's tenant
	tenantIDStr := r.URL.Query().Get("tenant_id")
	var tenantID uuid.UUID
	var err error

	if tenantIDStr != "" {
		tenantID, err = uuid.Parse(tenantIDStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid tenant_id")
			return
		}
	} else {
		tenantID, err = uuid.Parse(user.TenantID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
			return
		}
	}

	// Check if user has access to this tenant
	var hasAccess bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM tenant_users 
			WHERE user_id = $1 AND tenant_id = $2
		)
	`, user.UserID, tenantID).Scan(&hasAccess)

	if err != nil || !hasAccess {
		respondWithError(w, http.StatusForbidden, "Access denied to this tenant")
		return
	}

	// Get all child tenants (recursive)
	rows, err := database.DB.Query(`
		WITH RECURSIVE tenant_hierarchy AS (
			SELECT id, parent_id, name, location, country_code, currency, 
			       number_format, date_format, capacity, created_at, updated_at, 0 as level
			FROM tenants
			WHERE id = $1
			
			UNION ALL
			
			SELECT t.id, t.parent_id, t.name, t.location, t.country_code, t.currency,
			       t.number_format, t.date_format, t.capacity, t.created_at, t.updated_at, th.level + 1
			FROM tenants t
			INNER JOIN tenant_hierarchy th ON t.parent_id = th.id
		)
		SELECT id, parent_id, name, location, country_code, currency, 
		       number_format, date_format, capacity, created_at, updated_at
		FROM tenant_hierarchy
		ORDER BY level, name
	`, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch tenants")
		return
	}
	defer rows.Close()

	var tenants []models.Tenant
	for rows.Next() {
		var tenant models.Tenant
		var parentID sql.NullString
		var capacity sql.NullInt64

		err := rows.Scan(
			&tenant.ID, &parentID, &tenant.Name, &tenant.Location,
			&tenant.CountryCode, &tenant.Currency, &tenant.NumberFormat,
			&tenant.DateFormat, &capacity, &tenant.CreatedAt, &tenant.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if parentID.Valid {
			pid, _ := uuid.Parse(parentID.String)
			tenant.ParentID = &pid
		}
		if capacity.Valid {
			cap := int(capacity.Int64)
			tenant.Capacity = &cap
		}

		tenants = append(tenants, tenant)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    tenants,
	})
}

// GetTenant returns a single tenant by ID
func GetTenant(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	tenantID, err := uuid.Parse(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	// Check access
	var hasAccess bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM tenant_users 
			WHERE user_id = $1 AND tenant_id = $2
		)
	`, user.UserID, tenantID).Scan(&hasAccess)

	if err != nil || !hasAccess {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	var tenant models.Tenant
	var parentID sql.NullString
	var capacity sql.NullInt64

	err = database.DB.QueryRow(`
		SELECT id, parent_id, name, location, country_code, currency,
		       number_format, date_format, capacity, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`, tenantID).Scan(
		&tenant.ID, &parentID, &tenant.Name, &tenant.Location,
		&tenant.CountryCode, &tenant.Currency, &tenant.NumberFormat,
		&tenant.DateFormat, &capacity, &tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Tenant not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if parentID.Valid {
		pid, _ := uuid.Parse(parentID.String)
		tenant.ParentID = &pid
	}
	if capacity.Valid {
		cap := int(capacity.Int64)
		tenant.Capacity = &cap
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    tenant,
	})
}

// CreateTenant creates a new tenant (child tenant)
func CreateTenant(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Check permissions (only ADMIN, OWNER, CO_OWNER can create tenants)
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanManageUsers {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	var req models.TenantCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate parent_id if provided
	if req.ParentID != nil {
		// Verify parent exists and user has access
		var hasAccess bool
		err = database.DB.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM tenant_users 
				WHERE user_id = $1 AND tenant_id = $2
			)
		`, user.UserID, *req.ParentID).Scan(&hasAccess)

		if err != nil || !hasAccess {
			respondWithError(w, http.StatusForbidden, "Access denied to parent tenant")
			return
		}
	}

	// Insert tenant
	var tenantID uuid.UUID
	err = database.DB.QueryRow(`
		INSERT INTO tenants (parent_id, name, location, country_code, currency, 
		                     number_format, date_format, capacity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, req.ParentID, req.Name, req.Location, req.CountryCode, req.Currency,
		req.NumberFormat, req.DateFormat, req.Capacity).Scan(&tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create tenant")
		return
	}

	// Get created tenant
	var tenant models.Tenant
	var parentID sql.NullString
	var capacity sql.NullInt64

	err = database.DB.QueryRow(`
		SELECT id, parent_id, name, location, country_code, currency,
		       number_format, date_format, capacity, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`, tenantID).Scan(
		&tenant.ID, &parentID, &tenant.Name, &tenant.Location,
		&tenant.CountryCode, &tenant.Currency, &tenant.NumberFormat,
		&tenant.DateFormat, &capacity, &tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if parentID.Valid {
		pid, _ := uuid.Parse(parentID.String)
		tenant.ParentID = &pid
	}
	if capacity.Valid {
		cap := int(capacity.Int64)
		tenant.Capacity = &cap
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    tenant,
	})
}

// UpdateTenant updates an existing tenant
func UpdateTenant(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	tenantID, err := uuid.Parse(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanManageUsers {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	var req models.TenantUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Location != nil {
		updates = append(updates, fmt.Sprintf("location = $%d", argIndex))
		args = append(args, *req.Location)
		argIndex++
	}
	if req.CountryCode != nil {
		updates = append(updates, fmt.Sprintf("country_code = $%d", argIndex))
		args = append(args, *req.CountryCode)
		argIndex++
	}
	if req.Currency != nil {
		updates = append(updates, fmt.Sprintf("currency = $%d", argIndex))
		args = append(args, *req.Currency)
		argIndex++
	}
	if req.NumberFormat != nil {
		updates = append(updates, fmt.Sprintf("number_format = $%d", argIndex))
		args = append(args, *req.NumberFormat)
		argIndex++
	}
	if req.DateFormat != nil {
		updates = append(updates, fmt.Sprintf("date_format = $%d", argIndex))
		args = append(args, *req.DateFormat)
		argIndex++
	}
	if req.Capacity != nil {
		updates = append(updates, fmt.Sprintf("capacity = $%d", argIndex))
		args = append(args, *req.Capacity)
		argIndex++
	}

	if len(updates) == 0 {
		respondWithError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, tenantID)

	query := fmt.Sprintf("UPDATE tenants SET %s WHERE id = $%d", strings.Join(updates, ", "), argIndex)
	
	_, err = database.DB.Exec(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update tenant")
		return
	}

	// Return updated tenant
	GetTenant(w, r)
}

// GetTenantItems returns all items for a tenant, optionally filtered by category
func GetTenantItems(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("GetTenantItems: Request received - URL: %s, Method: %s\n", r.URL.String(), r.Method)
	user := middleware.GetUserFromContext(r)
	if user == nil {
		fmt.Printf("GetTenantItems: No user in context - returning 401\n")
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	fmt.Printf("GetTenantItems: User authenticated - UserID: %d, TenantID: %s\n", user.UserID, user.TenantID)

	// Get tenant ID from query or use user's tenant
	tenantIDStr := r.URL.Query().Get("tenant_id")
	var tenantID uuid.UUID
	var err error

	if tenantIDStr != "" {
		tenantID, err = uuid.Parse(tenantIDStr)
		if err != nil {
			fmt.Printf("GetTenantItems: Invalid tenant_id in query: %s, error: %v\n", tenantIDStr, err)
			respondWithError(w, http.StatusBadRequest, "Invalid tenant_id")
			return
		}
	} else {
		fmt.Printf("GetTenantItems: Parsing tenant_id from token: %s (type: %T)\n", user.TenantID, user.TenantID)
		tenantID, err = uuid.Parse(user.TenantID)
		if err != nil {
			fmt.Printf("GetTenantItems: Invalid tenant_id in token: %s, error: %v\n", user.TenantID, err)
			respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
			return
		}
	}
	fmt.Printf("GetTenantItems: Successfully parsed tenant_id: %s\n", tenantID.String())

	// Check if user has access to this tenant
	var hasAccess bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM tenant_users 
			WHERE user_id = $1 AND tenant_id = $2
		)
	`, user.UserID, tenantID).Scan(&hasAccess)

	if err != nil || !hasAccess {
		respondWithError(w, http.StatusForbidden, "Access denied to this tenant")
		return
	}

	// Get category filter if provided
	category := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("category")))
	fmt.Printf("GetTenantItems: Category from query: '%s' (after processing: '%s')\n", r.URL.Query().Get("category"), category)
	
	// Validate category if provided
	if category != "" {
		validCategories := map[string]bool{
			"EGG": true, "FEED": true, "MEDICINE": true, "OTHER": true,
			"CHICK": true, "GROWER": true, "MANURE": true, "EMPLOYEE": true,
		}
		if !validCategories[category] {
			fmt.Printf("GetTenantItems: Invalid category: %s\n", category)
			respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid category: %s. Valid categories are: EGG, FEED, MEDICINE, OTHER, CHICK, GROWER, MANURE, EMPLOYEE", category))
			return
		}
		fmt.Printf("GetTenantItems: Category validated: %s\n", category)
	}
	
	var rows *sql.Rows
	if category != "" {
		// Cast category to category_enum type explicitly
		rows, err = database.DB.Query(`
			SELECT id, tenant_id, category, item_name, display_order, is_active
			FROM tenant_items
			WHERE tenant_id = $1 AND category = $2::category_enum AND is_active = TRUE
			ORDER BY display_order, item_name
		`, tenantID, category)
	} else {
		rows, err = database.DB.Query(`
			SELECT id, tenant_id, category, item_name, display_order, is_active
			FROM tenant_items
			WHERE tenant_id = $1 AND is_active = TRUE
			ORDER BY category, display_order, item_name
		`, tenantID)
	}

	if err != nil {
		fmt.Printf("Error fetching tenant items: %v\n", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch tenant items: %v", err))
		return
	}
	defer rows.Close()

	var items []models.TenantItem
	for rows.Next() {
		var item models.TenantItem
		err := rows.Scan(
			&item.ID, &item.TenantID, &item.Category,
			&item.ItemName, &item.DisplayOrder, &item.IsActive,
		)
		if err != nil {
			fmt.Printf("Error scanning tenant item row: %v\n", err)
			continue
		}
		items = append(items, item)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		fmt.Printf("Error iterating tenant item rows: %v\n", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process tenant items: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    items,
	})
}

