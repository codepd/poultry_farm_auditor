package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"poultry-farm-api/config"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Tenants   []TenantInfo `json:"tenants"`
}

type TenantInfo struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	IsOwner  bool      `json:"is_owner"`
}

// Login handles user authentication
func Login(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Get user from database
		var userID int
		var passwordHash string
		var fullName string
		var isActive bool

		err := database.DB.QueryRow(`
			SELECT id, password_hash, full_name, is_active
			FROM users
			WHERE email = $1
		`, req.Email).Scan(&userID, &passwordHash, &fullName, &isActive)

		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Database error")
			return
		}

		if !isActive {
			respondWithError(w, http.StatusUnauthorized, "Account is inactive")
			return
		}

		// Verify password
		err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		// Get user's tenants
		rows, err := database.DB.Query(`
			SELECT tu.tenant_id, t.name, tu.role, tu.is_owner
			FROM tenant_users tu
			JOIN tenants t ON t.id = tu.tenant_id
			WHERE tu.user_id = $1
		`, userID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch tenants")
			return
		}
		defer rows.Close()

		var tenants []TenantInfo
		for rows.Next() {
			var tenant TenantInfo
			if err := rows.Scan(&tenant.TenantID, &tenant.Name, &tenant.Role, &tenant.IsOwner); err != nil {
				continue
			}
			tenants = append(tenants, tenant)
		}

		if len(tenants) == 0 {
			respondWithError(w, http.StatusForbidden, "User has no tenant access")
			return
		}

		// Use first tenant for token (user can switch tenants later)
		primaryTenant := tenants[0]

		// Generate JWT token
		claims := &middleware.Claims{
			UserID:   userID,
			Email:    req.Email,
			TenantID: primaryTenant.TenantID.String(),
			Role:     primaryTenant.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": LoginResponse{
				Token:    tokenString,
				UserID:   userID,
				Email:    req.Email,
				FullName: fullName,
				Tenants:  tenants,
			},
		})
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

