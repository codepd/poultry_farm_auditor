package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
		SELECT u.id, u.email, u.full_name, u.is_active, u.created_at, u.updated_at,
		       tu.role, tu.is_owner
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
			&u.ID, &u.Email, &u.FullName, &u.IsActive,
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
	Email  string    `json:"email"`
	Role   string    `json:"role"`
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

	// Validate role
	validRoles := map[string]bool{
		"ADMIN": true, "OWNER": true, "CO_OWNER": true,
		"OTHER_USER": true, "AUDITOR": true,
	}
	if !validRoles[req.Role] {
		respondWithError(w, http.StatusBadRequest, "Invalid role")
		return
	}

	// Check if user already exists
	var existingUserID int
	err = database.DB.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&existingUserID)

	var userID int
	if err == sql.ErrNoRows {
		// Create new user (inactive until they accept invite)
		err = database.DB.QueryRow(`
			INSERT INTO users (email, is_active)
			VALUES ($1, FALSE)
			RETURNING id
		`, req.Email).Scan(&userID)

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	} else {
		userID = existingUserID
	}

	// Check if user already has access to this tenant
	var hasAccess bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM tenant_users 
			WHERE user_id = $1 AND tenant_id = $2
		)
	`, userID, tenantID).Scan(&hasAccess)

	if hasAccess {
		respondWithError(w, http.StatusBadRequest, "User already has access to this tenant")
		return
	}

	// Generate invitation token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	// Create invitation
	var inviteID int
	err = database.DB.QueryRow(`
		INSERT INTO invitations (
			tenant_id, invited_by_user_id, email, role, token, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, tenantID, user.UserID, req.Email, req.Role, token, expiresAt).Scan(&inviteID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create invitation")
		return
	}

	// Generate invitation link
	// In production, this would be the actual frontend URL
	// For local development, use localhost
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:4300"
	}
	invitationLink := fmt.Sprintf("%s/accept-invite?token=%s", frontendURL, token)

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Invitation sent",
		"data": map[string]interface{}{
			"invitation_id":  inviteID,
			"email":          req.Email,
			"token":          token,
			"expires_at":     expiresAt,
			"invitation_link": invitationLink,
		},
	})
}

type AcceptInviteRequest struct {
	Token       string `json:"token"`
	Password    string `json:"password"`
	FullName    string `json:"full_name"`
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
		SELECT id, tenant_id, invited_by_user_id, email, role, token, expires_at, accepted_at
		FROM invitations
		WHERE token = $1
	`, req.Token).Scan(
		&invite.ID, &invite.TenantID, &invite.InvitedByUserID,
		&invite.Email, &invite.Role, &invite.Token, &invite.ExpiresAt, &invite.AcceptedAt,
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

	// Get or create user
	var userID int
	err = database.DB.QueryRow("SELECT id FROM users WHERE email = $1", invite.Email).Scan(&userID)

	if err == sql.ErrNoRows {
		// Create user
		err = database.DB.QueryRow(`
			INSERT INTO users (email, password_hash, full_name, is_active)
			VALUES ($1, $2, $3, TRUE)
			RETURNING id
		`, invite.Email, passwordHash, req.FullName).Scan(&userID)
	} else if err == nil {
		// Update existing user
		_, err = database.DB.Exec(`
			UPDATE users
			SET password_hash = $1, full_name = $2, is_active = TRUE, updated_at = CURRENT_TIMESTAMP
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
		SELECT id, tenant_id, invited_by_user_id, email, role, token,
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
			&inv.ID, &inv.TenantID, &inv.InvitedByUserID, &inv.Email,
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

