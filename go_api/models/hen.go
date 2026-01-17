package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NullString is a custom type that properly marshals sql.NullString to JSON
type NullString struct {
	sql.NullString
}

// MarshalJSON implements json.Marshaler for NullString
func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

// UnmarshalJSON implements json.Unmarshaler for NullString
func (ns *NullString) UnmarshalJSON(data []byte) error {
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s != nil {
		ns.Valid = true
		ns.String = *s
	} else {
		ns.Valid = false
		ns.String = ""
	}
	return nil
}

type HenBatch struct {
	ID           int        `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	BatchName    string     `json:"batch_name"`
	InitialCount int        `json:"initial_count"`
	CurrentCount int        `json:"current_count"`
	AgeWeeks     int        `json:"age_weeks"`
	AgeDays      int        `json:"age_days"`
	DateAdded    time.Time  `json:"date_added"`
	Notes        NullString `json:"notes,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// AgeString returns age in format "16W 2D"
func (h *HenBatch) AgeString() string {
	if h.AgeDays > 0 {
		return fmt.Sprintf("%dW %dD", h.AgeWeeks, h.AgeDays)
	}
	return fmt.Sprintf("%dW", h.AgeWeeks)
}

type HenMortality struct {
	ID               int        `json:"id"`
	BatchID          int        `json:"batch_id"`
	MortalityDate    time.Time  `json:"mortality_date"`
	Count            int        `json:"count"`
	Reason           NullString `json:"reason,omitempty"`
	Notes            NullString `json:"notes,omitempty"`
	RecordedByUserID *int       `json:"recorded_by_user_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}
