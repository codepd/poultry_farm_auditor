package handlers

import (
	"database/sql"
	"net/http"
	"poultry-farm-api/config"
	"poultry-farm-api/database"
	"strings"
	"time"

	"github.com/google/uuid"
)

func RefreshToken(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(refreshTokenCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			respondWithError(w, http.StatusUnauthorized, "Refresh token missing")
			return
		}

		oldTokenHash := hashToken(cookie.Value)
		newRawToken, err := generateRandomToken(32)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to refresh session")
			return
		}
		newTokenHash := hashToken(newRawToken)

		tx, err := database.DB.Begin()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to refresh session")
			return
		}
		defer tx.Rollback()

		var userID int
		var tenantID uuid.UUID
		var rememberMe bool
		var revokedAt sql.NullTime
		var expiresAt time.Time
		err = tx.QueryRow(`
			SELECT user_id, tenant_id, remember_me, revoked_at, expires_at
			FROM auth_sessions
			WHERE refresh_token_hash = $1
			FOR UPDATE
		`, oldTokenHash).Scan(&userID, &tenantID, &rememberMe, &revokedAt, &expiresAt)
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
			return
		}
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to refresh session")
			return
		}
		if revokedAt.Valid || time.Now().UTC().After(expiresAt.UTC()) {
			_, _ = tx.Exec(`
				UPDATE auth_sessions
				SET revoked_at = COALESCE(revoked_at, NOW()), revoked_reason = COALESCE(revoked_reason, 'expired_or_revoked')
				WHERE refresh_token_hash = $1
			`, oldTokenHash)
			respondWithError(w, http.StatusUnauthorized, "Session expired. Please login again.")
			return
		}

		var email string
		var isActive bool
		err = tx.QueryRow(`
			SELECT COALESCE(email, ''), is_active
			FROM users
			WHERE id = $1
		`, userID).Scan(&email, &isActive)
		if err == sql.ErrNoRows || !isActive {
			respondWithError(w, http.StatusUnauthorized, "User account is inactive")
			return
		}
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to refresh session")
			return
		}

		var role string
		err = tx.QueryRow(`
			SELECT role
			FROM tenant_users
			WHERE user_id = $1 AND tenant_id = $2
		`, userID, tenantID).Scan(&role)
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Tenant access no longer available")
			return
		}
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to refresh session")
			return
		}

		noRememberTTL, rememberTTL := getTenantRefreshDurations(tenantID)
		refreshTTL := noRememberTTL
		if rememberMe {
			refreshTTL = rememberTTL
		}
		newExpiresAt := time.Now().UTC().Add(refreshTTL)

		_, err = tx.Exec(`
			UPDATE auth_sessions
			SET refresh_token_hash = $1,
			    expires_at = $2,
			    last_used_at = NOW()
			WHERE refresh_token_hash = $3 AND revoked_at IS NULL
		`, newTokenHash, newExpiresAt, oldTokenHash)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to refresh session")
			return
		}

		if err := tx.Commit(); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to refresh session")
			return
		}

		accessToken, err := issueAccessToken(cfg, userID, email, tenantID, role)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		setRefreshTokenCookie(w, cfg, newRawToken, newExpiresAt)
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"token": accessToken,
			},
		})
	}
}

func Logout(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(refreshTokenCookieName)
		if err == nil && strings.TrimSpace(cookie.Value) != "" {
			tokenHash := hashToken(cookie.Value)
			_, _ = database.DB.Exec(`
				UPDATE auth_sessions
				SET revoked_at = NOW(), revoked_reason = 'logout'
				WHERE refresh_token_hash = $1 AND revoked_at IS NULL
			`, tokenHash)
		}
		clearRefreshTokenCookie(w, cfg)
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Logged out",
		})
	}
}
