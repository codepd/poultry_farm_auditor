package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"poultry-farm-api/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// GetUsers returns all users for a tenant
func GetUsers(w http.ResponseWriter, r *http.Request) {
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

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanManageUsers {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	rows, err := database.DB.Query(`
		SELECT u.id, COALESCE(u.email, ''), COALESCE(u.phone, ''), COALESCE(u.full_name, ''),
		       u.is_active, u.created_at, u.updated_at, tu.role, tu.is_owner
		FROM users u
		JOIN tenant_users tu ON tu.user_id = u.id
		WHERE tu.tenant_id = $1
		ORDER BY u.full_name, u.email
	`, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	defer rows.Close()

	type UserResponse struct {
		ID        int       `json:"id"`
		Email     string    `json:"email"`
		Phone     string    `json:"phone"`
		FullName  string    `json:"full_name"`
		IsActive  bool      `json:"is_active"`
		Role      string    `json:"role"`
		IsOwner   bool      `json:"is_owner"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	var users []UserResponse
	for rows.Next() {
		var u UserResponse
		err := rows.Scan(
			&u.ID, &u.Email, &u.Phone, &u.FullName, &u.IsActive,
			&u.CreatedAt, &u.UpdatedAt, &u.Role, &u.IsOwner,
		)
		if err != nil {
			continue
		}
		users = append(users, u)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    users,
	})
}

type InviteUserRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Role  string `json:"role"`
}

// InviteUser creates an invitation for a new user
func InviteUser(w http.ResponseWriter, r *http.Request) {
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

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanManageUsers {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	var req InviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	email := strings.TrimSpace(req.Email)
	phone := strings.TrimSpace(req.Phone)

	if email == "" && phone == "" {
		respondWithError(w, http.StatusBadRequest, "Either email or phone is required")
		return
	}

	// Validate role
	validRoles := map[string]bool{
		"ADMIN": true, "OWNER": true, "CO_OWNER": true,
		"OTHER_USER": true, "AUDITOR": true, "MANAGER": true,
	}
	if !validRoles[req.Role] {
		respondWithError(w, http.StatusBadRequest, "Invalid role")
		return
	}

	isPhoneInvite := phone != "" && email == ""

	// Check if user already has access to this tenant
	var existingUserID int
	var userExists bool
	if isPhoneInvite {
		err = database.DB.QueryRow("SELECT id FROM users WHERE phone = $1", phone).Scan(&existingUserID)
	} else {
		err = database.DB.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&existingUserID)
	}
	if err == nil {
		userExists = true
		var hasAccess bool
		database.DB.QueryRow(`
			SELECT EXISTS(SELECT 1 FROM tenant_users WHERE user_id = $1 AND tenant_id = $2)
		`, existingUserID, tenantID).Scan(&hasAccess)
		if hasAccess {
			respondWithError(w, http.StatusBadRequest, "User already has access to this tenant")
			return
		}
	} else if err != sql.ErrNoRows {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Check for existing pending invitation
	var pendingExists bool
	if isPhoneInvite {
		database.DB.QueryRow(`
			SELECT EXISTS(SELECT 1 FROM invitations WHERE phone = $1 AND tenant_id = $2 AND accepted_at IS NULL AND expires_at > NOW())
		`, phone, tenantID).Scan(&pendingExists)
	} else {
		database.DB.QueryRow(`
			SELECT EXISTS(SELECT 1 FROM invitations WHERE email = $1 AND tenant_id = $2 AND accepted_at IS NULL AND expires_at > NOW())
		`, email, tenantID).Scan(&pendingExists)
	}
	if pendingExists {
		respondWithError(w, http.StatusBadRequest, "A pending invitation already exists")
		return
	}

	// For email invites with no existing user, create an inactive user stub
	if !isPhoneInvite && !userExists {
		err = database.DB.QueryRow(`
			INSERT INTO users (email, is_active) VALUES ($1, FALSE) RETURNING id
		`, email).Scan(&existingUserID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}
	}

	// Generate invitation token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)

	// Create invitation
	var inviteID int
	err = database.DB.QueryRow(`
		INSERT INTO invitations (tenant_id, invited_by_user_id, email, phone, role, token, expires_at)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7)
		RETURNING id
	`, tenantID, user.UserID, email, phone, req.Role, token, expiresAt).Scan(&inviteID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create invitation")
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:4300"
	}

	magicLoginLink := fmt.Sprintf("%s/login?magic_token=%s", frontendURL, token)
	invitationLink := fmt.Sprintf("%s/accept-invite?token=%s", frontendURL, token)

	responseData := map[string]interface{}{
		"invitation_id":    inviteID,
		"role":             req.Role,
		"expires_at":       expiresAt,
		"token":            token,
		"invitation_link":  invitationLink,
		"magic_login_link": magicLoginLink,
	}

	if isPhoneInvite {
		responseData["phone"] = phone
		responseData["message"] = "Invitation created. Share the WhatsApp Magic Link with the user."
	} else {
		responseData["email"] = email
		responseData["message"] = "Invitation created. Share the Magic Link with the user."
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Invitation created",
		"data":    responseData,
	})
}

// GenerateUserMagicLink creates a fresh single-use WhatsApp Magic Link for an existing team member
// so they can log in immediately and set/reset their password without needing SMS OTP.
func GenerateUserMagicLink(w http.ResponseWriter, r *http.Request) {
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

	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanManageUsers {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	var req struct {
		UserID int `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID <= 0 {
		respondWithError(w, http.StatusBadRequest, "Valid user_id is required")
		return
	}

	var targetEmail, targetPhone, targetRole string
	err = database.DB.QueryRow(`
		SELECT COALESCE(u.email, ''), COALESCE(u.phone, ''), tu.role
		FROM users u
		JOIN tenant_users tu ON tu.user_id = u.id
		WHERE u.id = $1 AND tu.tenant_id = $2
	`, req.UserID, tenantID).Scan(&targetEmail, &targetPhone, &targetRole)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Team member not found in this tenant")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().UTC().Add(48 * time.Hour)

	var inviteID int
	err = database.DB.QueryRow(`
		INSERT INTO invitations (tenant_id, invited_by_user_id, email, phone, role, token, expires_at)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7)
		RETURNING id
	`, tenantID, user.UserID, targetEmail, targetPhone, targetRole, token, expiresAt).Scan(&inviteID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate magic login link")
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:4300"
	}
	magicLoginLink := fmt.Sprintf("%s/login?magic_token=%s", frontendURL, token)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Magic login link generated",
		"data": map[string]interface{}{
			"invitation_id":    inviteID,
			"token":            token,
			"magic_login_link": magicLoginLink,
			"phone":            targetPhone,
			"email":            targetEmail,
			"expires_at":       expiresAt,
		},
	})
}

type AcceptInviteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// AcceptInvite accepts an invitation and activates user account
func AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Find invitation
	var invite models.Invitation
	err := database.DB.QueryRow(`
		SELECT id, tenant_id, invited_by_user_id, email, phone, role, token, expires_at, accepted_at
		FROM invitations
		WHERE token = $1
	`, req.Token).Scan(
		&invite.ID, &invite.TenantID, &invite.InvitedByUserID,
		&invite.Email, &invite.Phone, &invite.Role, &invite.Token, &invite.ExpiresAt, &invite.AcceptedAt,
	)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Invalid invitation token")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Check if already accepted
	if invite.AcceptedAt != nil {
		respondWithError(w, http.StatusBadRequest, "Invitation already accepted")
		return
	}

	// Check if expired
	if time.Now().After(invite.ExpiresAt) {
		respondWithError(w, http.StatusBadRequest, "Invitation has expired")
		return
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Get or create user by email or phone
	inviteEmail := ""
	if invite.Email.Valid {
		inviteEmail = strings.TrimSpace(invite.Email.String)
	}
	invitePhone := ""
	if invite.Phone.Valid {
		invitePhone = strings.TrimSpace(invite.Phone.String)
	}

	var userID int
	if invitePhone != "" {
		err = database.DB.QueryRow("SELECT id FROM users WHERE phone = $1", invitePhone).Scan(&userID)
	} else {
		err = database.DB.QueryRow("SELECT id FROM users WHERE email = $1", inviteEmail).Scan(&userID)
	}

	if err == sql.ErrNoRows {
		err = database.DB.QueryRow(`
			INSERT INTO users (email, phone, password_hash, full_name, is_active, must_set_password)
			VALUES (NULLIF($1, ''), NULLIF($2, ''), $3, $4, TRUE, FALSE)
			RETURNING id
		`, inviteEmail, invitePhone, passwordHash, req.FullName).Scan(&userID)
	} else if err == nil {
		_, err = database.DB.Exec(`
			UPDATE users
			SET password_hash = $1, full_name = $2, is_active = TRUE, must_set_password = FALSE, updated_at = CURRENT_TIMESTAMP
			WHERE id = $3
		`, passwordHash, req.FullName, userID)
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to activate user")
		return
	}

	// Add user to tenant
	_, err = database.DB.Exec(`
		INSERT INTO tenant_users (tenant_id, user_id, role, is_owner)
		VALUES ($1, $2, $3, FALSE)
		ON CONFLICT (tenant_id, user_id) DO UPDATE
		SET role = $3, updated_at = CURRENT_TIMESTAMP
	`, invite.TenantID, userID, invite.Role)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to add user to tenant")
		return
	}

	// Mark invitation as accepted
	_, err = database.DB.Exec(`
		UPDATE invitations
		SET accepted_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, invite.ID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update invitation")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Invitation accepted successfully",
	})
}

// GetProfile returns the authenticated user's profile (including preferred_language and must_set_password)
func GetProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var email, phone, fullName, preferredLanguage string
	var mustSetPassword bool
	err := database.DB.QueryRow(`
		SELECT COALESCE(email, ''), COALESCE(phone, ''), COALESCE(full_name, ''), COALESCE(preferred_language, 'en'), COALESCE(must_set_password, FALSE)
		FROM users
		WHERE id = $1
	`, user.UserID).Scan(&email, &phone, &fullName, &preferredLanguage, &mustSetPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load profile")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"user_id":            user.UserID,
			"email":              email,
			"phone":              phone,
			"full_name":          fullName,
			"preferred_language": preferredLanguage,
			"must_set_password":  mustSetPassword,
		},
	})
}

// UpdateProfile lets a user update their own profile (full_name and/or preferred_language)
func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req struct {
		FullName          *string `json:"full_name"`
		PreferredLanguage *string `json:"preferred_language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.FullName == nil && req.PreferredLanguage == nil {
		respondWithError(w, http.StatusBadRequest, "At least one of full_name or preferred_language is required")
		return
	}

	if req.FullName != nil {
		fullName := strings.TrimSpace(*req.FullName)
		if fullName == "" {
			respondWithError(w, http.StatusBadRequest, "Full name cannot be empty")
			return
		}
		if _, err := database.DB.Exec(`
			UPDATE users SET full_name = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2
		`, fullName, user.UserID); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to update full name")
			return
		}
	}

	if req.PreferredLanguage != nil {
		lang := strings.ToLower(strings.TrimSpace(*req.PreferredLanguage))
		if lang != "en" && lang != "ta" {
			respondWithError(w, http.StatusBadRequest, "Unsupported language code (allowed: en, ta)")
			return
		}
		if _, err := database.DB.Exec(`
			UPDATE users SET preferred_language = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2
		`, lang, user.UserID); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to update preferred language")
			return
		}
	}

	var updatedFullName, updatedLang string
	_ = database.DB.QueryRow(`
		SELECT COALESCE(full_name, ''), COALESCE(preferred_language, 'en')
		FROM users
		WHERE id = $1
	`, user.UserID).Scan(&updatedFullName, &updatedLang)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Profile updated",
		"data": map[string]interface{}{
			"full_name":          updatedFullName,
			"preferred_language": updatedLang,
		},
	})
}

