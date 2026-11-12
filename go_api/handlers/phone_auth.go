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

		// Encrypt OTP with AES-256-GCM (key resolved from AWS Secrets Manager / SSM SecureString)
		// and compute HMAC-SHA256 digest for lookup so plaintext OTP is never stored or logged.
		encryptedOTP, keySource, encErr := EncryptOTP(otp, cfg)
		if encErr != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to encrypt OTP")
			return
		}
		hashedOTP := HashOTPForLookup(phone, otp, cfg)

		// Store hashed & encrypted OTP (expires in 5 minutes) — use UTC to match PostgreSQL's NOW()
		expiresAt := time.Now().UTC().Add(5 * time.Minute)
		_, err = database.DB.Exec(`
			INSERT INTO otp_codes (phone, code, encrypted_code, tenant_id, expires_at)
			VALUES ($1, $2, $3, $4, $5)
		`, phone, hashedOTP, encryptedOTP, allowedTenantID, expiresAt)
		if err != nil {
			// Fallback if encrypted_code column is not yet migrated
			_, err = database.DB.Exec(`
				INSERT INTO otp_codes (phone, code, tenant_id, expires_at)
				VALUES ($1, $2, $3, $4)
			`, phone, hashedOTP, allowedTenantID, expiresAt)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to store OTP")
				return
			}
		}

		// Never log plaintext OTP! Log only the AES-256-GCM encrypted token and key vault source.
		log.Printf("🔒 [OTP] Phone: %s | EncryptedOTP: %s | KeyVault: %s | Tenant: %s", phone, encryptedOTP, keySource, tenantName)

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

		hashedCandidate := HashOTPForLookup(phone, code, cfg)

		// Find valid OTP (matches HMAC-SHA256 digest or legacy plaintext)
		var otpID int
		var tenantID uuid.UUID
		err := database.DB.QueryRow(`
			SELECT id, tenant_id FROM otp_codes
			WHERE phone = $1 AND (code = $2 OR code = $3) AND verified = FALSE AND expires_at > NOW()
			ORDER BY created_at DESC LIMIT 1
		`, phone, hashedCandidate, code).Scan(&otpID, &tenantID)

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
		var passwordHash sql.NullString
		var preferredLanguage string
		var mustSetPassword bool
		var isActive bool

		err = database.DB.QueryRow(
			"SELECT id, email, full_name, password_hash, COALESCE(preferred_language, 'en'), COALESCE(must_set_password, FALSE), is_active FROM users WHERE phone = $1", phone,
		).Scan(&userID, &email, &fullName, &passwordHash, &preferredLanguage, &mustSetPassword, &isActive)

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
				INSERT INTO users (phone, is_active, must_set_password) VALUES ($1, TRUE, TRUE) RETURNING id
			`, phone).Scan(&userID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to create user")
				return
			}
			isActive = true
			mustSetPassword = true

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

		if !passwordHash.Valid || strings.TrimSpace(passwordHash.String) == "" {
			mustSetPassword = true
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

		if preferredLanguage == "" {
			preferredLanguage = "en"
		}

		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": LoginResponse{
				Token:             tokenString,
				UserID:            userID,
				Email:             emailStr,
				Phone:             phone,
				FullName:          fullNameStr,
				PreferredLanguage: preferredLanguage,
				MustSetPassword:   mustSetPassword,
				Tenants:           tenants,
			},
		})
	}
}

type MagicLinkLoginRequest struct {
	Token      string `json:"token"`
	RememberMe bool   `json:"remember_me"`
}

// MagicLinkLogin logs a user in immediately using a single-use WhatsApp Magic Link token
// and sets MustSetPassword = true so the user is forced to set a password right away.
func MagicLinkLogin(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MagicLinkLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		token := strings.TrimSpace(req.Token)
		if token == "" {
			respondWithError(w, http.StatusBadRequest, "Magic login token is required")
			return
		}

		var inviteID int
		var tenantID uuid.UUID
		var invEmail sql.NullString
		var invPhone sql.NullString
		var role string
		var expiresAt time.Time
		var acceptedAt sql.NullTime

		err := database.DB.QueryRow(`
			SELECT id, tenant_id, email, phone, role, expires_at, accepted_at
			FROM invitations
			WHERE token = $1
		`, token).Scan(&inviteID, &tenantID, &invEmail, &invPhone, &role, &expiresAt, &acceptedAt)

		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired WhatsApp Magic Link")
			return
		}
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Database error")
			return
		}

		if acceptedAt.Valid {
			respondWithError(w, http.StatusBadRequest, "This WhatsApp Magic Link has already been used. Please log in with your password.")
			return
		}

		if time.Now().UTC().After(expiresAt.UTC()) {
			respondWithError(w, http.StatusBadRequest, "This WhatsApp Magic Link has expired. Ask your Farm Admin for a new link.")
			return
		}

		emailVal := ""
		if invEmail.Valid {
			emailVal = strings.TrimSpace(invEmail.String)
		}
		phoneVal := ""
		if invPhone.Valid {
			phoneVal = strings.TrimSpace(invPhone.String)
		}

		var userID int
		var dbEmail sql.NullString
		var dbPhone sql.NullString
		var fullName sql.NullString
		var preferredLanguage string

		if phoneVal != "" {
			err = database.DB.QueryRow(`
				SELECT id, email, phone, full_name, COALESCE(preferred_language, 'en')
				FROM users WHERE phone = $1
			`, phoneVal).Scan(&userID, &dbEmail, &dbPhone, &fullName, &preferredLanguage)
		} else if emailVal != "" {
			err = database.DB.QueryRow(`
				SELECT id, email, phone, full_name, COALESCE(preferred_language, 'en')
				FROM users WHERE email = $1
			`, emailVal).Scan(&userID, &dbEmail, &dbPhone, &fullName, &preferredLanguage)
		} else {
			respondWithError(w, http.StatusBadRequest, "Invalid invitation target")
			return
		}

		if err == sql.ErrNoRows {
			err = database.DB.QueryRow(`
				INSERT INTO users (email, phone, is_active, must_set_password)
				VALUES (NULLIF($1, ''), NULLIF($2, ''), TRUE, TRUE)
				RETURNING id
			`, emailVal, phoneVal).Scan(&userID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to create user account")
				return
			}
			preferredLanguage = "en"
		} else if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Database error")
			return
		} else {
			_, _ = database.DB.Exec(`
				UPDATE users SET is_active = TRUE, must_set_password = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = $1
			`, userID)
		}

		_, err = database.DB.Exec(`
			INSERT INTO tenant_users (tenant_id, user_id, role, is_owner)
			VALUES ($1, $2, $3, FALSE)
			ON CONFLICT (tenant_id, user_id) DO UPDATE SET role = $3, updated_at = CURRENT_TIMESTAMP
		`, tenantID, userID, role)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to assign tenant role")
			return
		}

		_, _ = database.DB.Exec(`UPDATE invitations SET accepted_at = NOW() WHERE id = $1`, inviteID)

		// Fetch user's tenants
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
			if err := rows.Scan(&tenant.TenantID, &tenant.Name, &tenant.Role, &tenant.IsOwner); err == nil {
				tenants = append(tenants, tenant)
			}
		}

		if len(tenants) == 0 {
			respondWithError(w, http.StatusForbidden, "User has no tenant access")
			return
		}

		primaryTenant := tenants[0]
		emailStr := emailVal
		if dbEmail.Valid && dbEmail.String != "" {
			emailStr = dbEmail.String
		}
		phoneStr := phoneVal
		if dbPhone.Valid && dbPhone.String != "" {
			phoneStr = dbPhone.String
		}
		fullNameStr := ""
		if fullName.Valid {
			fullNameStr = fullName.String
		}

		noRememberTTL, rememberTTL := getTenantRefreshDurations(primaryTenant.TenantID)
		refreshTTL := rememberTTL
		if !req.RememberMe {
			refreshTTL = noRememberTTL
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
			true,
			refreshExpiresAt,
			strings.TrimSpace(r.UserAgent()),
			strings.TrimSpace(r.RemoteAddr),
			DeviceIDFromRequest(r),
		)
		if err == nil {
			setRefreshTokenCookie(w, cfg, refreshToken, refreshExpiresAt)
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
				MustSetPassword:   true,
				Tenants:           tenants,
			},
		})
	}
}
