package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type HenBatch struct {
	ID           int            `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	BatchName    string         `json:"batch_name"`
	InitialCount int            `json:"initial_count"`
	CurrentCount int            `json:"current_count"`
	AgeWeeks     int            `json:"age_weeks"`
	AgeDays      int            `json:"age_days"`
	DateAdded    time.Time      `json:"date_added"`
	Notes        sql.NullString `json:"notes,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// AgeString returns age in format "16W 2D"
func (h *HenBatch) AgeString() string {
	if h.AgeDays > 0 {
		return fmt.Sprintf("%dW %dD", h.AgeWeeks, h.AgeDays)
	}
	return fmt.Sprintf("%dW", h.AgeWeeks)
}

type HenMortality struct {
	ID               int            `json:"id"`
	BatchID          int            `json:"batch_id"`
	MortalityDate    time.Time      `json:"mortality_date"`
	Count            int            `json:"count"`
	Reason           sql.NullString `json:"reason,omitempty"`
	Notes            sql.NullString `json:"notes,omitempty"`
	RecordedByUserID *int           `json:"recorded_by_user_id,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}