type ChangePasswordRequest struct {
	CurrentPassword    string `json:"current_password"`
	NewPassword        string `json:"new_password"`
	LogoutOtherDevices *bool  `json:"logout_other_devices,omitempty"`
}

// ChangePassword lets a user change/reset their own password and optionally logout other devices.
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	newPassword := strings.TrimSpace(req.NewPassword)
	if len(newPassword) < 6 {
		respondWithError(w, http.StatusBadRequest, "New password must be at least 6 characters")
		return
	}

	var existingPasswordHash sql.NullString
	var mustSetPassword bool
	err := database.DB.QueryRow(`
		SELECT password_hash, COALESCE(must_set_password, FALSE)
		FROM users
		WHERE id = $1
	`, user.UserID).Scan(&existingPasswordHash, &mustSetPassword)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusUnauthorized, "User not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to validate user")
		return
	}

	if !mustSetPassword && existingPasswordHash.Valid && strings.TrimSpace(existingPasswordHash.String) != "" {
		if strings.TrimSpace(req.CurrentPassword) == "" {
			respondWithError(w, http.StatusBadRequest, "Current password is required")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(existingPasswordHash.String), []byte(req.CurrentPassword)); err != nil {
			respondWithError(w, http.StatusUnauthorized, "Current password is incorrect")
			return
		}
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to secure password")
		return
	}

	logoutOtherDevices := true
	if req.LogoutOtherDevices != nil {
		logoutOtherDevices = *req.LogoutOtherDevices
	}

	tx, err := database.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to change password")
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE users
		SET password_hash = $1, must_set_password = FALSE, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, string(newPasswordHash), user.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	revokedSessions := int64(0)
	if logoutOtherDevices {
		currentTokenHash := ""
		if cookie, cookieErr := r.Cookie(refreshTokenCookieName); cookieErr == nil && strings.TrimSpace(cookie.Value) != "" {
			currentTokenHash = hashToken(cookie.Value)
		}

		var result sql.Result
		if currentTokenHash != "" {
			result, err = tx.Exec(`
				UPDATE auth_sessions
				SET revoked_at = NOW(), revoked_reason = 'password_changed'
				WHERE user_id = $1 AND revoked_at IS NULL AND refresh_token_hash <> $2
			`, user.UserID, currentTokenHash)
		} else {
			result, err = tx.Exec(`
				UPDATE auth_sessions
				SET revoked_at = NOW(), revoked_reason = 'password_changed'
				WHERE user_id = $1 AND revoked_at IS NULL
			`, user.UserID)
		}
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to revoke other sessions")
			return
		}
		revokedSessions, _ = result.RowsAffected()
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to change password")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Password changed successfully",
		"data": map[string]interface{}{
			"logout_other_devices": logoutOtherDevices,
			"revoked_sessions":     revokedSessions,
		},
	})
}

