package utils

import (
	"database/sql"
	"poultry-farm-api/database"
	"time"

	"github.com/google/uuid"
)

// GetTenantTimezoneLocation gets the timezone location for a tenant
// Returns the location and an error if the tenant doesn't exist or timezone is invalid
func GetTenantTimezoneLocation(tenantID uuid.UUID) (*time.Location, error) {
	var timezone string
	err := database.DB.QueryRow(`
		SELECT timezone FROM tenants WHERE id = $1
	`, tenantID).Scan(&timezone)

	if err == sql.ErrNoRows {
		// Default to UTC if tenant not found
		return time.UTC, nil
	}
	if err != nil {
		return nil, err
	}

	// Default to Asia/Kolkata if timezone is empty
	if timezone == "" {
		timezone = "Asia/Kolkata"
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		// If timezone is invalid, default to UTC
		return time.UTC, nil
	}

	return location, nil
}

// ParseDateInTimezone parses a date string in the given timezone
// For date-only strings like "2025-11-01", it interprets them as midnight in the timezone
func ParseDateInTimezone(dateStr string, location *time.Location) (time.Time, error) {
	// Try parsing as date-only format first
	t, err := time.ParseInLocation("2006-01-02", dateStr, location)
	if err == nil {
		return t, nil
	}

	// Try RFC3339 format
	t, err = time.Parse(time.RFC3339, dateStr)
	if err == nil {
		// Convert to the target timezone
		return t.In(location), nil
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		t, err = time.ParseInLocation(format, dateStr, location)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, err
}
