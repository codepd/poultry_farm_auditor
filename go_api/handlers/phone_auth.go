package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"poultry-farm-api/config"
	"poultry-farm-api/database"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var phoneRegex = regexp.MustCompile(`^\+\d{1,4}\d{6,14}$`)

type SendOTPRequest struct {
	Phone       string `json:"phone"`
	CountryCode string `json:"country_code,omitempty"`
	TenantID    string `json:"tenant_id,omitempty"`
}

type VerifyOTPRequest struct {
	Phone      string `json:"phone"`
	Code       string `json:"code"`
	RememberMe bool   `json:"remember_me"`
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

func extractCountryCode(phone string) string {
	if !strings.HasPrefix(phone, "+") {
		return ""
	}
	digits := phone[1:]
	// Try longest country codes first (up to 4 digits)
	for length := 4; length >= 1; length-- {
		if len(digits) > length {
			return "+" + digits[:length]
		}
	}
	return ""
}

// SendOTP generates an OTP, validates the country code against tenant config, and stores it.
// In production, this would integrate with an SMS provider (Twilio, MSG91, AWS SNS).
func SendOTP(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SendOTPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		phone := strings.TrimSpace(req.Phone)
		if !phoneRegex.MatchString(phone) {
			respondWithError(w, http.StatusBadRequest, "Invalid phone number format. Use international format: +<country_code><number>")
			return
		}

		countryCode := strings.TrimSpace(req.CountryCode)
		if countryCode == "" {
			countryCode = extractCountryCode(phone)
		}

		// Validate country code against allowed codes
		var allowedTenantID uuid.UUID
		var tenantName string

		if req.TenantID != "" {
			// Validate against specific tenant
			tid, err := uuid.Parse(req.TenantID)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid tenant_id")
				return
			}
			var exists bool
			err = database.DB.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM tenant_country_codes
					WHERE tenant_id = $1 AND country_code = $2
				)`, tid, countryCode).Scan(&exists)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Database error")
				return
			}
			if !exists {
				respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Country code %s is not allowed for this tenant", countryCode))
				return
			}
			allowedTenantID = tid
			database.DB.QueryRow("SELECT name FROM tenants WHERE id = $1", tid).Scan(&tenantName)
		} else {
			// Check if user already exists — look up their tenant
			var userID int
			err := database.DB.QueryRow("SELECT id FROM users WHERE phone = $1", phone).Scan(&userID)
			if err == nil {
				// User exists, find their tenant and validate country code
				err = database.DB.QueryRow(`
					SELECT t.id, t.name FROM tenant_users tu
					JOIN tenants t ON t.id = tu.tenant_id
					JOIN tenant_country_codes tcc ON tcc.tenant_id = t.id AND tcc.country_code = $2
					WHERE tu.user_id = $1 LIMIT 1
				`, userID, countryCode).Scan(&allowedTenantID, &tenantName)
				if err != nil {
					respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Country code %s is not allowed or user has no tenant access", countryCode))
					return
				}
			} else {
				// New user — must have a pending invitation
				err = database.DB.QueryRow(`
					SELECT t.id, t.name FROM invitations i
					JOIN tenants t ON t.id = i.tenant_id
					WHERE i.phone = $1 AND i.accepted_at IS NULL AND i.expires_at > NOW()
					LIMIT 1
				`, phone).Scan(&allowedTenantID, &tenantName)
				if err != nil {
					respondWithError(w, http.StatusBadRequest, "This phone number has not been invited. Contact your farm admin.")
					return
				}
			}
		}

		otp, err := generateOTP()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to generate OTP")
			return
		}

		// Invalidate previous unused OTPs for this phone
		database.DB.Exec("UPDATE otp_codes SET verified = TRUE WHERE phone = $1 AND verified = FALSE", phone)

		// Store OTP (expires in 5 minutes) — use UTC to match PostgreSQL's NOW()
		expiresAt := time.Now().UTC().Add(5 * time.Minute)
		_, err = database.DB.Exec(`
			INSERT INTO otp_codes (phone, code, tenant_id, expires_at)
			VALUES ($1, $2, $3, $4)
		`, phone, otp, allowedTenantID, expiresAt)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to store OTP")
			return
		}

		// TODO: Send OTP via SMS provider (Twilio, MSG91, AWS SNS)
		// For now, log the OTP for development
		log.Printf("📱 OTP for %s: %s (tenant: %s)", phone, otp, tenantName)

		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success":    true,
			"message":    "OTP sent successfully",
			"phone":      phone,
			"expires_in": 300,
		})
	}
}

// VerifyOTP validates the OTP and issues a JWT token. Creates the user if new.
func VerifyOTP(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req VerifyOTPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		phone := strings.TrimSpace(req.Phone)
		code := strings.TrimSpace(req.Code)

		if phone == "" || code == "" {
			respondWithError(w, http.StatusBadRequest, "Phone and code are required")
			return
		}

		// Find valid OTP
		var otpID int
		var tenantID uuid.UUID
		err := database.DB.QueryRow(`
			SELECT id, tenant_id FROM otp_codes
			WHERE phone = $1 AND code = $2 AND verified = FALSE AND expires_at > NOW()
			ORDER BY created_at DESC LIMIT 1
		`, phone, code).Scan(&otpID, &tenantID)

		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired OTP")
			return
		}
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Database error")
			return
		}

		// Mark OTP as verified
		database.DB.Exec("UPDATE otp_codes SET verified = TRUE WHERE id = $1", otpID)

		// Find or create user
		var userID int
		var email sql.NullString
		var fullName sql.NullString
		var isActive bool

		err = database.DB.QueryRow(
			"SELECT id, email, full_name, is_active FROM users WHERE phone = $1", phone,
		).Scan(&userID, &email, &fullName, &isActive)

		if err == sql.ErrNoRows {
			// New user — look up invitation for role
			var inviteID int
			var inviteRole string
			var inviteTenantID uuid.UUID
			invErr := database.DB.QueryRow(`
				SELECT id, role, tenant_id FROM invitations
				WHERE phone = $1 AND accepted_at IS NULL AND expires_at > NOW()
				ORDER BY created_at DESC LIMIT 1
			`, phone).Scan(&inviteID, &inviteRole, &inviteTenantID)
			if invErr != nil {
				respondWithError(w, http.StatusForbidden, "No valid invitation found for this phone number")
				return
			}

			err = database.DB.QueryRow(`
				INSERT INTO users (phone, is_active) VALUES ($1, TRUE) RETURNING id
			`, phone).Scan(&userID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to create user")
				return
			}
			isActive = true

			_, err = database.DB.Exec(`
				INSERT INTO tenant_users (tenant_id, user_id, role, is_owner)
				VALUES ($1, $2, $3, FALSE)
				ON CONFLICT (tenant_id, user_id) DO UPDATE SET role = $3, updated_at = CURRENT_TIMESTAMP
			`, inviteTenantID, userID, inviteRole)
			if err != nil {
				log.Printf("Failed to assign user to tenant: %v", err)
			}

			// Mark invitation as accepted
			database.DB.Exec("UPDATE invitations SET accepted_at = NOW() WHERE id = $1", inviteID)
			log.Printf("User %d created from invitation %d with role %s", userID, inviteID, inviteRole)
		} else if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Database error")
			return
		}

		// Check for pending invitations that may upgrade an existing user's role
		var pendingInviteID int
		var pendingRole string
		var pendingTenantID uuid.UUID
		invErr := database.DB.QueryRow(`
			SELECT id, role, tenant_id FROM invitations
			WHERE phone = $1 AND accepted_at IS NULL AND expires_at > NOW()
			ORDER BY created_at DESC LIMIT 1
		`, phone).Scan(&pendingInviteID, &pendingRole, &pendingTenantID)
		if invErr == nil {
			_, _ = database.DB.Exec(`
				INSERT INTO tenant_users (tenant_id, user_id, role, is_owner)
				VALUES ($1, $2, $3, FALSE)
				ON CONFLICT (tenant_id, user_id) DO UPDATE SET role = $3, updated_at = CURRENT_TIMESTAMP
			`, pendingTenantID, userID, pendingRole)
			database.DB.Exec("UPDATE invitations SET accepted_at = NOW() WHERE id = $1", pendingInviteID)
			log.Printf("Applied pending invitation %d: user %d upgraded to %s", pendingInviteID, userID, pendingRole)
		}

		if !isActive {
			respondWithError(w, http.StatusUnauthorized, "Account is inactive")
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

		primaryTenant := tenants[0]

		emailStr := ""
		if email.Valid {
			emailStr = email.String
		}
		fullNameStr := ""
		if fullName.Valid {
			fullNameStr = fullName.String
		}

		noRememberTTL, rememberTTL := getTenantRefreshDurations(primaryTenant.TenantID)
		refreshTTL := noRememberTTL
		if req.RememberMe {
			refreshTTL = rememberTTL
		}
		refreshExpiresAt := time.Now().UTC().Add(refreshTTL)

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

		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": LoginResponse{
				Token:    tokenString,
				UserID:   userID,
				Email:    emailStr,
				FullName: fullNameStr,
				Tenants:  tenants,
			},
		})
	}
}
