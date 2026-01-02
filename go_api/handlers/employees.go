package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"poultry-farm-api/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GetEmployees returns all employees for a tenant
func GetEmployees(w http.ResponseWriter, r *http.Request) {
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

	// Optional filter by active status
	isActiveFilter := r.URL.Query().Get("is_active")

	query := `
		SELECT id, tenant_id, full_name, phone, email, address,
		       designation, is_active, created_at, updated_at
		FROM employees
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}

	if isActiveFilter == "true" || isActiveFilter == "false" {
		query += " AND is_active = $2"
		args = append(args, isActiveFilter == "true")
	}

	query += " ORDER BY full_name"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch employees")
		return
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		var emp models.Employee
		var phone, email, address, designation sql.NullString

		err := rows.Scan(
			&emp.ID, &emp.TenantID, &emp.FullName, &phone, &email,
			&address, &designation, &emp.IsActive, &emp.CreatedAt, &emp.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if phone.Valid {
			emp.Phone = phone
		}
		if email.Valid {
			emp.Email = email
		}
		if address.Valid {
			emp.Address = address
		}
		if designation.Valid {
			emp.Designation = designation
		}

		employees = append(employees, emp)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    employees,
	})
}

// GetEmployee returns a single employee
func GetEmployee(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	employeeID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid employee ID")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	var emp models.Employee
	var phone, email, address, designation sql.NullString

	err = database.DB.QueryRow(`
		SELECT id, tenant_id, full_name, phone, email, address,
		       designation, is_active, created_at, updated_at
		FROM employees
		WHERE id = $1 AND tenant_id = $2
	`, employeeID, tenantID).Scan(
		&emp.ID, &emp.TenantID, &emp.FullName, &phone, &email,
		&address, &designation, &emp.IsActive, &emp.CreatedAt, &emp.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Employee not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if phone.Valid {
		emp.Phone = phone
	}
	if email.Valid {
		emp.Email = email
	}
	if address.Valid {
		emp.Address = address
	}
	if designation.Valid {
		emp.Designation = designation
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    emp,
	})
}

type EmployeeCreateRequest struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	FullName    string    `json:"full_name"`
	Phone       *string   `json:"phone,omitempty"`
	Email       *string   `json:"email,omitempty"`
	Address     *string   `json:"address,omitempty"`
	Designation *string   `json:"designation,omitempty"`
}

// CreateEmployee creates a new employee
func CreateEmployee(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req EmployeeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	if req.TenantID == uuid.Nil {
		req.TenantID = tenantID
	}

	// Verify access
	var hasAccess bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM tenant_users 
			WHERE user_id = $1 AND tenant_id = $2
		)
	`, user.UserID, req.TenantID).Scan(&hasAccess)

	if err != nil || !hasAccess {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Insert employee
	var empID int
	err = database.DB.QueryRow(`
		INSERT INTO employees (
			tenant_id, full_name, phone, email, address, designation
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, req.TenantID, req.FullName, req.Phone, req.Email, req.Address, req.Designation).Scan(&empID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create employee")
		return
	}

	// Return created employee
	vars := mux.Vars(r)
	vars["id"] = strconv.Itoa(empID)
	GetEmployee(w, r)
}

type EmployeeUpdateRequest struct {
	FullName    *string `json:"full_name,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Email       *string `json:"email,omitempty"`
	Address     *string `json:"address,omitempty"`
	Designation *string `json:"designation,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// UpdateEmployee updates an existing employee
func UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	employeeID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid employee ID")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id in token")
		return
	}

	var req EmployeeUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build update query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.FullName != nil {
		updates = append(updates, fmt.Sprintf("full_name = $%d", argIndex))
		args = append(args, *req.FullName)
		argIndex++
	}
	if req.Phone != nil {
		updates = append(updates, fmt.Sprintf("phone = $%d", argIndex))
		args = append(args, *req.Phone)
		argIndex++
	}
	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, *req.Email)
		argIndex++
	}
	if req.Address != nil {
		updates = append(updates, fmt.Sprintf("address = $%d", argIndex))
		args = append(args, *req.Address)
		argIndex++
	}
	if req.Designation != nil {
		updates = append(updates, fmt.Sprintf("designation = $%d", argIndex))
		args = append(args, *req.Designation)
		argIndex++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(updates) == 0 {
		respondWithError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, employeeID, tenantID)

	query := fmt.Sprintf("UPDATE employees SET %s WHERE id = $%d AND tenant_id = $%d",
		strings.Join(updates, ", "), argIndex, argIndex+1)

	_, err = database.DB.Exec(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update employee")
		return
	}

	GetEmployee(w, r)
}


