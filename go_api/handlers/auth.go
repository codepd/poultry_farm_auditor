package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"poultry-farm-api/config"
	"poultry-farm-api/database"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

type LoginResponse struct {
	Token             string       `json:"token"`
	UserID            int          `json:"user_id"`
	Email             string       `json:"email"`
	Phone             string       `json:"phone,omitempty"`
	FullName          string       `json:"full_name"`
	PreferredLanguage string       `json:"preferred_language"`
	MustSetPassword   bool         `json:"must_set_password"`
	Tenants           []TenantInfo `json:"tenants"`
}

type TenantInfo struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	IsOwner  bool      `json:"is_owner"`
}

// Login handles user authentication via Email/Username or Phone + Password
func Login(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		identifier := strings.TrimSpace(req.Email)
		if identifier == "" {
			identifier = strings.TrimSpace(req.Phone)
		}
		if identifier == "" || strings.TrimSpace(req.Password) == "" {
			respondWithError(w, http.StatusBadRequest, "Email/Phone and password are required")
			return
		}

		// Get user from database by email OR phone
		var userID int
		var dbEmail sql.NullString
		var dbPhone sql.NullString
		var passwordHash sql.NullString
		var fullName sql.NullString
		var preferredLanguage string
		var mustSetPassword bool
		var isActive bool

		err := database.DB.QueryRow(`
			SELECT id, email, phone, password_hash, full_name, COALESCE(preferred_language, 'en'), COALESCE(must_set_password, FALSE), is_active
			FROM users
			WHERE email = $1 OR phone = $1
			LIMIT 1
		`, identifier).Scan(&userID, &dbEmail, &dbPhone, &passwordHash, &fullName, &preferredLanguage, &mustSetPassword, &isActive)

		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
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

		if !passwordHash.Valid || strings.TrimSpace(passwordHash.String) == "" {
			respondWithError(w, http.StatusUnauthorized, "Password not set yet. Please use your WhatsApp Magic Link to set a password first.")
			return
		}

		// Verify password
		err = bcrypt.CompareHashAndPassword([]byte(passwordHash.String), []byte(req.Password))
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
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
		noRememberTTL, rememberTTL := getTenantRefreshDurations(primaryTenant.TenantID)
		refreshTTL := noRememberTTL
		if req.RememberMe {
			refreshTTL = rememberTTL
		}
		refreshExpiresAt := time.Now().UTC().Add(refreshTTL)

		emailStr := ""
		if dbEmail.Valid {
			emailStr = dbEmail.String
		}
		phoneStr := ""
		if dbPhone.Valid {
			phoneStr = dbPhone.String
		}

		tokenString, err := issueAccessToken(cfg, userID, emailStr, primaryTenant.TenantID, primaryTenant.Role)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		refreshToken, err := createRefreshSession(
			userID,
			primaryTenant.TenantID,
			req.RememberMe,
			refreshExpiresAt,
			strings.TrimSpace(r.UserAgent()),
			strings.TrimSpace(r.RemoteAddr),
			DeviceIDFromRequest(r),
		)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create session")
			return
		}
		setRefreshTokenCookie(w, cfg, refreshToken, refreshExpiresAt)

		fullNameStr := ""
		if fullName.Valid {
			fullNameStr = fullName.String
		}
		if preferredLanguage == "" {
			preferredLanguage = "en"
		}

		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": LoginResponse{
				Token:             tokenString,
				UserID:            userID,
				Email:             emailStr,
				Phone:             phoneStr,
				FullName:          fullNameStr,
				PreferredLanguage: preferredLanguage,
				MustSetPassword:   mustSetPassword,
				Tenants:           tenants,
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
