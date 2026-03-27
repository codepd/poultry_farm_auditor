package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"poultry-farm-api/config"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	refreshTokenCookieName            = "refresh_token"
	deviceIDHeaderName                = "X-Device-Id"
	maxDeviceIDLen                    = 128
	defaultNoRememberRefreshHours int = 12
	defaultRememberRefreshDays    int = 30
)

// DeviceIDFromRequest returns a normalized client device id from X-Device-Id, or empty if absent/invalid.
func DeviceIDFromRequest(r *http.Request) string {
	s := strings.TrimSpace(r.Header.Get(deviceIDHeaderName))
	if s == "" {
		return ""
	}
	if len(s) > maxDeviceIDLen {
		s = s[:maxDeviceIDLen]
	}
	return s
}

func issueAccessToken(cfg *config.Config, userID int, email string, tenantID uuid.UUID, role string) (string, error) {
	claims := &middleware.Claims{
		UserID:   userID,
		Email:    email,
		TenantID: tenantID.String(),
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func createRefreshSession(userID int, tenantID uuid.UUID, rememberMe bool, expiresAt time.Time, userAgent, ipAddress, deviceID string) (string, error) {
	rawToken, err := generateRandomToken(32)
	if err != nil {
		return "", err
	}
	tokenHash := hashToken(rawToken)

	var deviceIDArg interface{}
	if strings.TrimSpace(deviceID) != "" {
		deviceIDArg = deviceID
	}

	_, err = database.DB.Exec(`
		INSERT INTO auth_sessions (
			user_id, tenant_id, refresh_token_hash, remember_me, user_agent, ip_address, device_id, expires_at, last_used_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, userID, tenantID, tokenHash, rememberMe, userAgent, ipAddress, deviceIDArg, expiresAt.UTC())
	if err != nil {
		return "", err
	}

	return rawToken, nil
}

func rotateRefreshSession(oldTokenHash string, newTokenHash string, expiresAt time.Time) error {
	_, err := database.DB.Exec(`
		UPDATE auth_sessions
		SET refresh_token_hash = $1, expires_at = $2, last_used_at = NOW()
		WHERE refresh_token_hash = $3 AND revoked_at IS NULL
	`, newTokenHash, expiresAt.UTC(), oldTokenHash)
	return err
}

func getTenantRefreshDurations(tenantID uuid.UUID) (time.Duration, time.Duration) {
	noRememberHours := defaultNoRememberRefreshHours
	rememberDays := defaultRememberRefreshDays

	var noRemember sql.NullInt64
	var remember sql.NullInt64
	err := database.DB.QueryRow(`
		SELECT refresh_ttl_without_remember_hours, refresh_ttl_with_remember_days
		FROM tenants
		WHERE id = $1
	`, tenantID).Scan(&noRemember, &remember)
	if err == nil {
		if noRemember.Valid && noRemember.Int64 > 0 {
			noRememberHours = int(noRemember.Int64)
		}
		if remember.Valid && remember.Int64 > 0 {
			rememberDays = int(remember.Int64)
		}
	}

	return time.Duration(noRememberHours) * time.Hour, time.Duration(rememberDays) * 24 * time.Hour
}

func setRefreshTokenCookie(w http.ResponseWriter, cfg *config.Config, token string, expiresAt time.Time) {
	cookie := &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     "/api",
		HttpOnly: true,
		Secure:   cfg.RefreshCookieSecure,
		SameSite: parseSameSite(cfg.RefreshCookieSameSite),
		Expires:  expiresAt.UTC(),
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	}
	if cfg.RefreshCookieDomain != "" {
		cookie.Domain = cfg.RefreshCookieDomain
	}
	http.SetCookie(w, cookie)
}

func clearRefreshTokenCookie(w http.ResponseWriter, cfg *config.Config) {
	cookie := &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/api",
		HttpOnly: true,
		Secure:   cfg.RefreshCookieSecure,
		SameSite: parseSameSite(cfg.RefreshCookieSameSite),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
	if cfg.RefreshCookieDomain != "" {
		cookie.Domain = cfg.RefreshCookieDomain
	}
	http.SetCookie(w, cookie)
}

func parseSameSite(v string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return http.SameSiteLaxMode
	}
}

func generateRandomToken(numBytes int) (string, error) {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