// GetInvitations returns pending invitations for a tenant
func GetInvitations(w http.ResponseWriter, r *http.Request) {
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

	// Check permissions
	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanManageUsers {
		respondWithError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, tenant_id, invited_by_user_id, email, phone, role, token,
		       expires_at, accepted_at, created_at
		FROM invitations
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch invitations")
		return
	}
	defer rows.Close()

	var invitations []models.Invitation
	for rows.Next() {
		var inv models.Invitation
		err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.InvitedByUserID, &inv.Email, &inv.Phone,
			&inv.Role, &inv.Token, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt,
		)
		if err != nil {
			continue
		}
		invitations = append(invitations, inv)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    invitations,
	})
}

// LogoutOtherDevices revokes all refresh sessions except current device session.
func LogoutOtherDevices(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	currentTokenHash := ""
	if cookie, err := r.Cookie(refreshTokenCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		currentTokenHash = hashToken(cookie.Value)
	}

	var result sql.Result
	var err error
	if currentTokenHash != "" {
		result, err = database.DB.Exec(`
			UPDATE auth_sessions
			SET revoked_at = NOW(), revoked_reason = 'logout_other_devices'
			WHERE user_id = $1 AND revoked_at IS NULL AND refresh_token_hash <> $2
		`, user.UserID, currentTokenHash)
	} else {
		result, err = database.DB.Exec(`
			UPDATE auth_sessions
			SET revoked_at = NOW(), revoked_reason = 'logout_other_devices'
			WHERE user_id = $1 AND revoked_at IS NULL
		`, user.UserID)
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to logout other devices")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Other devices logged out",
		"data": map[string]interface{}{
			"revoked_sessions": rowsAffected,
		},
	})
}
