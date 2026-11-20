package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"poultry-farm-api/database"
	"poultry-farm-api/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID   int    `json:"user_id"`
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"` // UUID as string
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware validates JWT token and sets user context
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondWithError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			tokenString := parts[1]
			claims := &Claims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Verify user is active
			var isActive bool
			err = database.DB.QueryRow("SELECT is_active FROM users WHERE id = $1", claims.UserID).Scan(&isActive)
			if err != nil || !isActive {
				respondWithError(w, http.StatusUnauthorized, "User account is inactive")
				return
			}

			// Set user context
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext extracts user claims from request context
func GetUserFromContext(r *http.Request) *Claims {
	user, ok := r.Context().Value(UserContextKey).(*Claims)
	if !ok {
		return nil
	}
	return user
}

// RequireRole checks if user has required role
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r)
			if user == nil {
				respondWithError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			if user.Role != requiredRole {
				respondWithError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole checks if user has any of the required roles
func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r)
			if user == nil {
				respondWithError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			hasRole := false
			for _, role := range roles {
				if user.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				respondWithError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserPermissions retrieves role permissions for the user's tenant
func GetUserPermissions(userID int, tenantID string) (*models.RolePermission, error) {
	var perm models.RolePermission
	var role string

	// Parse tenantID string to UUID
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, err
	}

	// Get user's role and optional per-user overrides for this tenant
	var userSensitiveOverride sql.NullBool
	var customPermsRaw sql.NullString
	err = database.DB.QueryRow(`
		SELECT role, can_view_sensitive_data, custom_permissions::text FROM tenant_users 
		WHERE user_id = $1 AND tenant_id = $2
	`, userID, tenantUUID).Scan(&role, &userSensitiveOverride, &customPermsRaw)
	if err != nil {
		return nil, err
	}

	// Get permissions for this role
	err = database.DB.QueryRow(`
		SELECT id, tenant_id, role, can_view_sensitive_data, can_edit_transactions,
		       can_approve_transactions, can_manage_users, can_view_charts
		FROM role_permissions
		WHERE tenant_id = $1 AND role = $2
	`, tenantUUID, role).Scan(
		&perm.ID, &perm.TenantID, &perm.Role, &perm.CanViewSensitiveData,
		&perm.CanEditTransactions, &perm.CanApproveTransactions,
		&perm.CanManageUsers, &perm.CanViewCharts,
	)

	if err == sql.ErrNoRows {
		def := defaultRolePermissions(tenantUUID, role)
		perm = *def
	} else if err != nil {
		return nil, err
	}

	if userSensitiveOverride.Valid {
		perm.CanViewSensitiveData = userSensitiveOverride.Bool
	}

	// Resolve granular capability list (custom_permissions overrides role defaults if present)
	if customPermsRaw.Valid && strings.TrimSpace(customPermsRaw.String) != "" && strings.TrimSpace(customPermsRaw.String) != "null" {
		var customList []string
		if jsonErr := json.Unmarshal([]byte(customPermsRaw.String), &customList); jsonErr == nil {
			perm.Permissions = customList
			perm.CanViewSensitiveData = HasPermission(customList, "read:profit")
			perm.CanEditTransactions = HasPermission(customList, "manage:expense") || HasPermission(customList, "manage:hen_batch") || HasPermission(customList, "submit:daily_log")
			perm.CanApproveTransactions = HasPermission(customList, "approve:daily_log")
			perm.CanManageUsers = HasPermission(customList, "manage:team")
			perm.CanViewCharts = HasPermission(customList, "read:profit") || HasPermission(customList, "read:expense") || HasPermission(customList, "read:price_history")
			return &perm, nil
		}
	}

	perm.Permissions = DefaultPermissionsForRole(role, perm.CanViewSensitiveData)
	return &perm, nil
}

func defaultRolePermissions(tenantUUID uuid.UUID, role string) *models.RolePermission {
	switch role {
	case "ADMIN", "OWNER", "CO_OWNER":
		return &models.RolePermission{
			TenantID:               tenantUUID,
			Role:                   role,
			CanViewSensitiveData:   true,
			CanEditTransactions:    true,
			CanApproveTransactions: true,
			CanManageUsers:         true,
			CanViewCharts:          true,
		}
	case "MANAGER":
		return &models.RolePermission{
			TenantID:               tenantUUID,
			Role:                   role,
			CanViewSensitiveData:   false,
			CanEditTransactions:    true,
			CanApproveTransactions: true,
			CanManageUsers:         false,
			CanViewCharts:          true,
		}
	case "AUDITOR":
		return &models.RolePermission{
			TenantID:               tenantUUID,
			Role:                   role,
			CanViewSensitiveData:   true,
			CanEditTransactions:    false,
			CanApproveTransactions: false,
			CanManageUsers:         false,
			CanViewCharts:          true,
		}
	case "OTHER_USER":
		return &models.RolePermission{
			TenantID:               tenantUUID,
			Role:                   role,
			CanViewSensitiveData:   false,
			CanEditTransactions:    true,
			CanApproveTransactions: false,
			CanManageUsers:         false,
			CanViewCharts:          false,
		}
	default:
		return &models.RolePermission{
			TenantID: tenantUUID,
			Role:     role,
		}
	}
}

// HasPermission checks whether a permission slice grants `required`.
// Any `manage:<resource>` permission automatically grants `read:<resource>`.
func HasPermission(perms []string, required string) bool {
	req := strings.ToLower(strings.TrimSpace(required))
	if req == "" {
		return false
	}
	impliedManage := ""
	if strings.HasPrefix(req, "read:") {
		impliedManage = "manage:" + strings.TrimPrefix(req, "read:")
	}
	for _, p := range perms {
		norm := strings.ToLower(strings.TrimSpace(p))
		if norm == req || (impliedManage != "" && norm == impliedManage) {
			return true
		}
	}
	return false
}

// DefaultPermissionsForRole returns the canonical read:/manage: capabilities for each farm role.
func DefaultPermissionsForRole(role string, includeProfit bool) []string {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "OWNER":
		return []string{
			"read:profit",
			"manage:expense",
			"manage:hen_batch",
			"manage:price_history",
			"manage:ledger",
			"submit:daily_log",
			"approve:daily_log",
			"manage:team",
			"transfer:ownership",
			"manage:farm_settings",
			"manage:profile",
		}
	case "CO_OWNER", "ADMIN":
		return []string{
			"read:profit",
			"manage:expense",
			"manage:hen_batch",
			"manage:price_history",
			"manage:ledger",
			"submit:daily_log",
			"approve:daily_log",
			"manage:team",
			"manage:farm_settings",
			"manage:profile",
		}
	case "MANAGER":
		perms := []string{
			"manage:expense",
			"manage:hen_batch",
			"manage:price_history",
			"manage:ledger",
			"submit:daily_log",
			"approve:daily_log",
			"read:farm_settings",
			"manage:profile",
		}
		if includeProfit {
			perms = append([]string{"read:profit"}, perms...)
		}
		return perms
	case "AUDITOR":
		return []string{
			"read:profit",
			"read:expense",
			"read:hen_batch",
			"read:price_history",
			"read:ledger",
			"read:team",
			"read:farm_settings",
			"manage:profile",
		}
	case "OTHER_USER":
		return []string{
			"read:hen_batch",
			"submit:daily_log",
			"manage:profile",
		}
	default:
		return []string{"manage:profile"}
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}


