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

	// Get user's role for this tenant
	err = database.DB.QueryRow(`
		SELECT role FROM tenant_users 
		WHERE user_id = $1 AND tenant_id = $2
	`, userID, tenantUUID).Scan(&role)
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
		// Return default permissions (no access)
		return &models.RolePermission{
			TenantID:              tenantUUID,
			Role:                  role,
			CanViewSensitiveData:  false,
			CanEditTransactions:   false,
			CanApproveTransactions: false,
			CanManageUsers:        false,
			CanViewCharts:         false,
		}, nil
	}

	if err != nil {
		return nil, err
	}

	return &perm, nil
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}


